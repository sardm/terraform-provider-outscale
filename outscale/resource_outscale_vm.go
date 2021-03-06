package outscale

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-outscale/osc/fcu"
)

func resourceOutscaleVM() *schema.Resource {
	return &schema.Resource{
		Create: resourceVMCreate,
		Read:   resourceVMRead,
		Update: resourceVMUpdate,
		Delete: resourceVMDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: getVMSchema(),
	}
}

func resourceVMCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).FCU

	instanceOpts, err := buildOutscaleVMOpts(d, meta)
	if err != nil {
		return err
	}

	// Build the creation struct
	runOpts := &fcu.RunInstancesInput{
		BlockDeviceMappings:   instanceOpts.BlockDeviceMappings,
		DisableApiTermination: instanceOpts.DisableAPITermination,
		EbsOptimized:          instanceOpts.EBSOptimized,
		DryRun:                instanceOpts.DryRun,
		// Monitoring:            instanceOpts.Monitoring,
		// IamInstanceProfile:    instanceOpts.IAMInstanceProfile,
		ImageId:                           instanceOpts.ImageID,
		InstanceInitiatedShutdownBehavior: instanceOpts.InstanceInitiatedShutdownBehavior,
		InstanceType:                      instanceOpts.InstanceType,
		// Ipv6AddressCount:                  instanceOpts.Ipv6AddressCount,
		// Ipv6Addresses:                     instanceOpts.Ipv6Addresses,
		KeyName:           instanceOpts.KeyName,
		MaxCount:          aws.Int64(int64(1)),
		MinCount:          aws.Int64(int64(1)),
		NetworkInterfaces: instanceOpts.NetworkInterfaces,
		Placement:         instanceOpts.Placement,
		// PrivateIpAddress:                  instanceOpts.PrivateIPAddress,
		// RamdiskId:        instanceOpts.RamdiskId,
		SecurityGroupIds: instanceOpts.SecurityGroupIDs,
		SecurityGroups:   instanceOpts.SecurityGroups,
		SubnetId:         instanceOpts.SubnetID,
		UserData:         instanceOpts.UserData,
	}

	tagsSpec := make([]*fcu.TagSpecification, 0)

	if v, ok := d.GetOk("tag"); ok {
		tag := tagsFromMap(v.(map[string]interface{}))

		spec := &fcu.TagSpecification{
			ResourceType: aws.String("instance"),
			Tags:         tag,
		}

		tagsSpec = append(tagsSpec, spec)
	}

	if len(tagsSpec) > 0 {
		runOpts.TagSpecifications = tagsSpec
	}

	var runResp *fcu.Reservation
	err = resource.Retry(60*time.Second, func() *resource.RetryError {
		var err error
		runResp, err = conn.VM.RunInstance(runOpts)

		return resource.RetryableError(err)
	})

	if err != nil {
		return fmt.Errorf("Error launching source instance 1: %s", err)
	}
	if runResp == nil || len(runResp.Instances) == 0 {
		return errors.New("Error launching source instance 2: no instances returned in response")
	}

	instance := runResp.Instances[0]

	d.SetId(*instance.InstanceId)
	d.Set("instance_id", *instance.InstanceId)

	if d.IsNewResource() {
		if err := setTags(conn, d); err != nil {
			return err
		}
		d.SetPartial("tag_set")
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"running"},
		Refresh:    InstanceStateRefreshFunc(conn, *instance.InstanceId, "terminated"),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to stop: %s", d.Id(), err)
	}

	// Initialize the connection info
	if instance.IpAddress != nil {
		d.SetConnInfo(map[string]string{
			"type": "ssh",
			"host": *instance.IpAddress,
		})
	} else if instance.PrivateIpAddress != nil {
		d.SetConnInfo(map[string]string{
			"type": "ssh",
			"host": *instance.PrivateIpAddress,
		})
	}

	return resourceVMRead(d, meta)
}

func resourceVMRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).FCU

	input := &fcu.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(d.Id())},
	}

	var resp *fcu.DescribeInstancesOutput
	var err error

	err = resource.Retry(30*time.Second, func() *resource.RetryError {
		resp, err = conn.VM.DescribeInstances(input)

		return resource.RetryableError(err)
	})

	if err != nil {
		return fmt.Errorf("Error reading the instance %s", err)
	}

	if err != nil {
		if strings.Contains(fmt.Sprint(err), "InvalidInstanceID.NotFound") {
			d.SetId("")
			return nil
		}

		return err
	}

	if len(resp.Reservations) == 0 {
		d.SetId("")
		return nil
	}

	instance := resp.Reservations[0].Instances[0]

	d.Set("block_device_mapping", getBlockDeviceMapping(instance.BlockDeviceMappings))
	d.Set("client_token", instance.ClientToken)
	d.Set("ebs_optimized", instance.EbsOptimized)
	d.Set("image_id", instance.ImageId)
	d.Set("instance_type", instance.InstanceType)
	d.Set("key_name", instance.KeyName)
	d.Set("network_interface", getNetworkInterfaceSet(instance.NetworkInterfaces))
	d.Set("private_ip", instance.PrivateIpAddress)
	d.Set("ramdisk_id", instance.RamdiskId)
	d.Set("subnet_id", instance.SubnetId)
	d.Set("tag_set", tagsToMap(instance.Tags))
	d.Set("owner_id", resp.Reservations[0].OwnerId)
	d.Set("reservation_id", resp.Reservations[0].ReservationId)

	if err := d.Set("group_set", getGroupSet(resp.Reservations[0].Groups)); err != nil {
		return err
	}

	if err := d.Set("instances_set", flattenedInstanceSet([]*fcu.Instance{instance})); err != nil {
		return err
	}

	if instance.Platform != nil && *instance.Platform == "windows" && len(*instance.KeyName) > 0 {
		var passRes *fcu.GetPasswordDataOutput
		err = resource.Retry(1200*time.Second, func() *resource.RetryError {
			var err error
			passRes, err = conn.VM.GetPasswordData(&fcu.GetPasswordDataInput{
				InstanceId: instance.InstanceId,
			})

			if err != nil {
				if strings.Contains(fmt.Sprint(err), "RequestLimitExceeded") {
					return resource.RetryableError(fmt.Errorf("Got empty password for instance (%s)", d.Id()))
				}
			}

			if passRes.PasswordData == nil || *passRes.PasswordData == "" {
				return resource.RetryableError(fmt.Errorf("Got empty password for instance (%s)", d.Id()))
			}

			return resource.NonRetryableError(err)
		})

		if passRes == nil {
			return fmt.Errorf("Error launching source instance 3: (%s)", d.Id())
		}

		if err != nil {
			return err
		}

		d.Set("password_data", passRes.PasswordData)

		return nil
	}

	return d.Set("request_id", aws.StringValue(resp.RequestId))
}

func resourceVMUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).FCU

	d.Partial(true)

	if d.HasChange("key_name") {
		input := &fcu.ModifyInstanceKeyPairInput{
			InstanceId: aws.String(d.Id()),
			KeyName:    aws.String(d.Get("key_name").(string)),
		}

		err := conn.VM.ModifyInstanceKeyPair(input)
		if err != nil {
			return err
		}
	}

	if d.HasChange("tag_set") {
		if err := setTags(conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("instance_type") && !d.IsNewResource() {
		opts := &fcu.ModifyInstanceAttributeInput{
			InstanceId: aws.String(d.Id()),
			InstanceType: &fcu.AttributeValue{
				Value: aws.String(d.Get("instance_type").(string)),
			},
		}
		if err := modifyInstanceAttr(conn, opts, "instance_type"); err != nil {
			return err
		}
	}

	if d.HasChange("user_data") && !d.IsNewResource() {
		opts := &fcu.ModifyInstanceAttributeInput{
			InstanceId: aws.String(d.Id()),
			UserData: &fcu.BlobAttributeValue{
				Value: d.Get("user_data").([]byte),
			},
		}
		if err := modifyInstanceAttr(conn, opts, "user_data"); err != nil {
			return err
		}
	}

	if d.HasChange("ebs_optimized") && !d.IsNewResource() {
		opts := &fcu.ModifyInstanceAttributeInput{
			InstanceId: aws.String(d.Id()),
			EbsOptimized: &fcu.AttributeBooleanValue{
				Value: aws.Bool(d.Get("ebs_optimized").(bool)),
			},
		}
		if err := modifyInstanceAttr(conn, opts, "ebs_optimized"); err != nil {
			return err
		}
	}

	if d.HasChange("delete_on_termination") && !d.IsNewResource() {
		opts := &fcu.ModifyInstanceAttributeInput{
			InstanceId: aws.String(d.Id()),
			DeleteOnTermination: &fcu.AttributeBooleanValue{
				Value: d.Get("delete_on_termination").(*bool),
			},
		}
		if err := modifyInstanceAttr(conn, opts, "delete_on_termination"); err != nil {
			return err
		}
	}

	if d.HasChange("disable_api_termination") {
		opts := &fcu.ModifyInstanceAttributeInput{
			InstanceId: aws.String(d.Id()),
			DisableApiTermination: &fcu.AttributeBooleanValue{
				Value: aws.Bool(d.Get("disable_api_termination").(bool)),
			},
		}
		if err := modifyInstanceAttr(conn, opts, "disable_api_termination"); err != nil {
			return err
		}
	}

	if d.HasChange("instance_initiated_shutdown_behavior") {
		opts := &fcu.ModifyInstanceAttributeInput{
			InstanceId: aws.String(d.Id()),
			InstanceInitiatedShutdownBehavior: &fcu.AttributeValue{
				Value: aws.String(d.Get("instance_initiated_shutdown_behavior").(string)),
			},
		}
		if err := modifyInstanceAttr(conn, opts, "instance_initiated_shutdown_behavior"); err != nil {
			return err
		}
	}

	if d.HasChange("group_set") {
		opts := &fcu.ModifyInstanceAttributeInput{
			InstanceId: aws.String(d.Id()),
			Groups:     d.Get("group_set").([]*string),
		}
		if err := modifyInstanceAttr(conn, opts, "group_set"); err != nil {
			return err
		}
	}

	if d.HasChange("source_dest_check") {
		opts := &fcu.ModifyInstanceAttributeInput{
			InstanceId: aws.String(d.Id()),
			SourceDestCheck: &fcu.AttributeBooleanValue{
				Value: aws.Bool(d.Get("source_dest_check").(bool)),
			},
		}
		if err := modifyInstanceAttr(conn, opts, "source_dest_check"); err != nil {
			return err
		}
	}

	if d.HasChange("block_device_mapping") {
		if err := setBlockDevice(d.Get("block_device_mapping"), conn, d.Id()); err != nil {
			return err
		}
	}

	d.Partial(false)

	return resourceVMRead(d, meta)
}

func resourceVMDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).FCU

	id := d.Id()

	req := &fcu.TerminateInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	}

	var err error
	err = resource.Retry(3*time.Minute, func() *resource.RetryError {
		_, err = conn.VM.TerminateInstances(req)

		if err != nil {
			if strings.Contains(err.Error(), "RequestLimitExceeded") {
				return resource.RetryableError(err)
			}

			if strings.Contains(err.Error(), "InvalidInstanceID.NotFound") {
				resource.NonRetryableError(err)
			}
		}

		return resource.RetryableError(err)
	})

	if err != nil {

		if strings.Contains(err.Error(), "InvalidInstanceID.NotFound") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error deleting the instance %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending", "running", "shutting-down", "stopped", "stopping"},
		Target:     []string{"terminated"},
		Refresh:    InstanceStateRefreshFunc(conn, id, ""),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to terminate: %s", id, err)
	}

	return nil
}

func getVMSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		// Attributes
		"block_device_mapping": {
			Type: schema.TypeSet,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"device_name": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"ebs": {
						Type:     schema.TypeMap,
						Optional: true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"delete_on_termination": {
									Type:     schema.TypeBool,
									Optional: true,
								},
								"iops": {
									Type:     schema.TypeInt,
									Optional: true,
								},
								"snapshot_id": {
									Type:     schema.TypeString,
									Optional: true,
								},
								"volume_size": {
									Type:     schema.TypeInt,
									Optional: true,
								},
								"volume_type": {
									Type:     schema.TypeString,
									Optional: true,
								},
							},
						},
					},
					"no_device": {
						Type:     schema.TypeBool,
						Optional: true,
					},
					"virtual_name": {
						Type:     schema.TypeString,
						Optional: true,
					},
				},
			},
			Optional: true,
		},

		"client_token": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"disable_api_termination": {
			Type:     schema.TypeBool,
			Optional: true,
			Computed: true,
		},
		"dry_run": {
			Type:     schema.TypeBool,
			Optional: true,
		},
		"ebs_optimized": {
			Type:     schema.TypeBool,
			Optional: true,
		},
		"image_id": {
			Type:     schema.TypeString,
			ForceNew: true,
			Required: true,
		},
		"instance_initiated_shutdown_behavior": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"instance_type": {
			Type:     schema.TypeString,
			ForceNew: true,
			Required: true,
		},
		"instance_name": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"key_name": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"max_count": {
			Type:     schema.TypeInt,
			Optional: true,
		},
		"min_count": {
			Type:     schema.TypeInt,
			Optional: true,
		},
		"network_interface": {
			ConflictsWith: []string{"subnet_id", "security_group_id", "security_group"},
			Type:          schema.TypeSet,
			Optional:      true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"delete_on_termination": {
						Type:     schema.TypeBool,
						Optional: true,
					},
					"description": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"device_index": {
						Type:     schema.TypeInt,
						Optional: true,
					},
					"network_interface_id": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"private_ip_address": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"private_ip_addresses_set": {
						Type:     schema.TypeSet,
						Optional: true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"primary": {
									Type:     schema.TypeBool,
									Optional: true,
								},
								"private_ip_address": {
									Type:     schema.TypeString,
									Optional: true,
								},
							},
						},
					},
					"secondary_private_ip_address_count": {
						Type:     schema.TypeInt,
						Optional: true,
					},
					"security_group_id": {
						Type:     schema.TypeSet,
						Optional: true,
						Elem:     &schema.Schema{Type: schema.TypeString},
					},
					"subnet_id": {
						Type:     schema.TypeString,
						Optional: true,
					},
				},
			},
		},
		"placement": {
			Type:     schema.TypeMap,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"affinity": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"availability_zone": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"group_name": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"host_id": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"tenancy": {
						Type:     schema.TypeString,
						Optional: true,
					},
				},
			},
		},
		"private_ip_address": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"private_ip_addresses": {
			Type:     schema.TypeString,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"ramdisk_id": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"security_group": {
			Type:     schema.TypeSet,
			Optional: true,
			Computed: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"security_group_id": {
			Type:     schema.TypeSet,
			Optional: true,

			Elem: &schema.Schema{Type: schema.TypeString},
		},
		"subnet_id": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"user_data": {
			Type:     schema.TypeString,
			Optional: true,
		},
		//Attributes reference:
		"group_set": {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"group_id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"group_name": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
			},
		},
		"instances_set": {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"ami_launch_index": {
						Type:     schema.TypeInt,
						Computed: true,
					},
					"architecture": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"block_device_mapping": {
						Type:     schema.TypeList,
						Computed: true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"device_name": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"ebs": {
									Type:     schema.TypeMap,
									Computed: true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"delete_on_termination": {
												Type:     schema.TypeBool,
												Computed: true,
											},
											"status": {
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
							},
						},
					},
					"client_token": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"instance_lifecycle": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"dns_name": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"ebs_optimized": {
						Type:     schema.TypeBool,
						Computed: true,
					},
					"group_set": {
						Type:     schema.TypeList,
						Computed: true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"group_id": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"group_name": {
									Type:     schema.TypeString,
									Computed: true,
								},
							},
						},
					},
					"hypervisor": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"iam_instance_profile": {
						Type: schema.TypeMap,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"arn": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"id": {
									Type:     schema.TypeString,
									Computed: true,
								},
							},
						},
						Computed: true,
					},
					"image_id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"instance_id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"instance_state": {
						Type: schema.TypeMap,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"code": {
									Type:     schema.TypeInt,
									Computed: true,
								},
								"name": {
									Type:     schema.TypeString,
									Computed: true,
								},
							},
						},
						Computed: true,
					},
					"instance_type": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"ip_address": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"kernel_id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"key_name": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"monitoring": {
						Type: schema.TypeMap,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"state": {
									Type:     schema.TypeString,
									Computed: true,
								},
							},
						},
						Computed: true,
					},
					"network_interface_set": {
						Type:     schema.TypeList,
						Computed: true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"association": {
									Type:     schema.TypeMap,
									Computed: true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"ip_owner_id": {
												Type:     schema.TypeString,
												Computed: true,
											},
											"public_dns_name": {
												Type:     schema.TypeString,
												Computed: true,
											},
											"public_ip": {
												Type:     schema.TypeString,
												Computed: true,
											},
										},
									},
								},
								"attachment": {
									Type: schema.TypeMap,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"attachement_id": {
												Type:     schema.TypeString,
												Computed: true,
											},
											"delete_on_termination": {
												Type:     schema.TypeBool,
												Computed: true,
											},
											"device_index": {
												Type:     schema.TypeInt,
												Computed: true,
											},
											"status": {
												Type:     schema.TypeString,
												Computed: true,
											},
										},
									},
									Computed: true,
								},
								"description": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"group_set": {
									Type:     schema.TypeList,
									Computed: true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"group_id": {
												Type:     schema.TypeString,
												Computed: true,
											},
											"group_name": {
												Type:     schema.TypeString,
												Computed: true,
											},
										},
									},
								},
								"mac_address": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"network_interface_id": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"owner_id": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"private_dns_name": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"private_ip_address": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"private_ip_addresses_set": {
									Type:     schema.TypeList,
									Computed: true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"association": {
												Type:     schema.TypeMap,
												Computed: true,
												Elem: &schema.Resource{
													Schema: map[string]*schema.Schema{
														"ip_owner_id": {
															Type:     schema.TypeString,
															Computed: true,
														},
														"public_dns_name": {
															Type:     schema.TypeString,
															Computed: true,
														},
														"public_ip": {
															Type:     schema.TypeString,
															Computed: true,
														},
													},
												},
											},
											"primary": {
												Type:     schema.TypeBool,
												Computed: true,
											},
											"private_dns_name": {
												Type:     schema.TypeString,
												Computed: true,
											},
											"private_ip_address": {
												Type:     schema.TypeString,
												Computed: true,
											},
										},
									},
								},
								"source_dest_check": {
									Type:     schema.TypeBool,
									Computed: true,
								},
								"status": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"subnet_id": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"vpc_id": {
									Type:     schema.TypeString,
									Computed: true,
								},
							},
						},
					},
					"placement": {
						Type: schema.TypeMap,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"affinity": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"availability_zone": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"group_name": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"host_id": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"tenancy": {
									Type:     schema.TypeString,
									Computed: true,
								},
							},
						},
						Computed: true,
					},
					"platform": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"private_dns_name": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"private_ip_address": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"product_codes": {
						Type: schema.TypeList,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"product_code": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"type": {
									Type:     schema.TypeString,
									Computed: true,
								},
							},
						},
						Computed: true,
					},
					"ramdisk_id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"reason": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"root_device_name": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"root_device_type": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"source_dest_check": {
						Type:     schema.TypeBool,
						Computed: true,
					},
					"spot_instance_request_id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"sriov_net_support": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"state_reason": {
						Type: schema.TypeMap,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"code": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"message": {
									Type:     schema.TypeString,
									Computed: true,
								},
							},
						},
						Computed: true,
					},
					"subnet_id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"tag_set": {
						Type: schema.TypeList,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"key": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"value": {
									Type:     schema.TypeString,
									Computed: true,
								},
							},
						},
						Computed: true,
					},
					"virtualization_type": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"vpc_id": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
			},
		},
		"owner_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"tag": tagsSchema(),
		"request_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"requester_id": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"reservation_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"password_data": {
			Type:     schema.TypeString,
			Computed: true,
		},
		//instance set is closed here
	}
}

