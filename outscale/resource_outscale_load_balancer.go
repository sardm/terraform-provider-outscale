package outscale

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-outscale/osc/lbu"
)

func resourceOutscaleLoadBalancer() *schema.Resource {
	return &schema.Resource{
		Create: resourceOutscaleLoadBalancerCreate,
		Read:   resourceOutscaleLoadBalancerRead,
		Update: resourceOutscaleLoadBalancerUpdate,
		Delete: resourceOutscaleLoadBalancerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"availability_zones": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"listeners": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"instance_protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"load_balancer_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"ssl_certificate_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"load_balancer_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"scheme": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"security_groups": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"subnets": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tag": tagsSchema(),

			"dns_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_time": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"health_check": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"healthy_threshold": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"unhealthy_threshold": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"target": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"interval": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"timeout": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"instances": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"listener_descriptions": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"listener": &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"instance_port": &schema.Schema{
										Type:     schema.TypeInt,
										Computed: true,
									},
									"instance_protocol": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"load_balancer_port": &schema.Schema{
										Type:     schema.TypeInt,
										Computed: true,
									},
									"protocol": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"ssl_certificate_id": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"policy_names": &schema.Schema{
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"source_security_group": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"group_name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"owner_alias": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"policies_app_cookie_stickiness_policies": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cookie_name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"policy_name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"policies_lb_cookie_stickiness_policies": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"policy_name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"policies_other_policies": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"request_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOutscaleLoadBalancerCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).LBU

	elbOpts := &lbu.CreateLoadBalancerInput{}

	listeners, err := expandListeners(d.Get("listeners").([]interface{}))
	if err != nil {
		return err
	}

	elbOpts.Listeners = listeners

	if v, ok := d.GetOk("load_balancer_name"); ok {
		elbOpts.LoadBalancerName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("tag"); ok {
		elbOpts.Tags = tagsFromMapLBU(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("scheme"); ok {
		elbOpts.Scheme = aws.String(v.(string))
	}

	if v, ok := d.GetOk("availability_zones"); ok {
		elbOpts.AvailabilityZones = expandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("security_groups"); ok {
		elbOpts.SecurityGroups = expandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("subnets"); ok {
		elbOpts.Subnets = expandStringList(v.([]interface{}))
	}

	log.Printf("[DEBUG] ELB create configuration: %#v", elbOpts)
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err = conn.API.CreateLoadBalancer(elbOpts)

		if err != nil {
			if strings.Contains(fmt.Sprint(err), "CertificateNotFound") {
				return resource.RetryableError(
					fmt.Errorf("[WARN] Error creating ELB Listener with SSL Cert, retrying: %s", err))
			}
			if strings.Contains(fmt.Sprint(err), "Throttling") {
				return resource.RetryableError(
					fmt.Errorf("[WARN] Error creating ELB Listener with SSL Cert, retrying: %s", err))
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Assign the lbu's unique identifier for use later
	d.SetId(*elbOpts.LoadBalancerName)
	log.Printf("[INFO] ELB ID: %s", d.Id())

	if err := d.Set("listener_descriptions", make([]map[string]interface{}, 0)); err != nil {
		return err
	}
	d.Set("policies", make([]map[string]interface{}, 0))

	return resourceOutscaleLoadBalancerRead(d, meta)
}

func resourceOutscaleLoadBalancerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).LBU
	elbName := d.Id()

	// Retrieve the ELB properties for updating the state
	describeElbOpts := &lbu.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{aws.String(elbName)},
	}

	var resp *lbu.DescribeLoadBalancersOutput
	var describeResp *lbu.DescribeLoadBalancersResult
	var err error
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		resp, err = conn.API.DescribeLoadBalancers(describeElbOpts)
		if err != nil {
			if strings.Contains(fmt.Sprint(err), "Throttling:") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		describeResp = resp.DescribeLoadBalancersResult
		return nil
	})

	if err != nil {
		if isLoadBalancerNotFound(err) {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving ELB: %s", err)
	}

	if describeResp.LoadBalancerDescriptions == nil {
		return fmt.Errorf("NO ELB FOUND")
	}

	if len(describeResp.LoadBalancerDescriptions) != 1 {
		return fmt.Errorf("Unable to find ELB: %#v", describeResp.LoadBalancerDescriptions)
	}

	lb := describeResp.LoadBalancerDescriptions[0]

	d.Set("availability_zones", flattenStringList(lb.AvailabilityZones))
	d.Set("dns_name", aws.StringValue(lb.DNSName))
	d.Set("health_check", flattenHealthCheck(lb.HealthCheck))
	d.Set("created_time", lb.CreatedTime.String())

	if lb.Instances != nil {
		d.Set("instances", flattenInstances(lb.Instances))
	} else {
		d.Set("instances", make([]map[string]interface{}, 0))
	}
	if lb.ListenerDescriptions != nil {
		if err := d.Set("listener_descriptions", flattenListeners(lb.ListenerDescriptions)); err != nil {
			return err
		}
	} else {
		if err := d.Set("listener_descriptions", make([]map[string]interface{}, 0)); err != nil {
			return err
		}
	}
	d.Set("load_balancer_name", aws.StringValue(lb.LoadBalancerName))

	appPolicies := make([]map[string]interface{}, 0)
	lbPolicies := make([]map[string]interface{}, 0)
	otherPolicies := make([]interface{}, 0)

	if lb.Policies != nil {
		for _, v := range lb.Policies.AppCookieStickinessPolicies {
			a := make(map[string]interface{})
			a["cookie_name"] = aws.StringValue(v.CookieName)
			a["policy_name"] = aws.StringValue(v.PolicyName)
			appPolicies = append(appPolicies, a)
		}

		for _, v := range lb.Policies.LBCookieStickinessPolicies {
			a := make(map[string]interface{})
			a["policy_name"] = aws.StringValue(v.PolicyName)
			lbPolicies = append(lbPolicies, a)
		}

		otherPolicies = flattenStringList(lb.Policies.OtherPolicies)
	}

	d.Set("policies_app_cookie_stickiness_policies", appPolicies)
	d.Set("policies_lb_cookie_stickiness_policies", lbPolicies)
	d.Set("policies_other_policies", otherPolicies)

	d.Set("scheme", aws.StringValue(lb.Scheme))
	if lb.SecurityGroups != nil {
		d.Set("security_groups", flattenStringList(lb.SecurityGroups))
	} else {
		d.Set("security_groups", make([]map[string]interface{}, 0))
	}
	ssg := make(map[string]string)
	if lb.SourceSecurityGroup != nil {
		ssg["group_name"] = aws.StringValue(lb.SourceSecurityGroup.GroupName)
		ssg["owner_alias"] = aws.StringValue(lb.SourceSecurityGroup.OwnerAlias)
	}
	d.Set("source_security_group", ssg)
	d.Set("subnets", flattenStringList(lb.Subnets))
	d.Set("vpc_id", aws.StringValue(lb.VPCId))
	d.Set("request_id", resp.ResponseMetadata.RequestID)

	return nil
}

func resourceOutscaleLoadBalancerUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).LBU

	d.Partial(true)

	if d.HasChange("listeners") {
		o, n := d.GetChange("listeners")
		os := o.([]interface{})
		ns := n.([]interface{})

		remove, _ := expandListeners(ns)
		add, _ := expandListeners(os)

		if len(remove) > 0 {
			ports := make([]*int64, 0, len(remove))
			for _, listener := range remove {
				ports = append(ports, listener.LoadBalancerPort)
			}

			deleteListenersOpts := &lbu.DeleteLoadBalancerListenersInput{
				LoadBalancerName:  aws.String(d.Id()),
				LoadBalancerPorts: ports,
			}

			var err error
			err = resource.Retry(5*time.Minute, func() *resource.RetryError {
				_, err = conn.API.DeleteLoadBalancerListeners(deleteListenersOpts)

				if err != nil {
					if strings.Contains(err.Error(), "Throttling:") {
						return resource.RetryableError(err)
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})

			if err != nil {
				return fmt.Errorf("Failure removing outdated ELB listeners: %s", err)
			}
		}

		if len(add) > 0 {
			createListenersOpts := &lbu.CreateLoadBalancerListenersInput{
				LoadBalancerName: aws.String(d.Id()),
				Listeners:        add,
			}

			// Occasionally AWS will error with a 'duplicate listener', without any
			// other listeners on the ELB. Retry here to eliminate that.
			var err error
			err = resource.Retry(5*time.Minute, func() *resource.RetryError {
				_, err = conn.API.CreateLoadBalancerListeners(createListenersOpts)
				if err != nil {
					if strings.Contains(fmt.Sprint(err), "DuplicateListener") {
						log.Printf("[DEBUG] Duplicate listener found for ELB (%s), retrying", d.Id())
						return resource.RetryableError(err)
					}
					if strings.Contains(fmt.Sprint(err), "CertificateNotFound") && strings.Contains(fmt.Sprint(err), "Server Certificate not found for the key: arn") {
						log.Printf("[DEBUG] SSL Cert not found for given ARN, retrying")
						return resource.RetryableError(err)
					}
					if strings.Contains(fmt.Sprint(err), "Throttling") && strings.Contains(fmt.Sprint(err), "Server Certificate not found for the key: arn") {
						log.Printf("[DEBUG] SSL Cert not found for given ARN, retrying")
						return resource.RetryableError(err)
					}
					// Didn't recognize the error, so shouldn't retry.
					return resource.NonRetryableError(err)
				}
				// Successful creation
				return nil
			})
			if err != nil {
				return fmt.Errorf("Failure adding new or updated ELB listeners: %s", err)
			}
		}

		d.SetPartial("listeners")
	}

	if d.HasChange("instances") {
		o, n := d.GetChange("instances")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		remove := expandInstanceString(os.Difference(ns).List())
		add := expandInstanceString(ns.Difference(os).List())

		if len(add) > 0 {
			registerInstancesOpts := lbu.RegisterInstancesWithLoadBalancerInput{
				LoadBalancerName: aws.String(d.Id()),
				Instances:        add,
			}

			var err error
			err = resource.Retry(5*time.Minute, func() *resource.RetryError {
				_, err = conn.API.RegisterInstancesWithLoadBalancer(&registerInstancesOpts)

				if err != nil {
					if strings.Contains(err.Error(), "Throttling:") {
						return resource.RetryableError(err)
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})

			if err != nil {
				return fmt.Errorf("Failure registering instances with ELB: %s", err)
			}
		}
		if len(remove) > 0 {
			deRegisterInstancesOpts := lbu.DeregisterInstancesFromLoadBalancerInput{
				LoadBalancerName: aws.String(d.Id()),
				Instances:        remove,
			}

			var err error
			err = resource.Retry(5*time.Minute, func() *resource.RetryError {
				_, err = conn.API.DeregisterInstancesFromLoadBalancer(&deRegisterInstancesOpts)

				if err != nil {
					if strings.Contains(err.Error(), "Throttling:") {
						return resource.RetryableError(err)
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})

			if err != nil {
				return fmt.Errorf("Failure deregistering instances from ELB: %s", err)
			}
		}

		d.SetPartial("instances")
	}

	if d.HasChange("health_check") {
		hc := d.Get("health_check").([]interface{})
		if len(hc) > 0 {
			check := hc[0].(map[string]interface{})
			configureHealthCheckOpts := lbu.ConfigureHealthCheckInput{
				LoadBalancerName: aws.String(d.Id()),
				HealthCheck: &lbu.HealthCheck{
					HealthyThreshold:   aws.Int64(int64(check["healthy_threshold"].(int))),
					UnhealthyThreshold: aws.Int64(int64(check["unhealthy_threshold"].(int))),
					Interval:           aws.Int64(int64(check["interval"].(int))),
					Target:             aws.String(check["target"].(string)),
					Timeout:            aws.Int64(int64(check["timeout"].(int))),
				},
			}
			var err error

			err = resource.Retry(5*time.Minute, func() *resource.RetryError {
				_, err = conn.API.ConfigureHealthCheck(&configureHealthCheckOpts)

				if err != nil {
					if strings.Contains(err.Error(), "Throttling:") {
						return resource.RetryableError(err)
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})

			if err != nil {
				return fmt.Errorf("Failure configuring health check for ELB: %s", err)
			}
			d.SetPartial("health_check")
		}
	}

	if d.HasChange("security_groups") {
		groups := d.Get("security_groups").([]interface{})

		applySecurityGroupsOpts := lbu.ApplySecurityGroupsToLoadBalancerInput{
			LoadBalancerName: aws.String(d.Id()),
			SecurityGroups:   expandStringList(groups),
		}

		var err error
		err = resource.Retry(5*time.Minute, func() *resource.RetryError {
			_, err = conn.API.ApplySecurityGroupsToLoadBalancer(&applySecurityGroupsOpts)

			if err != nil {
				if strings.Contains(err.Error(), "Throttling:") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("Failure applying security groups to ELB: %s", err)
		}

		d.SetPartial("security_groups")
	}

	if d.HasChange("availability_zones") {
		o, n := d.GetChange("availability_zones")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		removed := expandStringList(os.Difference(ns).List())
		added := expandStringList(ns.Difference(os).List())

		if len(added) > 0 {
			enableOpts := &lbu.EnableAvailabilityZonesForLoadBalancerInput{
				LoadBalancerName:  aws.String(d.Id()),
				AvailabilityZones: added,
			}

			var err error

			err = resource.Retry(5*time.Minute, func() *resource.RetryError {
				_, err = conn.API.EnableAvailabilityZonesForLoadBalancer(enableOpts)

				if err != nil {
					if strings.Contains(err.Error(), "Throttling:") {
						return resource.RetryableError(err)
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})

			if err != nil {
				return fmt.Errorf("Failure enabling ELB availability zones: %s", err)
			}
		}

		if len(removed) > 0 {
			disableOpts := &lbu.DisableAvailabilityZonesForLoadBalancerInput{
				LoadBalancerName:  aws.String(d.Id()),
				AvailabilityZones: removed,
			}

			var err error

			err = resource.Retry(5*time.Minute, func() *resource.RetryError {
				_, err = conn.API.DisableAvailabilityZonesForLoadBalancer(disableOpts)

				if err != nil {
					if strings.Contains(err.Error(), "Throttling:") {
						return resource.RetryableError(err)
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})

			if err != nil {
				return fmt.Errorf("Failure disabling ELB availability zones: %s", err)
			}
		}

		d.SetPartial("availability_zones")
	}

	if d.HasChange("subnets") {
		o, n := d.GetChange("subnets")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		removed := expandStringList(os.Difference(ns).List())
		added := expandStringList(ns.Difference(os).List())

		if len(removed) > 0 {
			detachOpts := &lbu.DetachLoadBalancerFromSubnetsInput{
				LoadBalancerName: aws.String(d.Id()),
				Subnets:          removed,
			}

			var err error

			err = resource.Retry(5*time.Minute, func() *resource.RetryError {
				_, err := conn.API.DetachLoadBalancerFromSubnets(detachOpts)

				if err != nil {
					if strings.Contains(err.Error(), "Throttling:") {
						return resource.RetryableError(err)
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})

			if err != nil {
				return fmt.Errorf("Failure removing ELB subnets: %s", err)
			}
		}

		if len(added) > 0 {
			attachOpts := &lbu.AttachLoadBalancerToSubnetsInput{
				LoadBalancerName: aws.String(d.Id()),
				Subnets:          added,
			}
			var err error

			err = resource.Retry(5*time.Minute, func() *resource.RetryError {
				_, err = conn.API.AttachLoadBalancerToSubnets(attachOpts)
				if err != nil {
					if err, ok := err.(awserr.Error); ok {
						// eventually consistent issue with removing a subnet in AZ1 and
						// immediately adding a new one in the same AZ
						if err.Code() == "InvalidConfigurationRequest" && strings.Contains(err.Message(), "cannot be attached to multiple subnets in the same AZ") {
							log.Printf("[DEBUG] retrying az association")
							return resource.RetryableError(err)
						}
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("Failure adding ELB subnets: %s", err)
			}
		}

		d.SetPartial("subnets")
	}

	d.SetPartial("listener_descriptions")
	d.SetPartial("policies")

	d.Partial(false)

	return resourceOutscaleLoadBalancerRead(d, meta)
}

func resourceOutscaleLoadBalancerDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).LBU

	log.Printf("[INFO] Deleting ELB: %s", d.Id())

	// Destroy the load balancer
	deleteElbOpts := lbu.DeleteLoadBalancerInput{
		LoadBalancerName: aws.String(d.Id()),
	}
	var err error

	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err = conn.API.DeleteLoadBalancer(&deleteElbOpts)
		if err != nil {
			if strings.Contains(err.Error(), "Throttling:") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error deleting ELB: %s", err)
	}

	d.SetId("")

	return nil
}

func isLoadBalancerNotFound(err error) bool {
	return strings.Contains(fmt.Sprint(err), "LoadBalancerNotFound")
}

func expandListeners(configured []interface{}) ([]*lbu.Listener, error) {
	listeners := make([]*lbu.Listener, 0, len(configured))

	for _, lRaw := range configured {
		data := lRaw.(map[string]interface{})

		ip := int64(data["instance_port"].(int))
		lp := int64(data["load_balancer_port"].(int))
		l := &lbu.Listener{
			InstancePort:     &ip,
			InstanceProtocol: aws.String(data["instance_protocol"].(string)),
			LoadBalancerPort: &lp,
			Protocol:         aws.String(data["protocol"].(string)),
		}

		if v, ok := data["ssl_certificate_id"]; ok && v != "" {
			l.SSLCertificateId = aws.String(v.(string))
		}

		var valid bool
		if l.SSLCertificateId != nil && *l.SSLCertificateId != "" {
			// validate the protocol is correct
			for _, p := range []string{"https", "ssl"} {
				if (strings.ToLower(*l.InstanceProtocol) == p) || (strings.ToLower(*l.Protocol) == p) {
					valid = true
				}
			}
		} else {
			valid = true
		}

		if valid {
			listeners = append(listeners, l)
		} else {
			return nil, fmt.Errorf("[ERR] ELB Listener: ssl_certificate_id may be set only when protocol is 'https' or 'ssl'")
		}
	}

	return listeners, nil
}

func flattenStringList(list []*string) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		vs = append(vs, *v)
	}
	return vs
}

func flattenInstances(list []*lbu.Instance) []map[string]string {
	result := make([]map[string]string, len(list))
	for _, i := range list {
		result = append(result, map[string]string{"instance_id": *i.InstanceId})
	}
	return result
}

// Expands an array of String Instance IDs into a []Instances
func expandInstanceString(list []interface{}) []*lbu.Instance {
	result := make([]*lbu.Instance, 0, len(list))
	for _, i := range list {
		result = append(result, &lbu.Instance{InstanceId: aws.String(i.(string))})
	}
	return result
}

// Flattens an array of Backend Descriptions into a a map of instance_port to policy names.
func flattenBackendPolicies(backends []*lbu.BackendServerDescription) map[int64][]string {
	policies := make(map[int64][]string)
	for _, i := range backends {
		for _, p := range i.PolicyNames {
			policies[*i.InstancePort] = append(policies[*i.InstancePort], *p)
		}
		sort.Strings(policies[*i.InstancePort])
	}
	return policies
}

// Flattens an array of Listeners into a []map[string]interface{}
func flattenListeners(list []*lbu.ListenerDescription) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))

	for _, i := range list {
		l := make(map[string]interface{})
		listener := map[string]interface{}{
			"instance_port":      strconv.Itoa(int(aws.Int64Value(i.Listener.InstancePort))),
			"instance_protocol":  strings.ToLower(aws.StringValue(i.Listener.InstanceProtocol)),
			"load_balancer_port": strconv.Itoa(int(aws.Int64Value(i.Listener.LoadBalancerPort))),
			"protocol":           strings.ToLower(aws.StringValue(i.Listener.Protocol)),
			"ssl_certificate_id": aws.StringValue(i.Listener.SSLCertificateId),
		}
		l["listener"] = listener
		l["policy_names"] = flattenStringList(i.PolicyNames)
		result = append(result, l)
	}
	return result
}

func flattenHealthCheck(check *lbu.HealthCheck) map[string]interface{} {
	chk := make(map[string]interface{})

	if check != nil {
		chk["unhealthy_threshold"] = strconv.Itoa(int(aws.Int64Value(check.UnhealthyThreshold)))
		chk["healthy_threshold"] = strconv.Itoa(int(aws.Int64Value(check.HealthyThreshold)))
		chk["target"] = aws.StringValue(check.Target)
		chk["timeout"] = strconv.Itoa(int(aws.Int64Value(check.Timeout)))
		chk["interval"] = strconv.Itoa(int(aws.Int64Value(check.Interval)))
	}

	return chk
}
