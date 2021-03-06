package outscale

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/terraform-providers/terraform-provider-outscale/osc/oapi"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOutscaleOAPIPublicIP_basic(t *testing.T) {
	o := os.Getenv("OUTSCALE_OAPI")

	isOAPI, err := strconv.ParseBool(o)
	if err != nil {
		isOAPI = false
	}

	if !isOAPI {
		t.Skip()
	}

	var conf oapi.PublicIp

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "outscale_public_ip.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOutscaleOAPIPublicIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccOutscaleOAPIPublicIPConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOutscaleOAPIPublicIPExists("outscale_public_ip.bar", &conf),
					testAccCheckOutscaleOAPIPublicIPAttributes(&conf),
				),
			},
		},
	})
}

func TestAccOutscaleOAPIPublicIP_instance(t *testing.T) {
	o := os.Getenv("OUTSCALE_OAPI")

	isOAPI, err := strconv.ParseBool(o)
	if err != nil {
		isOAPI = false
	}

	if !isOAPI {
		t.Skip()
	}
	var conf oapi.PublicIp

	//rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "outscale_public_ip.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOutscaleOAPIPublicIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccOutscaleOAPIPublicIPInstanceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOutscaleOAPIPublicIPExists("outscale_public_ip.bar", &conf),
					testAccCheckOutscaleOAPIPublicIPAttributes(&conf),
				),
			},

			resource.TestStep{
				Config: testAccOutscaleOAPIPublicIPInstanceConfig2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOutscaleOAPIPublicIPExists("outscale_public_ip.bar", &conf),
					testAccCheckOutscaleOAPIPublicIPAttributes(&conf),
				),
			},
		},
	})
}

// // This test is an expansion of TestAccOutscalePublicIP_instance, by testing the
// // associated Private PublicIPs of two instances
func TestAccOutscaleOAPIPublicIP_associated_user_private_ip(t *testing.T) {
	o := os.Getenv("OUTSCALE_OAPI")

	isOAPI, err := strconv.ParseBool(o)
	if err != nil {
		isOAPI = false
	}

	if !isOAPI {
		t.Skip()
	}
	var one oapi.PublicIp

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "outscale_public_ip.bar",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckOutscaleOAPIPublicIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccOutscaleOAPIPublicIPInstanceConfigAssociated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOutscaleOAPIPublicIPExists("outscale_public_ip.bar", &one),
					testAccCheckOutscaleOAPIPublicIPAttributes(&one),
				),
			},

			resource.TestStep{
				Config: testAccOutscaleOAPIPublicIPInstanceConfigAssociatedSwitch,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOutscaleOAPIPublicIPExists("outscale_public_ip.bar", &one),
					testAccCheckOutscaleOAPIPublicIPAttributes(&one),
				),
			},
		},
	})
}

func testAccCheckOutscaleOAPIPublicIPDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*OutscaleClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "outscale_public_ip" {
			continue
		}
		//Missing on Swagger Spec
		// if strings.Contains(rs.Primary.ID, "reservation") {
		// 	req := oapi.ReadPublicIpsRequest{
		// 		Filters: oapi.FiltersPublicIpcIp{
		// 			ReservationIds: []string{rs.Primary.ID},
		// 		},
		// 	}

		// 	var describe *oapi.ReadPublicIpsResponse
		// 	err := resource.Retry(60*time.Second, func() *resource.RetryError {
		// 		var err error
		// 		resp, err := conn.OAPI.POST_ReadPublicIps(req)
		// 		describe = resp.OK
		// 		return resource.RetryableError(err)
		// 	})

		// 	if err != nil {
		// 		// Verify the error is what we want
		// 		if e := fmt.Sprint(err); strings.Contains(e, "InvalidAllocationID.NotFound") || strings.Contains(e, "InvalidPublicIps.NotFound") {
		// 			return nil
		// 		}

		// 		return err
		// 	}

		// 	if len(describe.PublicIps) > 0 {
		// 		return fmt.Errorf("still exists")
		// 	}
		// } else {
		req := oapi.ReadPublicIpsRequest{
			Filters: oapi.FiltersPublicIp{
				PublicIps: []string{rs.Primary.ID},
			},
		}

		var describe *oapi.ReadPublicIpsResponse
		err := resource.Retry(60*time.Second, func() *resource.RetryError {
			var err error
			resp, err := conn.OAPI.POST_ReadPublicIps(req)
			describe = resp.OK
			return resource.RetryableError(err)
		})

		if err != nil {
			// Verify the error is what we want
			if e := fmt.Sprint(err); strings.Contains(e, "InvalidAllocationID.NotFound") || strings.Contains(e, "InvalidPublicIps.NotFound") {
				return nil
			}

			return err
		}

		if len(describe.PublicIps) > 0 {
			return fmt.Errorf("still exists")
		}
		//}
	}

	return nil
}