type outscaleInstanceOpts struct {
	BlockDeviceMappings               []*fcu.BlockDeviceMapping
	DisableAPITermination             *bool
	EBSOptimized                      *bool
	DryRun                            *bool
	ImageID                           *string
	InstanceInitiatedShutdownBehavior *string
	InstanceType                      *string
	Ipv6AddressCount                  *int64
	KeyName                           *string
	NetworkInterfaces                 []*fcu.InstanceNetworkInterfaceSpecification
	Placement                         *fcu.Placement
	PrivateIPAddress                  *string
	SecurityGroupIDs                  []*string
	SecurityGroups                    []*string
	SubnetID                          *string
	UserData                          *string
	RamdiskId                         *string
	RequesterId                       *string
	ReservationId                     *string
	PasswordData                      *string
	OwnerId                           *string
	// Monitoring                        *fcu.RunInstancesMonitoringEnabled
	// SpotPlacement                     *fcu.SpotPlacement
	// Ipv6Addresses                     []*fcu.InstanceIpv6Address
	// IAMInstanceProfile                *fcu.IamInstanceProfileSpecification
}

func buildOutscaleVMOpts(
	d *schema.ResourceData, meta interface{}) (*outscaleInstanceOpts, error) {
	conn := meta.(*OutscaleClient).FCU

	opts := &outscaleInstanceOpts{
		EBSOptimized: aws.Bool(d.Get("ebs_optimized").(bool)),
		ImageID:      aws.String(d.Get("image_id").(string)),
		InstanceType: aws.String(d.Get("instance_type").(string)),
	}

	if v := d.Get("instance_initiated_shutdown_behavior").(string); v != "" {
		opts.InstanceInitiatedShutdownBehavior = aws.String(v)
	}

	userData := d.Get("user_data").(string)
	opts.UserData = &userData

	if t, hasDisableAPITerminartion := d.GetOk("disable_api_termination"); hasDisableAPITerminartion {
		opts.DisableAPITermination = aws.Bool(t.(bool))
	} else {
		opts.DisableAPITermination = aws.Bool(false)
	}

	if t, hasTenancy := d.GetOk("tenancy"); hasTenancy {
		opts.Placement.Tenancy = aws.String(t.(string))
	}

	az, azOk := d.GetOk("availability_zone")
	gn, gnOk := d.GetOk("placement_group")

	if azOk && gnOk {
		opts.Placement = &fcu.Placement{
			AvailabilityZone: aws.String(az.(string)),
			GroupName:        aws.String(gn.(string)),
		}
	}

	subnetID, hasSubnet := d.GetOk("subnet_id")

	groups := make([]*string, 0)
	if v := d.Get("security_group"); v != nil {
		groups = expandStringList(v.(*schema.Set).List())
		if len(groups) > 0 && hasSubnet {
			log.Print("[WARN] Deprecated. Attempting to use 'security_group' within a VPC instance. Use 'security_group_id' instead.")
		}
	}

	networkInterfaces, interfacesOk := d.GetOk("network_interface")
	if hasSubnet || interfacesOk {
		opts.NetworkInterfaces = buildNetworkInterfaceOpts(d, groups, networkInterfaces)
	} else {
		if hasSubnet {
			s := subnetID.(string)
			opts.SubnetID = &s
		}

		if opts.SubnetID != nil &&
			*opts.SubnetID != "" {
			opts.SecurityGroupIDs = groups
		} else {
			opts.SecurityGroups = groups
		}

		var groupIDs []*string
		if v := d.Get("security_group_id"); v != nil {

			sgs := v.(*schema.Set).List()
			for _, v := range sgs {
				str := v.(string)
				groupIDs = append(groupIDs, aws.String(str))
			}
		}
		opts.SecurityGroupIDs = groupIDs

		if v, ok := d.GetOk("private_ip"); ok {
			opts.PrivateIPAddress = aws.String(v.(string))
		}

		if v, ok := d.GetOk("ipv6_address_count"); ok {
			opts.Ipv6AddressCount = aws.Int64(int64(v.(int)))
		}
	}

	if v, ok := d.GetOk("key_name"); ok {
		opts.KeyName = aws.String(v.(string))
	}

	blockDevices, err := readBlockDeviceMappingsFromConfig(d, conn)
	if err != nil {
		return nil, err
	}
	if len(blockDevices) > 0 {
		opts.BlockDeviceMappings = blockDevices
	}

	if dryRun, ok := d.GetOk("dry_run"); ok {
		opts.DryRun = aws.Bool(dryRun.(bool))
	}

	opts.RamdiskId = aws.String(d.Get("ramdisk_id").(string))
	opts.OwnerId = aws.String(d.Get("owner_id").(string))
	opts.RequesterId = aws.String(d.Get("requester_id").(string))
	opts.ReservationId = aws.String(d.Get("reservation_id").(string))

	if p := d.Get("password_data"); p != nil {
		opts.PasswordData = aws.String(p.(string))
	} else {
		opts.PasswordData = aws.String("pending")
	}

	return opts, nil
}

