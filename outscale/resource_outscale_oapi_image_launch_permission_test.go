package outscale

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/terraform/helper/acctest"
	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-outscale/osc/oapi"
)

func TestAccOutscaleOAPIImageLaunchPermission_Basic(t *testing.T) {
	o := os.Getenv("OUTSCALE_OAPI")

	oapi, err := strconv.ParseBool(o)
	if err != nil {
		oapi = false
	}

	if !oapi {
		t.Skip()
	}

	imageID := ""
	accountID := "520679080430"

	rInt := acctest.RandInt()

	r.Test(t, r.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []r.TestStep{
			// Scaffold everything
			r.TestStep{
				Config: testAccOutscaleOAPIImageLaunchPermissionConfig(rInt),
				Check: r.ComposeTestCheckFunc(
					testCheckResourceOAPILPIGetAttr("outscale_image.outscale_image", "id", &imageID),
				),
			},
			// Drop just launch permission to test destruction
			r.TestStep{
				Config: testAccOutscaleOAPIImageLaunchPermissionConfig(rInt),
				Check: r.ComposeTestCheckFunc(
					testAccOutscaleOAPIImageLaunchPermissionDestroyed(accountID, &imageID),
				),
			},
			// Re-add everything so we can test when AMI disappears
			r.TestStep{
				Config: testAccOutscaleOAPIImageLaunchPermissionConfig(rInt),
				Check: r.ComposeTestCheckFunc(
					testCheckResourceOAPILPIGetAttr("outscale_image.outscale_image", "id", &imageID),
				),
			},
			// Here we delete the AMI to verify the follow-on refresh after this step
			// should not error.
			r.TestStep{
				Config: testAccOutscaleOAPIImageLaunchPermissionConfig(rInt),
				Check: r.ComposeTestCheckFunc(
					testAccOutscaleOAPIImageDisappears(&imageID),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testCheckResourceOAPILPIGetAttr(name, key string, value *string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("No primary instance: %s", name)
		}

		*value = is.Attributes[key]
		return nil
	}
}

func testAccOutscaleOAPIImageLaunchPermissionExists(accountID string, imageID *string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*OutscaleClient).OAPI
		if has, err := hasOAPILaunchPermission(conn, *imageID); err != nil {
			return err
		} else if !has {
			return fmt.Errorf("launch permission does not exist for '%s' on '%s'", accountID, *imageID)
		}
		return nil
	}
}

func testAccOutscaleOAPIImageLaunchPermissionDestroyed(accountID string, imageID *string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*OutscaleClient).OAPI
		if has, err := hasOAPILaunchPermission(conn, *imageID); err != nil {
			return err
		} else if has {
			return fmt.Errorf("launch permission still exists for '%s' on '%s'", accountID, *imageID)
		}
		return nil
	}
}

// testAccOutscaleOAPIImageDisappears is technically a "test check function" but really it
// exists to perform a side effect of deleting an AMI out from under a resource
// so we can test that Terraform will react properly
func testAccOutscaleOAPIImageDisappears(imageID *string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*OutscaleClient).OAPI
		req := &oapi.DeleteImageRequest{
			ImageId: aws.StringValue(imageID),
		}

		err := r.Retry(5*time.Minute, func() *r.RetryError {
			var err error
			_, err = conn.POST_DeleteImage(*req)
			if err != nil {
				if strings.Contains(err.Error(), "RequestLimitExceeded:") {
					return r.RetryableError(err)
				}
				return r.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return err
		}

		return resourceOutscaleOAPIImageWaitForDestroy(*imageID, conn)
	}
}

func testAccOutscaleOAPIImageLaunchPermissionConfig(r int) string {
	return fmt.Sprintf(`
resource "outscale_vm" "outscale_instance" {
    count = 1
    image_id           = "ami-880caa66"
    type               = "t2.micro"
}

resource "outscale_image" "outscale_image" {
    name        = "terraform test-123-%d"
    vm_id = "${outscale_vm.outscale_instance.id}"
	no_reboot   = "true"
}

resource "outscale_image_launch_permission" "outscale_image_launch_permission" {
    image_id    = "${outscale_image.outscale_image.image_id}"
    permission_additions {
        account_id = "520679080430"
	}
}
`, r)
}