func testAccCheckOutscaleOAPIPublicIPAttributes(conf *oapi.PublicIp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if conf.PublicIp == "" {
			return fmt.Errorf("empty public_ip")
		}

		return nil
	}
}

func testAccCheckOutscaleOAPIPublicIPExists(n string, res *oapi.PublicIp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No PublicIP ID is set")
		}

		conn := testAccProvider.Meta().(*OutscaleClient)

		//Missing on Swagger Spec
		// if strings.Contains(rs.Primary.ID, "link") {
		// 	req := oapi.ReadPublicIpsRequest{
		// 		Filters: oapi.FiltersPublicIp{
		// 			ReservationIds: []string{rs.Primary.ID},
		// 		},
		// 	}
		// 	describe, err := conn.OAPI.POST_ReadPublicIps(req)

		// 	if err != nil {
		// 		return err
		// 	}

		// 	if len(describe.OK.PublicIps) != 1 ||
		// 		describe.OK.PublicIps[0].ReservationId != rs.Primary.ID {
		// 		return fmt.Errorf("PublicIP not found")
		// 	}
		// 	*res = describe.OK.PublicIps[0]

		// } else {
		req := oapi.ReadPublicIpsRequest{
			Filters: oapi.FiltersPublicIp{
				PublicIps: []string{rs.Primary.ID},
			},
		}

		var describe *oapi.ReadPublicIpsResponse
		err := resource.Retry(120*time.Second, func() *resource.RetryError {
			var err error
			resp, err := conn.OAPI.POST_ReadPublicIps(req)

			if err != nil {
				if e := fmt.Sprint(err); strings.Contains(e, "InvalidAllocationID.NotFound") || strings.Contains(e, "InvalidPublicIps.NotFound") {
					return resource.RetryableError(err)
				}

				return resource.NonRetryableError(err)
			}

			describe = resp.OK

			return nil
		})

		if err != nil {
			if e := fmt.Sprint(err); strings.Contains(e, "InvalidAllocationID.NotFound") || strings.Contains(e, "InvalidPublicIps.NotFound") {
				return nil
			}

			return err
		}

		if err != nil {

			// Verify the error is what we want
			if e := fmt.Sprint(err); strings.Contains(e, "InvalidAllocationID.NotFound") || strings.Contains(e, "InvalidPublicIps.NotFound") {
				return nil
			}

			return err
		}

		if len(describe.PublicIps) != 1 ||
			describe.PublicIps[0].PublicIp != rs.Primary.ID {
			return fmt.Errorf("PublicIP not found")
		}
		*res = describe.PublicIps[0]
		//}

		return nil
	}
}

const testAccOutscaleOAPIPublicIPConfig = `
resource "outscale_public_ip" "bar" {}
`

const testAccOutscaleOAPIPublicIPInstanceConfig = `
resource "outscale_vm" "basic" {
	image_id = "ami-8a6a0120"
	instance_type = "t2.micro"
	key_name = "terraform-basic"
}
resource "outscale_public_ip" "bar" {}
`

const testAccOutscaleOAPIPublicIPInstanceConfig2 = `
resource "outscale_vm" "basic" {
	image_id = "ami-8a6a0120"
	instance_type = "t2.micro"
	key_name = "terraform-basic"
}
resource "outscale_public_ip" "bar" {}
`

const testAccOutscaleOAPIPublicIPInstanceConfigAssociated = `
resource "outscale_vm" "foo" {
  image_id = "ami-8a6a0120"
	instance_type = "t2.micro"
	key_name = "terraform-basic"
  private_ip_address = "10.0.0.12"
  subnet_id  = "subnet-861fbecc"
}
resource "outscale_vm" "bar" {
  image_id = "ami-8a6a0120"
	instance_type = "t2.micro"
	key_name = "terraform-basic"
  private_ip_address = "10.0.0.19"
  subnet_id  = "subnet-861fbecc"
}
resource "outscale_public_ip" "bar" {}
`

const testAccOutscaleOAPIPublicIPInstanceConfigAssociatedSwitch = `
resource "outscale_vm" "foo" {
 image_id = "ami-8a6a0120"
	instance_type = "t2.micro"
	key_name = "terraform-basic"
  private_ip_address = "10.0.0.12"
  subnet_id  = "subnet-861fbecc"
}
resource "outscale_vm" "bar" {
  image_id = "ami-8a6a0120"
	instance_type = "t2.micro"
	key_name = "terraform-basic"
  private_ip_address = "10.0.0.19"
  subnet_id  = "subnet-861fbecc"
}
resource "outscale_public_ip" "bar" {}
`