func buildNetworkInterfaceOpts(d *schema.ResourceData, groups []*string, nInterfaces interface{}) []*fcu.InstanceNetworkInterfaceSpecification {
	networkInterfaces := []*fcu.InstanceNetworkInterfaceSpecification{}
	subnet, hasSubnet := d.GetOk("subnet_id")

	if hasSubnet {
		ni := &fcu.InstanceNetworkInterfaceSpecification{
			DeviceIndex: aws.Int64(int64(0)),
			SubnetId:    aws.String(subnet.(string)),
			Groups:      groups,
		}

		if v, ok := d.GetOkExists("associate_public_ip_address"); ok {
			ni.AssociatePublicIpAddress = aws.Bool(v.(bool))
		}

		if v, ok := d.GetOk("private_ip"); ok {
			ni.PrivateIpAddress = aws.String(v.(string))
		}

		if v, ok := d.GetOk("ipv6_address_count"); ok {
			ni.Ipv6AddressCount = aws.Int64(int64(v.(int)))
		}

		if v, ok := d.GetOk("security_group_id"); ok && v.(*schema.Set).Len() > 0 {
			for _, v := range v.(*schema.Set).List() {
				ni.Groups = append(ni.Groups, aws.String(v.(string)))
			}
		}

		networkInterfaces = append(networkInterfaces, ni)
	} else {
		vL := nInterfaces.(*schema.Set).List()
		for _, v := range vL {
			ini := v.(map[string]interface{})
			ni := &fcu.InstanceNetworkInterfaceSpecification{
				DeviceIndex:         aws.Int64(int64(ini["device_index"].(int))),
				NetworkInterfaceId:  aws.String(ini["network_interface_id"].(string)),
				DeleteOnTermination: aws.Bool(ini["delete_on_termination"].(bool)),
			}
			networkInterfaces = append(networkInterfaces, ni)
		}
	}

	return networkInterfaces
}

