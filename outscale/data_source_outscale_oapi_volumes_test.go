package outscale

import (
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOutscaleOAPIVolumesDataSource_multipleFilters(t *testing.T) {
	o := os.Getenv("OUTSCALE_OAPI")

	oapi, err := strconv.ParseBool(o)
	if err != nil {
		oapi = false
	}

	if !oapi {
		t.Skip()
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckOutscaleOAPIVolumeDataSourceConfigWithMultipleFilters,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOutscaleOAPIVolumeDataSourceID("data.outscale_volumes.ebs_volume"),
					testAccCheckState("data.outscale_volumes.ebs_volume"),
					resource.TestCheckResourceAttr("data.outscale_volumes.ebs_volume", "volumes.0.size", "10"),
					resource.TestCheckResourceAttr("data.outscale_volumes.ebs_volume", "volumes.0.volume_type", "gp2"),
				),
			},
		},
	})
}

func TestAccOutscaleOAPIVolumeDataSource_multipleVIdsFilters(t *testing.T) {
	o := os.Getenv("OUTSCALE_OAPI")

	oapi, err := strconv.ParseBool(o)
	if err != nil {
		oapi = false
	}

	if !oapi {
		t.Skip()
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckOutscaleOAPIVolumesDataSourceConfigWithMultipleVolumeIDsFilter,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOutscaleOAPIVolumeDataSourceID("data.outscale_volumes.outscale_volumes"),
					//Commented until backend issue is solved
					//resource.TestCheckResourceAttr("data.outscale_volumes.outscale_volumes", "volume_set.0.size", "40"),
				),
			},
		},
	})
}

const testAccCheckOutscaleOAPIVolumeDataSourceConfigWithMultipleFilters = `
resource "outscale_volume" "external1" {
	subregion_name = "us-west-1a"
    type = "gp2"
    size = 10
	tags {
		key = "Name" 
		value = "tf-acc-test-ebs-volume-test"
	}
}

data "outscale_volumes" "ebs_volume" {
    filter {
	name = "size"
	values = ["${outscale_volume.external1.size}"]
    }
    filter {
	name = "volume-type"
	values = ["${outscale_volume.external1.type}"]
    }
}
`

const testAccCheckOutscaleOAPIVolumesDataSourceConfigWithMultipleVolumeIDsFilter = `
resource "outscale_volume" "outscale_volume" {
	subregion_name = "us-west-1a"
	size = 40
}

resource "outscale_volume" "outscale_volume2" {
	subregion_name = "us-west-1a"
	size = 40
}

data "outscale_volumes" "outscale_volumes" {
	filter {
		name = "volume-ids"
		values = ["${outscale_volume.outscale_volume.volume_id}", "${outscale_volume.outscale_volume2.volume_id}"]
	}
}
`
