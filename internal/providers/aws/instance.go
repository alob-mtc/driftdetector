package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"driftdetector/internal/models"
)

const (
	// EC2ResourceType is the AWS resource type for EC2 instances
	EC2ResourceType = "EC2Instance"
	// maxIDsPerRequest is the maximum number of instance IDs that can be requested in a single API call
	maxIDsPerRequest = 10
)

// InstanceService handles interactions with AWS EC2 instances
type InstanceService struct {
	client EC2ClientAPI
}

// NewInstanceServiceWithDefaultConfig creates a new InstanceService with the default AWS SDK configuration.
// It loads AWS credentials and region information from the environment, config files, or instance metadata.
func NewInstanceServiceWithDefaultConfig(ctx context.Context) (*InstanceService, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, NewAWSError(
			ErrConfigurationError,
			"AWS",
			"",
			"unable to load AWS SDK config",
			err,
		)
	}

	return NewInstanceServiceWithClient(ec2.NewFromConfig(cfg)), nil
}

// NewInstanceServiceWithClient creates a new InstanceService with a provided client.
// This is useful for testing and dependency injection.
func NewInstanceServiceWithClient(client EC2ClientAPI) *InstanceService {
	return &InstanceService{
		client: client,
	}
}

// GetInstancesDetails retrieves details for multiple EC2 instances in a single API call.
// This is more efficient than making separate calls for each instance.
func (s *InstanceService) GetInstancesDetails(ctx context.Context, instanceIDs []string) ([]*models.InstanceDetails, error) {
	if len(instanceIDs) == 0 {
		return nil, NewAWSError(
			ErrInvalidInput,
			EC2ResourceType,
			"",
			"at least one instance ID must be provided",
			nil,
		)
	}

	allInstances := make([]*models.InstanceDetails, 0, len(instanceIDs))
	// Process in batches
	for i := 0; i < len(instanceIDs); i += maxIDsPerRequest {
		end := i + maxIDsPerRequest
		if end > len(instanceIDs) {
			end = len(instanceIDs)
		}
		batch := instanceIDs[i:end]

		// Make the API call for this batch
		instances, err := s.getInstancesBatch(ctx, batch)
		if err != nil {
			return nil, err // Error already wrapped in getInstancesBatch
		}

		allInstances = append(allInstances, instances...)
	}

	return allInstances, nil
}

// getInstancesBatch retrieves a batch of instances (up to 50) in a single API call
func (s *InstanceService) getInstancesBatch(ctx context.Context, instanceIDs []string) ([]*models.InstanceDetails, error) {
	resp, err := s.client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	})
	if err != nil {
		// For better error messages, use just the ID if there's only one
		resourceID := fmt.Sprintf("one or more of the following: %v", instanceIDs)
		if len(instanceIDs) == 1 {
			resourceID = instanceIDs[0]
		}

		// Wrap the AWS error with our custom error type
		return nil, ClassifyAWSError(err, EC2ResourceType, resourceID)
	}

	// Process all instances in all reservations
	var instances []*models.InstanceDetails
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			details := convertInstanceToModel(instance)
			instances = append(instances, details)
		}
	}

	return instances, nil
}

// convertInstanceToModel converts an AWS EC2 instance to our domain model
func convertInstanceToModel(instance types.Instance) *models.InstanceDetails {
	instanceID := aws.ToString(instance.InstanceId)

	details := &models.InstanceDetails{
		InstanceID:   instanceID,
		InstanceType: string(instance.InstanceType),
		AMI:          aws.ToString(instance.ImageId),
		Tags:         convertTags(instance.Tags),
	}

	// Add security groups
	if len(instance.SecurityGroups) > 0 {
		details.SecurityGroups = make([]string, len(instance.SecurityGroups))
		for i, sg := range instance.SecurityGroups {
			details.SecurityGroups[i] = aws.ToString(sg.GroupId)
		}
	}

	// Add subnet ID
	if instance.SubnetId != nil {
		details.SubnetID = aws.ToString(instance.SubnetId)
	}

	return details
}

// convertTags converts AWS SDK tags to a map
func convertTags(tags []types.Tag) map[string]string {
	if len(tags) == 0 {
		return nil
	}

	result := make(map[string]string, len(tags))
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}
	return result
}