func readBlockDeviceMappingsFromConfig(
	d *schema.ResourceData, conn *fcu.Client) ([]*fcu.BlockDeviceMapping, error) {
	blockDevices := make([]*fcu.BlockDeviceMapping, 0)

	if v, ok := d.GetOk("ebs_block_device"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			ebs := &fcu.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(bd["delete_on_termination"].(bool)),
			}

			if v, ok := bd["snapshot_id"].(string); ok && v != "" {
				ebs.SnapshotId = aws.String(v)
			}

			if v, ok := bd["encrypted"].(bool); ok && v {
				ebs.Encrypted = aws.Bool(v)
			}

			if v, ok := bd["volume_size"].(int); ok && v != 0 {
				ebs.VolumeSize = aws.Int64(int64(v))
			}

			if v, ok := bd["volume_type"].(string); ok && v != "" {
				ebs.VolumeType = aws.String(v)

				if v, ok := bd["iops"].(int); ok && v > 0 {
					ebs.Iops = aws.Int64(int64(v))

				}

			}

			blockDevices = append(blockDevices, &fcu.BlockDeviceMapping{
				DeviceName: aws.String(bd["device_name"].(string)),
				Ebs:        ebs,
			})
		}
	}

	if v, ok := d.GetOk("ephemeral_block_device"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			bdm := &fcu.BlockDeviceMapping{
				DeviceName:  aws.String(bd["device_name"].(string)),
				VirtualName: aws.String(bd["virtual_name"].(string)),
			}
			if v, ok := bd["no_device"].(bool); ok && v {
				bdm.NoDevice = aws.String("")
				// When NoDevice is true, just ignore VirtualName since it's not needed
				bdm.VirtualName = nil
			}

			if bdm.NoDevice == nil && aws.StringValue(bdm.VirtualName) == "" {
				return nil, errors.New("virtual_name cannot be empty when no_device is false or undefined")
			}

			blockDevices = append(blockDevices, bdm)
		}
	}

	if v, ok := d.GetOk("root_block_device"); ok {
		vL := v.([]interface{})
		if len(vL) > 1 {
			return nil, errors.New("Cannot specify more than one root_block_device")
		}
		for _, v := range vL {
			bd := v.(map[string]interface{})
			ebs := &fcu.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(bd["delete_on_termination"].(bool)),
			}

			if v, ok := bd["volume_size"].(int); ok && v != 0 {
				ebs.VolumeSize = aws.Int64(int64(v))
			}

			if v, ok := bd["volume_type"].(string); ok && v != "" {
				ebs.VolumeType = aws.String(v)
			}

			if v, ok := bd["iops"].(int); ok && v > 0 && *ebs.VolumeType == "io1" {
				// Only set the iops attribute if the volume type is io1. Setting otherwise
				// can trigger a refresh/plan loop based on the computed value that is given
				// from AWS, and prevent us from specifying 0 as a valid iops.
				//   See https://github.com/hashicorp/terraform/pull/4146
				//   See https://github.com/hashicorp/terraform/issues/7765
				ebs.Iops = aws.Int64(int64(v))
			} else if v, ok := bd["iops"].(int); ok && v > 0 && *ebs.VolumeType != "io1" {
				// Message user about incompatibility
				fmt.Print("[WARN] IOPs is only valid for storate type io1 for EBS Volumes")
			}
		}
	}

	return blockDevices, nil
}

