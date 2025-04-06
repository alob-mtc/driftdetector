package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"

	"driftdetector/internal/models"
	"driftdetector/pkg/logging"
)

const awsInstanceType = "aws_instance"

type DefaultParser struct {
	logger logging.Logger
}

// NewDefaultParser creates a new instance of DefaultParser
func NewDefaultParser() *DefaultParser {
	return NewParserWithLogger(
		logging.NewDefaultLogger(),
	)
}

// NewParserWithLogger creates a new instance of DefaultParser with a specific logger
func NewParserWithLogger(logger logging.Logger) *DefaultParser {
	return &DefaultParser{
		logger: logger,
	}
}

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
	p.logger.Debug("Searching for %s resources in configuration", awsInstanceType)
	for _, res := range cfg.Resources {
		if res.Type == awsInstanceType {
			p.logger.Info("Found aws_instance resource: %s", res.Name)
			// Found an aws_instance, now decode its attributes
			var instance HCLInstance
			diags = gohcl.DecodeBody(res.Body, nil, &instance)
			if diags.HasErrors() {
				p.logger.Warn("Failed to decode aws_instance '%s': %s", res.Name, diags.Error())
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

			p.logger.Debug("Successfully parsed instance details: type=%s, ami=%s", instance.InstanceType, instance.AMI)
			return instanceDetails, nil
		}
	}

	return nil, fmt.Errorf("no '%s' resource found in %s", awsInstanceType, configPath)
}
