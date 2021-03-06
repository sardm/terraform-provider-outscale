package outscale

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOutscaleVMSDataSource_basic(t *testing.T) {
	o := os.Getenv("OUTSCALE_OAPI")

	oapi, err := strconv.ParseBool(o)
	if err != nil {
		oapi = false
	}

	if oapi {
		t.Skip()
	}
	rInt := acctest.RandIntRange(200, 5000)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccVMSDataSourceConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.outscale_vms.basic_web", "instance_id.#", "2"),
					resource.TestCheckResourceAttr(
						"data.outscale_vms.basic_web", "reservation_set.#", "2"),
					resource.TestCheckResourceAttr(
						"data.outscale_vms.basic_web", "reservation_set.0.instances_set.0.group_set.#", "1"),
					resource.TestCheckResourceAttr(
						"data.outscale_vms.basic_web", "reservation_set.0.instances_set.0.tag_set.#", "1"),
				),
			},
		},
	})
}

// Lookup based on InstanceID
func testAccVMSDataSourceConfig(rInt int) string {
	return fmt.Sprintf(`
resource "outscale_keypair" "a_key_pair" {
	key_name   = "terraform-key-%d"
}

resource "outscale_firewall_rules_set" "web" {
  group_name = "terraform_acceptance_test_example_1-%d"
  group_description = "Used in the terraform acceptance tests"
}

resource "outscale_vm" "basic" {
	image_id      = "ami-880caa66"
	instance_type = "t2.micro"
	security_group = ["${outscale_firewall_rules_set.web.id}"]
	key_name = "${outscale_keypair.a_key_pair.key_name}"
	tag = {
		Name = "Hellow"
	}
}
resource "outscale_vm" "basic2" {
	image_id      = "ami-880caa66"
	instance_type = "t2.micro"
	security_group = ["${outscale_firewall_rules_set.web.id}"]
	key_name = "${outscale_keypair.a_key_pair.key_name}"
	tag = {
		Name = "Hellow"
	}
}

data "outscale_vms" "basic_web" {
	instance_id = ["${outscale_vm.basic.id}", "${outscale_vm.basic2.id}"]
}`, rInt, rInt+1)
}
