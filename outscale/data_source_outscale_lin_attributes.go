package outscale

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-outscale/osc/fcu"
)

func dataSourceOutscaleVpcAttr() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceOutscaleVpcAttrRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),
			"enable_dns_hostnames": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"enable_dns_support": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"attribute": {
				Type:     schema.TypeString,
				Required: true,
			},
			"request_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceOutscaleVpcAttrRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).FCU

	req := &fcu.DescribeVpcAttributeInput{}

	if id := d.Get("vpc_id"); id != "" {
		req.VpcId = aws.String(id.(string))
	} else {
		return fmt.Errorf("Please provide a vpc_id to be able to make the request")
	}

	if id := d.Get("attribute"); id != "" {
		req.Attribute = aws.String(id.(string))
	} else {
		return fmt.Errorf("Please provide an attribute to be able to make the request")
	}

	if v, ok := d.GetOk("filter"); ok {
		req.Filters = buildOutscaleDataSourceFilters(v.(*schema.Set))
	}

	var err error
	var resp *fcu.DescribeVpcAttributeOutput
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		var err error

		resp, err = conn.VM.DescribeVpcAttribute(req)
		if err != nil {
			if strings.Contains(err.Error(), "RequestLimitExceeded:") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	if resp == nil {
		d.SetId("")
		return fmt.Errorf("Lin not found")
	}

	d.SetId(*resp.VpcId)
	d.Set("vpc_id", resp.VpcId)
	if resp.EnableDnsHostnames != nil {
		d.Set("enable_dns_hostnames", *resp.EnableDnsHostnames.Value)
	}
	if resp.EnableDnsSupport != nil {
		d.Set("enable_dns_support", *resp.EnableDnsSupport.Value)
	}

	d.Set("request_id", resp.RequestId)

	return nil
}
