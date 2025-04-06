package models

// InstanceDetails holds configuration details for an EC2 instance from a source (AWS or Terraform).
type InstanceDetails struct {
	InstanceID     string            `json:"instance_id,omitempty"`
	InstanceType   string            `json:"instance_type,omitempty"`
	AMI            string            `json:"ami,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	SecurityGroups []string          `json:"security_groups,omitempty"`
	SubnetID       string            `json:"subnet_id,omitempty"`
}

// DriftDetail represents the difference found for a specific attribute.
type DriftDetail struct {
	Attribute      string
	AWSValue       any
	TerraformValue any
}
