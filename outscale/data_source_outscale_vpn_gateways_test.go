package outscale

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceOutscaleVpnGateways_unattached(t *testing.T) {
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceOutscaleVpnGatewaysUnattachedConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.outscale_vpn_gateways.test_by_id", "vpn_gateway_set.#", "1"),
				),
			},
		},
	})
}

func testAccDataSourceOutscaleVpnGatewaysUnattachedConfig(rInt int) string {
	return fmt.Sprintf(`
resource "outscale_vpn_gateway" "unattached" {
    tag {
		Name = "terraform-testacc-vpn-gateway-data-source-unattached-%d"
      	ABC  = "testacc-%d"
		XYZ  = "testacc-%d"
    }
}

data "outscale_vpn_gateways" "test_by_id" {
	vpn_gateway_id = ["${outscale_vpn_gateway.unattached.id}"]
}

`, rInt, rInt+1, rInt-1)
}
