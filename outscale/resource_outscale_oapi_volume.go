package outscale

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-outscale/osc/oapi"
	"github.com/terraform-providers/terraform-provider-outscale/utils"
)

func resourceOutscaleOAPIVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceOAPIVolumeCreate,
		Read:   resourceOAPIVolumeRead,
		Delete: resourceOAPIVolumeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			// Arguments
			"subregion_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"iops": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"size": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"snapshot_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"volume_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			// Attributes
			"linked_volumes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delete_on_vm_termination": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"device": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"vm_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"state": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"volume_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsListOAPISchema(),
			"volume_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"request_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOAPIVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).OAPI

	request := &oapi.CreateVolumeRequest{
		SubregionName: d.Get("subregion_name").(string),
	}
	if value, ok := d.GetOk("size"); ok {
		request.Size = int64(value.(int))
	}
	if value, ok := d.GetOk("snapshot_id"); ok {
		request.SnapshotId = value.(string)
	}

	var t string
	if value, ok := d.GetOk("volume_type"); ok {
		t = value.(string)
		request.VolumeType = t
	}

	iops := d.Get("iops").(int)
	if t != "io1" && iops > 0 {
		log.Printf("[WARN] IOPs is only valid for storate type io1 for EBS Volumes")
	} else if t == "io1" {
		request.Iops = int64(iops)
	}
	//Missing on Swagger Spec
	// tagsSpec := make([]*oapi.Tags, 0)

	// if v, ok := d.GetOk("tag"); ok {
	// 	tag := tagsFromMap(v.(map[string]interface{}))

	// 	spec := &oapi.TagSpecification{
	// 		ResourceType: aws.String("volume"),
	// 		Tags:         tag,
	// 	}

	// 	tagsSpec = append(tagsSpec, spec)
	// }

	// if len(tagsSpec) > 0 {
	// 	request.TagSpecifications = tagsSpec
	// }

	var result *oapi.CreateVolumeResponse
	var resp *oapi.POST_CreateVolumeResponses
	var err error

	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		resp, err = conn.POST_CreateVolume(*request)
		if err != nil {
			if strings.Contains(err.Error(), "RequestLimitExceeded:") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	var errString string

	if err != nil || resp.OK == nil {
		if err != nil {
			errString = err.Error()
		} else if resp.Code401 != nil {
			errString = fmt.Sprintf("ErrorCode: 401, %s", utils.ToJSONString(resp.Code401))
		} else if resp.Code400 != nil {
			errString = fmt.Sprintf("ErrorCode: 400, %s", utils.ToJSONString(resp.Code400))
		} else if resp.Code500 != nil {
			errString = fmt.Sprintf("ErrorCode: 500, %s", utils.ToJSONString(resp.Code500))
		}

		return fmt.Errorf("Error creating Outscale VM volume: %s", errString)
	}

	result = resp.OK
	log.Println("[DEBUG] Waiting for Volume to become available")

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating"},
		Target:     []string{"available"},
		Refresh:    volumeOAPIStateRefreshFunc(conn, result.Volume.VolumeId),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Volume (%s) to become available: %s", result.Volume.VolumeId, err)
	}

	d.SetId(result.Volume.VolumeId)

	//Missing in swagger spec
	if d.IsNewResource() {
		if err := setOAPITags(conn, d); err != nil {
			return err
		}
		d.SetPartial("tags")
	}

	return resourceOAPIVolumeRead(d, meta)
}

func resourceOAPIVolumeRead(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*OutscaleClient).OAPI

	request := &oapi.ReadVolumesRequest{
		Filters: oapi.FiltersVolume{VolumeIds: []string{d.Id()}},
	}

	var response *oapi.ReadVolumesResponse
	var resp *oapi.POST_ReadVolumesResponses
	var err error

	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		resp, err = conn.POST_ReadVolumes(*request)
		if err != nil {
			if strings.Contains(err.Error(), "RequestLimitExceeded:") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	response = resp.OK

	utils.PrintToJSON(response, "##RESPONSE READ")

	if err != nil {
		if strings.Contains(fmt.Sprint(err), "InvalidVolume.NotFound") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading Outscale volume %s: %s", d.Id(), err)
	}
	d.Set("request_id", response.ResponseContext.RequestId)
	return readOAPIVolume(d, &response.Volumes[0])
}

func resourceOAPIVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).OAPI

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		request := &oapi.DeleteVolumeRequest{
			VolumeId: d.Id(),
		}
		response, err := conn.POST_DeleteVolume(*request)
		if err == nil {
			return nil
		}

		if strings.Contains(fmt.Sprint(err), "VolumeInUse") {
			return resource.RetryableError(fmt.Errorf("Outscale VolumeInUse - trying again while it detaches"))
		}
		fmt.Println(err)
		utils.PrintToJSON(response.OK, "##RESPONSE-DELETE")

		return resource.NonRetryableError(err)
	})

}

func volumeOAPIStateRefreshFunc(conn *oapi.Client, volumeID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.POST_ReadVolumes(oapi.ReadVolumesRequest{
			Filters: oapi.FiltersVolume{
				VolumeIds: []string{volumeID},
			},
		})

		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok {
				log.Printf("Error on Volume State Refresh: message: \"%s\", code:\"%s\"", ec2err.Message(), ec2err.Code())
				resp = nil
				return nil, "", err
			}
			log.Printf("Error on Volume State Refresh: %s", err)
			return nil, "", err
		}

		v := resp.OK.Volumes[0]
		return v, v.State, nil
	}
}

func readOAPIVolume(d *schema.ResourceData, volume *oapi.Volume) error {
	d.SetId(volume.VolumeId)

	d.Set("subregion_name", volume.SubregionName)

	//Commented until backend issues is resolved.
	//d.Set("size", volume.Size)
	d.Set("snapshot_id", volume.SnapshotId)

	if volume.VolumeType != "" {
		d.Set("volume_type", volume.VolumeType)
	} else if vType, ok := d.GetOk("volume_type"); ok {
		volume.VolumeType = vType.(string)
	}

	if volume.VolumeType != "" && volume.VolumeType == "io1" {
		//if volume.Iops != "" {
		d.Set("iops", volume.Iops)
		//}
	}

	d.Set("state", volume.State)
	d.Set("volume_id", volume.VolumeId)

	if volume.LinkedVolumes != nil {
		res := make([]map[string]interface{}, len(volume.LinkedVolumes))
		for k, g := range volume.LinkedVolumes {
			r := make(map[string]interface{})
			//if g.DeleteOnVmDeletion != "" {
			r["delete_on_vm_termination"] = g.DeleteOnVmDeletion
			//}
			if g.DeviceName != "" {
				r["device"] = g.DeviceName
			}
			if g.VmId != "" {
				r["vm_id"] = g.VmId
			}
			if g.State != "" {
				r["state"] = g.State
			}
			if g.VolumeId != "" {
				r["volume_id"] = g.VolumeId
			}

			res[k] = r

		}

		if err := d.Set("linked_volumes", res); err != nil {
			return err
		}
	} else {
		if err := d.Set("linked_volumes", []map[string]interface{}{
			map[string]interface{}{
				"delete_on_vm_termination": false,
				"device":                   "none",
				"vm_id":                    "none",
				"state":                    "none",
				"volume_id":                "none",
			},
		}); err != nil {
			return err
		}
	}
	if volume.Tags != nil {
		if err := d.Set("tags", tagsOAPIToMap(volume.Tags)); err != nil {
			return err
		}
	} else {
		if err := d.Set("tags", []map[string]string{
			map[string]string{
				"key":   "",
				"value": "",
			},
		}); err != nil {
			return err
		}
	}

	return nil
}