// InstanceStateRefreshFunc ...
func InstanceStateRefreshFunc(conn *fcu.Client, instanceID, failState string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		var resp *fcu.DescribeInstancesOutput
		var err error

		err = resource.Retry(30*time.Second, func() *resource.RetryError {
			resp, err = conn.VM.DescribeInstances(&fcu.DescribeInstancesInput{
				InstanceIds: []*string{aws.String(instanceID)},
			})
			return resource.RetryableError(err)
		})

		if err != nil {
			return nil, "", err
		}

		if resp == nil || len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
			return nil, "", nil
		}

		i := resp.Reservations[0].Instances[0]
		state := *i.State.Name

		if state == failState {
			return i, state, fmt.Errorf("Failed to reach target state. Reason: %v",
				*i.StateReason)

		}

		return i, state, nil
	}
}

// GetInstanceGetPasswordData func
func GetInstanceGetPasswordData(conn *fcu.Client, instanceID, failState string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		var resp *fcu.GetPasswordDataOutput
		var err error

		err = resource.Retry(30*time.Second, func() *resource.RetryError {
			resp, err = conn.VM.GetPasswordData(&fcu.GetPasswordDataInput{
				InstanceId: aws.String(instanceID),
			})
			return resource.RetryableError(err)
		})

		if err != nil {
			return nil, "", err
		}

		if resp == nil {
			return nil, "", nil
		}

		i := resp.PasswordData

		if len(*i) < 0 {
			return nil, "running", nil
		}
		return nil, "terminated", nil
	}
}

// InstancePa ...
func InstancePa(conn *fcu.Client, instanceID, failState string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		var resp *fcu.DescribeInstancesOutput
		var err error

		err = resource.Retry(30*time.Second, func() *resource.RetryError {
			resp, err = conn.VM.DescribeInstances(&fcu.DescribeInstancesInput{
				InstanceIds: []*string{aws.String(instanceID)},
			})

			return resource.RetryableError(err)
		})

		if err != nil {
			return nil, "", err
		}

		if resp == nil || len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
			return nil, "", nil
		}

		i := resp.Reservations[0].Instances[0]
		state := *i.State.Name

		if state == failState {
			return i, state, fmt.Errorf("Failed to reach target state. Reason: %v",
				*i.StateReason)

		}

		return i, state, nil
	}
}

