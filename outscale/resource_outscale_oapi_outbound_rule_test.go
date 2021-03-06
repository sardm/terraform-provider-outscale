package outscale

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/terraform-providers/terraform-provider-outscale/osc/oapi"
)

func TestAccOutscaleOAPIOutboundRule(t *testing.T) {
	o := os.Getenv("OUTSCALE_OAPI")

	oapiFlag, err := strconv.ParseBool(o)
	if err != nil {
		oapiFlag = false
	}

	if !oapiFlag {
		t.Skip()
	}
	var group oapi.SecurityGroup
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOutscaleOAPISecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOutscaleOAPISecurityGroupRuleEgressConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOutscaleOAPIRuleExists("outscale_firewall_rules_set.web", &group),
					testAccCheckOutscaleOAPIRuleAttributes("outscale_outbound_rule.egress_1", &group, nil, "egress"),
				),
			},
		},
	})
}

func testAccOutscaleOAPISecurityGroupRuleEgressConfig(rInt int) string {
	return fmt.Sprintf(`
	#resource "outscale_firewall_rules_set" "web" {
	#	firewall_rules_set_name = "terraform_test_%d"
	#	description = "Used in the terraform acceptance tests"
	#	lin_id = "vpc-e9d09d63"
	#	tag = {
	#					Name = "tf-acc-test"
	#	}
	#}
	resource "outscale_outbound_rule" "egress_1" {
			inbound_rule = {
				ip_protocol = "tcp"
				from_port_range = 80
				to_port_range = 8000
				ip_ranges = ["10.0.0.0/8"]
		}
		#firewall_rules_set_id = "${outscale_firewall_rules_set.web.id}"
		firewall_rules_set_id = "123"
	}`, rInt)
}
