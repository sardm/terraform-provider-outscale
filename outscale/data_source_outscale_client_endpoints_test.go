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

func TestAccOutscaleDSCustomerGateways_basic(t *testing.T) {
	o := os.Getenv("OUTSCALE_OAPI")

	oapi, err := strconv.ParseBool(o)
	if err != nil {
		oapi = false
	}

	if oapi {
		t.Skip()
	}

	rBgpAsn := acctest.RandIntRange(64512, 65534)
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "outscale_client_endpoint.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckCustomerGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomerGatewaysDSConfig(rInt, rBgpAsn),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOutscaleCGsDataSourceID("data.outscale_client_endpoints.test"),
					resource.TestCheckResourceAttr("data.outscale_client_endpoints.test", "customer_gateway_set.#", "1"),
				),
			},
		},
	})
}

func testAccCheckOutscaleCGsDataSourceID(n string) resource.TestCheckFunc {
	// Wait for IAM role
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find Customer Gateway data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Customer Gateway data source ID not set")
		}
		return nil
	}
}

func testAccCustomerGatewaysDSConfig(rInt, rBgpAsn int) string {
	return fmt.Sprintf(`
		resource "outscale_client_endpoint" "foo" {
			bgp_asn = %d
			ip_address = "172.0.0.1"
			type = "ipsec.1"
			tag {
				Name = "foo-gateway-%d"
			}
		}

		data "outscale_client_endpoints" "test" {
			customer_gateway_id = ["${outscale_client_endpoint.foo.id}"]
		}
		`, rBgpAsn, rInt)
}
