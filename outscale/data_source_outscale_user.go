package outscale

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/terraform-providers/terraform-provider-outscale/osc/eim"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceOutscaleUser() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceOutscaleUserRead,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"user_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"request_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"user_name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateOutscaleUserName,
			},
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceOutscaleUserRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*OutscaleClient).EIM

	request := &eim.GetUserInput{
		UserName: aws.String(d.Get("user_name").(string)),
	}

	var err error
	var resp *eim.GetUserOutput
	var getResp *eim.GetUserResult
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		resp, err = iamconn.API.GetUser(request)
		if err != nil {
			if strings.Contains(err.Error(), "Throttling:") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		if resp.GetUserResult != nil {
			getResp = resp.GetUserResult
		}
		return nil
	})

	if err != nil {
		if strings.Contains(fmt.Sprint(err), "NoSuchEntity") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM User %s: %s", d.Id(), err)
	}

	d.Set("user_name", aws.StringValue(getResp.User.UserName))
	d.Set("arn", aws.StringValue(getResp.User.Arn))
	d.Set("path", aws.StringValue(getResp.User.Path))
	d.Set("request_id", aws.StringValue(resp.ResponseMetadata.RequestID))
	d.SetId(resource.UniqueId())

	return d.Set("user_id", aws.StringValue(getResp.User.UserId))
}
