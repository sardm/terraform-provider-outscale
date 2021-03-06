package outscale

import (
	"fmt"
	"strings"
	"time"

	"github.com/terraform-providers/terraform-provider-outscale/osc/lbu"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOutscaleLBUTags() *schema.Resource {
	return &schema.Resource{
		Create: resourceOutscaleLBUTagsCreate,
		Read:   resourceOutscaleLBUTagsRead,
		Delete: resourceOutscaleLBUTagsDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: getLBUTagsSchema(),
	}
}

func resourceOutscaleLBUTagsCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).LBU

	request := &lbu.AddTagsInput{}

	tag, tagsOk := d.GetOk("tags")

	lbus, lbusok := d.GetOk("load_balancer_names")

	if !tagsOk && !lbusok {
		return fmt.Errorf("One tag and resource id, must be assigned")
	}

	request.LoadBalancerNames = expandStringList(lbus.([]interface{}))

	ts := make([]*lbu.Tag, len(lbus.([]interface{})))

	for k, v := range tag.([]interface{}) {
		ta := v.(map[string]interface{})
		t := &lbu.Tag{
			Key:   aws.String(ta["key"].(string)),
			Value: aws.String(ta["value"].(string)),
		}
		ts[k] = t
	}

	request.Tags = ts

	var err error
	err = resource.Retry(60*time.Second, func() *resource.RetryError {
		_, err = conn.API.AddTags(request)
		if err != nil {
			if strings.Contains(fmt.Sprint(err), "Throttling") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	d.SetId(resource.UniqueId())

	return resourceOutscaleLBUTagsRead(d, meta)
}

func resourceOutscaleLBUTagsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).LBU

	lbus := d.Get("load_balancer_names")

	params := &lbu.DescribeTagsInput{
		LoadBalancerNames: expandStringList(lbus.([]interface{})),
	}

	var rs *lbu.DescribeTagsOutput
	var err error

	err = resource.Retry(60*time.Second, func() *resource.RetryError {
		rs, err = conn.API.DescribeTags(params)
		return resource.RetryableError(err)
	})

	if err != nil {
		return err
	}

	d.Set("request_id", rs.ResponseMetadata.RequestID)
	// tg := tagsLBUDescToList(resp.TagDescriptions)
	// err = d.Set("tags", tg)

	return err
}

func resourceOutscaleLBUTagsDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).LBU

	tag := d.Get("tags")

	lbus := d.Get("load_balancer_names")

	request := &lbu.RemoveTagsInput{}

	request.LoadBalancerNames = expandStringList(lbus.([]interface{}))

	ts := make([]*lbu.TagKeyOnly, len(lbus.([]interface{})))

	for k, v := range tag.([]interface{}) {
		ta := v.(map[string]interface{})
		t := &lbu.TagKeyOnly{
			Key: aws.String(ta["key"].(string)),
		}
		ts[k] = t
	}

	request.Tags = ts

	err := resource.Retry(60*time.Second, func() *resource.RetryError {
		_, err := conn.API.RemoveTags(request)
		if err != nil {
			if strings.Contains(fmt.Sprint(err), "Throttling") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func getLBUTagsSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"load_balancer_names": {
			Type:     schema.TypeList,
			Required: true,
			ForceNew: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"tags": {
			Type:     schema.TypeList,
			Required: true,
			ForceNew: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"key": {
						Type:     schema.TypeString,
						Required: true,
					},
					"value": {
						Type:     schema.TypeString,
						Required: true,
					},
				},
			},
		},
		"request_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
	}
}
