package outscale

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-outscale/osc/fcu"
)

func dataSourceOutscaleVMS() *schema.Resource {
	return &schema.Resource{
		Read:   dataSourceOutscaleVMSRead,
		Schema: getDataSourceVMSSchemas(),
	}
}

func getDataSourceVMSSchemas() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		//Attributes
		"filter": dataSourceFiltersSchema(),
		"instance_id": {
			Type:     schema.TypeList,
			Optional: true,
			ForceNew: false,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"request_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"reservation_set": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"owner_id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"reservation_id": {
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
					"instances_set": {
						Type:     schema.TypeList,
						Computed: true,
						// Set:      resourceInstancSetHash,
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
									Type: schema.TypeList,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"device_name": {
												Type:     schema.TypeString,
												Computed: true,
											},
											"ebs": {
												Type: schema.TypeMap,
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
												Computed: true,
											},
										},
									},
									Computed: true,
								},
								"client_token": {
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
								"instance_lifecycle": {
									Type:     schema.TypeString,
									Computed: true,
								},
								"instance_state": {
									Type: schema.TypeMap,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"code": {
												Type:     schema.TypeString,
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
												Type:     schema.TypeMap,
												Computed: true,
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
									Type:     schema.TypeMap,
									Computed: true,
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
									Type:     schema.TypeList,
									Computed: true,
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
												Type:     schema.TypeInt,
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
									Type:     schema.TypeList,
									Computed: true,
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
				},
			},
		},
		//End of Attributes
	}
}

func dataSourceOutscaleVMSRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*OutscaleClient).FCU.VM

	params := &fcu.DescribeInstancesInput{}

	filters, filtersOk := d.GetOk("filter")

	instancesIds, instancesIdsOk := d.GetOk("instance_id")

	if !filtersOk && !instancesIdsOk {
		return fmt.Errorf("One of instance_id or filters must be assigned")
	}

	if instancesIdsOk {
		var ids []*string

		for _, id := range instancesIds.([]interface{}) {
			ids = append(ids, aws.String(id.(string)))
		}

		params.InstanceIds = ids
	}

	// Build up search parameters
	if filtersOk {
		params.Filters = buildOutscaleDataSourceFilters(filters.(*schema.Set))
	}

	var resp *fcu.DescribeInstancesOutput
	var err error

	err = resource.Retry(30*time.Second, func() *resource.RetryError {
		resp, err = client.DescribeInstances(params)
		return resource.RetryableError(err)
	})

	if err != nil {
		return err
	}

	if resp.Reservations == nil {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again")
	}

	// If no instances were returned, return
	if len(resp.Reservations) == 0 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again")
	}

	d.SetId(resource.UniqueId())

	d.Set("owner_id", resp.Reservations[0].OwnerId)
	d.Set("request_id", resp.RequestId)
	d.Set("reservation_id", resp.Reservations[0].ReservationId)

	flattenedReservations := []map[string]interface{}{}

	for _, r := range resp.Reservations {
		var filteredInstances []*fcu.Instance
		for _, instance := range r.Instances {
			if instance.State != nil && *instance.State.Name != "terminated" {
				filteredInstances = append(filteredInstances, instance)
			}
		}

		if len(filteredInstances) == 0 {
			continue
		}

		f := map[string]interface{}{
			"owner_id":       *r.OwnerId,
			"reservation_id": *r.ReservationId,
			"group_set":      getGroupSet(r.Groups),
			"instances_set":  flattenedInstanceSet(filteredInstances),
		}
		flattenedReservations = append(flattenedReservations, f)
	}

	err = d.Set("reservation_set", flattenedReservations)

	return err
}

func dataSourceFiltersSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		ForceNew: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Required: true,
				},

				"values": {
					Type:     schema.TypeList,
					Required: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}
