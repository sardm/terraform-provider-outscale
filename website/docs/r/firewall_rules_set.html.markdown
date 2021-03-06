---
layout: "outscale"
page_title: "OUTSCALE: outscale_firewall_rules_set"
sidebar_current: "docs-outscale-resource-firewall_rules_set"
description: |-
  Creates a security group.
---

# outscale_firewall_rules_set

This action creates a security group either in the public Cloud or in a specified VPC. By default, a default security group for use in the public Cloud and a default security group for use in a VPC are created.
When launching an instance, if no security group is explicitly specified, the appropriate default security group is assigned to the instance. Default security groups include a default rule granting instances network access to each other.
When creating a security group, you specify a name. Two security groups for use in the public Cloud or for use in a VPC cannot have the same name.
You can have up to 500 security groups in the public Cloud. You can create up to 500 security groups per VPC.
To add or remove rules, use the [AuthorizeSecurityGroupIngress](http://docs.outscale.com/api_fcu/operations/Action_AuthorizeSecurityGroupIngress_post.html#_api_fcu-action_authorizesecuritygroupingress_post), [AuthorizeSecurityGroupEgress](http://docs.outscale.com/api_fcu/operations/Action_AuthorizeSecurityGroupEgress_post.html#_api_fcu-action_authorizesecuritygroupegress_post), [RevokeSecurityGroupIngress](http://docs.outscale.com/api_fcu/operations/Action_RevokeSecurityGroupIngress_post.html#_api_fcu-action_revokesecuritygroupingress_pos), or [RevokeSecurityGroupEgress](http://docs.outscale.com/api_fcu/operations/Action_RevokeSecurityGroupEgress_post.html#_api_fcu-action_revokesecuritygroupegress_pos) methods.

## Example Usage

```hcl
resource "outscale_firewall_rules_set" "web" {
		group_name = "terraform"
		group_description = "Used in the terraform acceptance tests"
		tag = {
						Name = "tf-acc-test"
		}
		vpc_id = "vpc-e9d09d63"
	}
```

## Argument Reference

The following arguments are supported:

* `group_name` - (Required) The name of the security group (between 1 and 255 of the following characters: ASCII for the public Cloud, and a-z, A-Z, 0-9, spaces or ._-:/()#,@[]+=&;{}!$* for a VPC).
* `group_description` - (Required) A description for the security group (between 1 and 255 of the following characters: ASCII for the public Cloud, and a-z, A-Z, 0-9, spaces or ._-:/()#,@[]+=&;{}!$* for a VPC).
* `ip_permissions` - The inbound rules associated with the security group.
* `ip_permissions_egress` - The outbound rules associated with the security group.
* `vpc_id` - The ID of the VPC.
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

* `group_id` - The ID of the security group.
* `request_id` - The ID of the request.
* `tag_set` - One or more tags associated with the security group.

See detailed information in [Authorize Security Group Egress](http://docs.outscale.com/api_fcu/operations/Action_AuthorizeSecurityGroupEgress_get.html#_api_fcu-action_authorizesecuritygroupegress_get).
See detailed information in [Authorize Security Group Ingress](http://docs.outscale.com/api_fcu/operations/Action_AuthorizeSecurityGroupIngress_get.html#_api_fcu-action_authorizesecuritygroupingress_get).
See detailed information in [Create Security Group](http://docs.outscale.com/api_fcu/operations/Action_CreateSecurityGroup_get.html#_api_fcu-action_createsecuritygroup_get).
See detailed information in [Delete Security Group](http://docs.outscale.com/api_fcu/operations/Action_DeleteSecurityGroup_get.html#_api_fcu-action_deletesecuritygroup_get).
See detailed information in [Describe Security Groups](http://docs.outscale.com/api_fcu/operations/Action_DescribeSecurityGroups_get.html#_api_fcu-action_describesecuritygroups_get).
See detailed information in [Revoke Security Group Egress](http://docs.outscale.com/api_fcu/operations/Action_RevokeSecurityGroupEgress_get.html#_api_fcu-action_revokesecuritygroupegress_get).
See detailed information in [Revoke Security Group Ingress](http://docs.outscale.com/api_fcu/operations/Action_RevokeSecurityGroupIngress_get.html#_api_fcu-action_revokesecuritygroupingress_get).
