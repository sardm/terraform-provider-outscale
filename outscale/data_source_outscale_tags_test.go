package outscale

import (
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOutscaleTagsDataSource_basic(t *testing.T) {
	o := os.Getenv("OUTSCALE_OAPI")

	oapi, err := strconv.ParseBool(o)
	if err != nil {
		oapi = false
	}

	if oapi {
		t.Skip()
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccTagsDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.outscale_tags.web", "tag_set.#", "2"),
				),
			},
		},
	})
}

// Lookup based on InstanceID
const testAccTagsDataSourceConfig = `
resource "outscale_vm" "basic" {
  image_id = "ami-880caa66"
	instance_type = "m1.small"
	tag {
		foo = "bar"
	}
}
resource "outscale_vm" "basic2" {
  image_id = "ami-880caa66"
	instance_type = "m1.small"
	tag {
		foo = "baz"
	}
}

data "outscale_tags" "web" {
	filter {
       name = "resource-type"
       values = ["instance"]
	}

	filter {
       name = "key"
       values = ["foo"]
	}
}`
