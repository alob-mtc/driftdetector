package terraform

import "github.com/hashicorp/hcl/v2"

// HCLInstance represents the structure of an aws_instance resource in HCL.
type HCLInstance struct {
	AMI            string            `hcl:"ami,optional"`
	InstanceType   string            `hcl:"instance_type"`
	Tags           map[string]string `hcl:"tags,optional"`
	SecurityGroups []string          `hcl:"vpc_security_group_ids,optional"`
	SubnetID       string            `hcl:"subnet_id,optional"`
}

// ResourceBlock represents a single resource block in HCL.
type ResourceBlock struct {
	Type string   `hcl:"type,label"`
	Name string   `hcl:"name,label"`
	Body hcl.Body `hcl:",remain"`
}

// ConfigFile represents the top-level structure containing resource blocks.
type ConfigFile struct {
	Resources []*ResourceBlock `hcl:"resource,block"`
	Remain    hcl.Body         `hcl:",remain"` // Catch-all for other blocks if necessary
}