func getInstanceSet(instance *fcu.Instance) *schema.Set {

	instanceSet := map[string]interface{}{}
	s := schema.NewSet(nil, []interface{}{})

	instanceSet["ami_launch_index"] = aws.Int64Value(instance.AmiLaunchIndex)
	instanceSet["ebs_optimized"] = aws.BoolValue(instance.EbsOptimized)
	instanceSet["architecture"] = aws.StringValue(instance.Architecture)
	instanceSet["client_token"] = aws.StringValue(instance.ClientToken)
	instanceSet["hypervisor"] = aws.StringValue(instance.Hypervisor)
	instanceSet["image_id"] = aws.StringValue(instance.ImageId)
	instanceSet["instance_id"] = aws.StringValue(instance.InstanceId)
	instanceSet["instance_type"] = aws.StringValue(instance.InstanceType)
	instanceSet["kernel_id"] = aws.StringValue(instance.KernelId)
	instanceSet["key_name"] = aws.StringValue(instance.KeyName)
	instanceSet["private_dns_name"] = aws.StringValue(instance.PrivateDnsName)
	instanceSet["private_ip_address"] = aws.StringValue(instance.PrivateIpAddress)
	instanceSet["root_device_name"] = aws.StringValue(instance.RootDeviceName)
	instanceSet["dns_name"] = aws.StringValue(instance.DnsName)
	instanceSet["ip_address"] = aws.StringValue(instance.IpAddress)
	instanceSet["platform"] = aws.StringValue(instance.Platform)
	instanceSet["ramdisk_id"] = aws.StringValue(instance.RamdiskId)
	instanceSet["reason"] = aws.StringValue(instance.Reason)
	instanceSet["source_dest_check"] = aws.BoolValue(instance.SourceDestCheck)
	instanceSet["spot_instance_request_id"] = aws.StringValue(instance.SpotInstanceRequestId)
	instanceSet["sriov_net_support"] = aws.StringValue(instance.SriovNetSupport)
	instanceSet["subnet_id"] = aws.StringValue(instance.SubnetId)
	instanceSet["virtualization_type"] = aws.StringValue(instance.VirtualizationType)
	instanceSet["vpc_id"] = aws.StringValue(instance.VpcId)

	s.Add(instanceSet)

	instanceSet["block_device_mapping"] = getBlockDeviceMapping(instance.BlockDeviceMappings)
	// instanceSet["group_set"] = getGroupSet(instance.GroupSet)
	// instanceSet["iam_instance_profile"] = getIAMInstanceProfile(instance.IamInstanceProfile)
	// instanceSet["instance_state"] = getInstanceState(instance.State)
	// instanceSet["monitoring"] = getMonitoring(instance.Monitoring)
	// instanceSet["network_interface_set"] = getNetworkInterfaceSet(instance.NetworkInterfaces)
	// instanceSet["placement"] = getPlacement(instance.Placement)
	// instanceSet["state_reason"] = getStateReason(instance.StateReason)
	// instanceSet["product_codes"] = getProductCodes(instance.ProductCodes)
	instanceSet["tag_set"] = getTagSet(instance.Tags)

	return s
}

func modifyInstanceAttr(conn *fcu.Client, instanceAttrOpts *fcu.ModifyInstanceAttributeInput, attr string) error {

	var err error
	var stateConf *resource.StateChangeConf

	switch attr {
	case "instance_type":
		fallthrough
	case "user_data":
		fallthrough
	case "ebs_optimized":
		fallthrough
	case "delete_on_termination":
		stateConf, err = stopInstance(instanceAttrOpts, conn, attr)
	}

	if err != nil {
		return err
	}

	if _, err := conn.VM.ModifyInstanceAttribute(instanceAttrOpts); err != nil {
		return err
	}

	switch attr {
	case "instance_type":
		fallthrough
	case "user_data":
		fallthrough
	case "ebs_optimized":
		fallthrough
	case "delete_on_termination":
		err = startInstance(instanceAttrOpts, stateConf, conn, attr)
	}

	if err != nil {
		return err
	}

	return nil
}

func stopInstance(instanceAttrOpts *fcu.ModifyInstanceAttributeInput, conn *fcu.Client, attr string) (*resource.StateChangeConf, error) {
	_, err := conn.VM.StopInstances(&fcu.StopInstancesInput{
		InstanceIds: []*string{instanceAttrOpts.InstanceId},
	})

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending", "running", "shutting-down", "stopped", "stopping"},
		Target:     []string{"stopped"},
		Refresh:    InstanceStateRefreshFunc(conn, *instanceAttrOpts.InstanceId, ""),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return nil, fmt.Errorf(
			"Error waiting for instance (%s) to stop: %s", *instanceAttrOpts.InstanceId, err)
	}

	return stateConf, nil
}

func startInstance(instanceAttrOpts *fcu.ModifyInstanceAttributeInput, stateConf *resource.StateChangeConf, conn *fcu.Client, attr string) error {
	if _, err := conn.VM.StartInstances(&fcu.StartInstancesInput{
		InstanceIds: []*string{instanceAttrOpts.InstanceId},
	}); err != nil {
		return err
	}

	stateConf = &resource.StateChangeConf{
		Pending:    []string{"pending", "stopped"},
		Target:     []string{"running"},
		Refresh:    InstanceStateRefreshFunc(conn, *instanceAttrOpts.InstanceId, ""),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for instance (%s) to become ready: %s", *instanceAttrOpts.InstanceId, err)
	}

	return nil
}
