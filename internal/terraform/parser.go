package terraform

import (
	"driftdetector/internal/models"
	"fmt"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"log"
)

const awsInstanceType = "aws_instance"

type DefaultParser struct{}

// ParseHCLConfig parses an HCL configuration file and extracts the details of the first aws_instance resource found.
func (p DefaultParser) ParseHCLConfig(configPath string) (*models.InstanceDetails, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(configPath)

	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL file %s: %s", configPath, diags.Error())
	}

	if file == nil || file.Body == nil {
		return nil, fmt.Errorf("parsed HCL file is empty or invalid: %s", configPath)
	}

	// First, decode the top-level resource blocks
	var cfg ConfigFile
	diags = gohcl.DecodeBody(file.Body, nil, &cfg)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode HCL body %s: %s", configPath, diags.Error())
	}

	// Find aws_instance resource blocks
	for _, res := range cfg.Resources {
		if res.Type == awsInstanceType {
			// Found an aws_instance, now decode its attributes
			var instance HCLInstance
			diags = gohcl.DecodeBody(res.Body, nil, &instance)
			if diags.HasErrors() {
				log.Printf("Warning: failed to decode aws_instance '%s': %s", res.Name, diags.Error())
				continue
			}

			// Map to domain model
			instanceDetails := &models.InstanceDetails{
				InstanceType:   instance.InstanceType,
				AMI:            instance.AMI,
				Tags:           instance.Tags,
				SecurityGroups: instance.SecurityGroups,
				SubnetID:       instance.SubnetID,
				// InstanceID is not defined in HCL, it is assigned by AWS
			}

			return instanceDetails, nil
		}
	}

	return nil, fmt.Errorf("no '%s' resource found in %s", awsInstanceType, configPath)
}
