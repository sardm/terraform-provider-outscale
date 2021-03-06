package outscale

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-outscale/osc/fcu"
)

func resourceOutscalePublicIP() *schema.Resource {
	return &schema.Resource{
		Create: resourceOutscalePublicIPCreate,
		Read:   resourceOutscalePublicIPRead,
		Delete: resourceOutscalePublicIPDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: getPublicIPSchema(),
	}
}

func getPublicIPSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		// Attributes
		"allocation_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"association_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"domain": {
			Type:     schema.TypeString,
			Computed: true,
			Optional: true,
		},
		"instance_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"network_interface_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"network_interface_owner_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"private_ip_address": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"public_ip": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"request_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
	}
}

func resourceOutscalePublicIPCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).FCU

	// By default, we're not in a VPC
	domainOpt := resourceOutscalePublicIPDomain(d)

	allocOpts := &fcu.AllocateAddressInput{
		Domain: aws.String(domainOpt),
	}

	fmt.Printf("[DEBUG] EIP create configuration: %#v", allocOpts)
	allocResp, err := conn.VM.AllocateAddress(allocOpts)
	if err != nil {
		return fmt.Errorf("Error creating EIP: %s", err)
	}

	// The domain tells us if we're in a VPC or not
	d.Set("domain", allocResp.Domain)

	// Assign the eips (unique) allocation id for use later
	// the EIP api has a conditional unique ID (really), so
	// if we're in a VPC we need to save the ID as such, otherwise
	// it defaults to using the public IP
	fmt.Printf("[DEBUG] EIP Allocate: %#v", allocResp)
	if d.Get("domain").(string) == "vpc" {
		d.SetId(*allocResp.AllocationId)
	} else {
		d.SetId(*allocResp.PublicIp)
	}

	fmt.Printf("[INFO] EIP ID: %s (domain: %v)", d.Id(), *allocResp.Domain)
	return resourceOutscalePublicIPUpdate(d, meta)
}

func resourceOutscalePublicIPRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).FCU

	domain := resourceOutscalePublicIPDomain(d)
	id := d.Id()

	req := &fcu.DescribeAddressesInput{}

	if domain == "vpc" {
		req.AllocationIds = []*string{aws.String(id)}
	} else {
		req.PublicIps = []*string{aws.String(id)}
	}

	var describeAddresses *fcu.DescribeAddressesOutput
	err := resource.Retry(60*time.Second, func() *resource.RetryError {
		var err error
		describeAddresses, err = conn.VM.DescribeAddressesRequest(req)

		return resource.RetryableError(err)
	})

	if err != nil {
		if e := fmt.Sprint(err); strings.Contains(e, "InvalidAllocationID.NotFound") || strings.Contains(e, "InvalidAddress.NotFound") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving EIP: %s", err)
	}

	// Verify Outscale returned our EIP
	if len(describeAddresses.Addresses) != 1 ||
		domain == "vpc" && *describeAddresses.Addresses[0].AllocationId != id ||
		*describeAddresses.Addresses[0].PublicIp != id {
		if err != nil {
			return fmt.Errorf("Unable to find EIP: %#v", describeAddresses.Addresses)
		}
	}

	address := describeAddresses.Addresses[0]

	fmt.Printf("[DEBUG] EIP read configuration: %+v", *address)

	if address.AssociationId != nil {
		d.Set("association_id", *address.AssociationId)
	} else {
		d.Set("association_id", "")
	}
	if address.InstanceId != nil {
		d.Set("instance_id", *address.InstanceId)
	} else {
		d.Set("instance_id", "")
	}
	if address.NetworkInterfaceId != nil {
		d.Set("network_interface_id", *address.NetworkInterfaceId)
	} else {
		d.Set("network_interface_id", "")
	}
	if address.NetworkInterfaceOwnerId != nil {
		d.Set("network_interface_owner_id", *address.NetworkInterfaceOwnerId)
	} else {
		d.Set("network_interface_owner_id", "")
	}
	if address.PrivateIpAddress != nil {
		d.Set("private_ip_address", *address.PrivateIpAddress)
	} else {
		d.Set("private_ip_address", "")
	}

	d.Set("request_id", *describeAddresses.RequestId)
	d.Set("public_ip", *address.PublicIp)
	d.Set("domain", *address.Domain)
	d.Set("allocation_id", *address.AllocationId)

	// Force ID to be an Allocation ID if we're on a VPC
	// This allows users to import the EIP based on the IP if they are in a VPC
	if *address.Domain == "vpc" && net.ParseIP(id) != nil {
		fmt.Printf("[DEBUG] Re-assigning EIP ID (%s) to it's Allocation ID (%s)", d.Id(), *address.AllocationId)
		d.SetId(*address.AllocationId)
	} else {
		d.SetId(*address.PublicIp)
	}

	return nil
}

func resourceOutscalePublicIPUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).FCU

	domain := resourceOutscalePublicIPDomain(d)

	// Associate to instance or interface if specified
	vInstance, okInstance := d.GetOk("instance_id")
	vInterface, okInterface := d.GetOk("network_interface_id")

	if okInstance || okInterface {
		instanceID := vInstance.(string)
		networkInterfaceID := vInterface.(string)

		assocOpts := &fcu.AssociateAddressInput{
			InstanceId: aws.String(instanceID),
			PublicIp:   aws.String(d.Id()),
		}

		// more unique ID conditionals
		if domain == "vpc" {
			var privateIPAddress *string
			if v := d.Get("associate_with_private_ip").(string); v != "" {
				privateIPAddress = aws.String(v)
			}
			assocOpts = &fcu.AssociateAddressInput{
				NetworkInterfaceId: aws.String(networkInterfaceID),
				InstanceId:         aws.String(instanceID),
				AllocationId:       aws.String(d.Id()),
				PrivateIpAddress:   privateIPAddress,
			}
		}

		err := resource.Retry(120*time.Second, func() *resource.RetryError {
			var err error
			_, err = conn.VM.AssociateAddress(assocOpts)

			if err != nil {
				if e := fmt.Sprint(err); strings.Contains(e, "InvalidAllocationID.NotFound") || strings.Contains(e, "InvalidAddress.NotFound") {
					return resource.RetryableError(err)
				}

				return resource.NonRetryableError(err)
			}

			return nil
		})

		if err != nil {
			// Prevent saving instance if association failed
			// e.g. missing internet gateway in VPC
			d.Set("instance_id", "")
			d.Set("network_interface_id", "")
			return fmt.Errorf("Failure associating EIP: %s", err)
		}
	}

	return resourceOutscalePublicIPRead(d, meta)
}

func resourceOutscalePublicIPDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*OutscaleClient).FCU

	if err := resourceOutscalePublicIPRead(d, meta); err != nil {
		return err
	}
	if d.Id() == "" {
		// This might happen from the read
		return nil
	}

	vInstance, okInstance := d.GetOk("instance")
	vAssociationID, okAssociationID := d.GetOk("association_id")

	// If we are attached to an instance or interface, detach first.
	if (okInstance && vInstance.(string) != "") || (okAssociationID && vAssociationID.(string) != "") {
		fmt.Printf("[DEBUG] Disassociating EIP: %s", d.Id())
		var err error
		switch resourceOutscalePublicIPDomain(d) {
		case "vpc":
			_, err = conn.VM.DisassociateAddress(&fcu.DisassociateAddressInput{
				AssociationId: aws.String(d.Get("association_id").(string)),
			})
		case "standard":
			_, err = conn.VM.DisassociateAddress(&fcu.DisassociateAddressInput{
				PublicIp: aws.String(d.Get("public_ip").(string)),
			})
		}

		if err != nil {

			// Verify the error is what we want
			if e := fmt.Sprint(err); strings.Contains(e, "InvalidAllocationID.NotFound") || strings.Contains(e, "InvalidAddress.NotFound") {
				return nil
			}

			return err
		}
	}

	domain := resourceOutscalePublicIPDomain(d)
	return resource.Retry(3*time.Minute, func() *resource.RetryError {
		var err error
		switch domain {
		case "vpc":
			fmt.Printf(
				"[DEBUG] EIP release (destroy) address allocation: %v",
				d.Id())
			_, err = conn.VM.ReleaseAddress(&fcu.ReleaseAddressInput{
				AllocationId: aws.String(d.Id()),
			})
		case "standard":
			fmt.Printf("[DEBUG] EIP release (destroy) address: %v", d.Id())
			_, err = conn.VM.ReleaseAddress(&fcu.ReleaseAddressInput{
				PublicIp: aws.String(d.Id()),
			})
		}

		// Verify the error is what we want
		if e := fmt.Sprint(err); strings.Contains(e, "InvalidAllocationID.NotFound") || strings.Contains(e, "InvalidAddress.NotFound") {
			return nil
		}

		if err == nil {
			return nil
		}
		if _, ok := err.(awserr.Error); !ok {
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(err)
	})
}

func resourceOutscalePublicIPDomain(d *schema.ResourceData) string {
	if v, ok := d.GetOk("domain"); ok {
		return v.(string)
	} else if strings.Contains(d.Id(), "eipalloc") {
		// We have to do this for backwards compatibility since TF 0.1
		// didn't have the "domain" computed attribute.
		return "vpc"
	}

	return "standard"
}
