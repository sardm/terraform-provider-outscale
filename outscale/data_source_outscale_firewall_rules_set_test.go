package outscale

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceOutscaleSecurityGroups_vpc(t *testing.T) {
	o := os.Getenv("OUTSCALE_OAPI")

	oapi, err := strconv.ParseBool(o)
	if err != nil {
		oapi = false
	}

	if oapi {
		t.Skip()
	}
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOutscaleSecurityGroupConfigVPC(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceOutscaleSecurityGroupsCheck("data.outscale_firewall_rules_sets.by_filter"),
				),
			},
		},
	})
}

func testAccDataSourceOutscaleSecurityGroupsCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		_, ok = s.RootModule().Resources["outscale_firewall_rules_sets.outscale_firewall_rules_sets"]
		if !ok {
			return fmt.Errorf("can't find outscale_firewall_rules_sets.outscale_firewall_rules_sets in state")
		}

		return nil
	}
}

func testAccDataSourceOutscaleSecurityGroupConfigVPC(rInt int) string {
	return fmt.Sprintf(`
		resource "outscale_outbound_rule" "outscale_outbound_rule1" {
	ip_permissions = {
		from_port = 22
		to_port = 22
		ip_protocol = "tcp"
		ip_ranges = ["46.231.147.8/32"]
	}

	group_id = "${outscale_firewall_rules_sets.outscale_firewall_rules_sets.id}"
}

resource "outscale_inbound_rule" "outscale_inbound_rule1" {
	ip_permissions = {
		from_port = 22
		to_port = 22
		ip_protocol = "tcp"
		ip_ranges = ["46.231.147.8/32"]
	}

	group_id = "${outscale_firewall_rules_sets.outscale_firewall_rules_sets.id}"
}

resource "outscale_inbound_rule" "outscale_inbound_rule2" {
	 ip_permissions = {
		from_port = 443
		to_port = 443
		ip_protocol = "tcp"
		ip_ranges = ["46.231.147.8/32"]
	}

	group_id = "${outscale_firewall_rules_sets.outscale_firewall_rules_sets.id}"
}

resource "outscale_lin" "vpc" {
			cidr_block = "10.0.0.0/16"
		}

resource "outscale_firewall_rules_sets" "outscale_firewall_rules_sets" {
		group_description = "Used in the terraform acceptance tests"
		group_name = "test-%d"
		vpc_id = "${outscale_lin.vpc.id}"
		tag = {
			Name = "tf-acctest"
			Seed = "%d"
		}
	}
	data "outscale_firewall_rules_sets" "by_filter" {
		filter {
			name = "group-name"
			values = ["${outscale_firewall_rules_sets.outscale_firewall_rules_sets.group_name}"]
		}
	}`, rInt, rInt)
}
