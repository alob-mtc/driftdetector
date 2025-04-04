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

// InstanceService handles interactions with AWS EC2 instances
type InstanceService struct {
	client EC2ClientAPI
}

// NewInstanceServiceWithDefaultConfig creates a new InstanceService with the default AWS SDK configuration
func NewInstanceServiceWithDefaultConfig(ctx context.Context) (*InstanceService, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	return NewInstanceServiceWithClient(ec2.NewFromConfig(cfg)), nil
}

// NewInstanceServiceWithClient creates a new InstanceService with a provided client
func NewInstanceServiceWithClient(client EC2ClientAPI) *InstanceService {
	return &InstanceService{
		client: client,
	}
}

// GetInstanceDetails retrieves the details of an EC2 instance by ID.
func (s *InstanceService) GetInstanceDetails(ctx context.Context, instanceID string) (*models.InstanceDetails, error) {
	resp, err := s.client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe EC2 instance %s: %w", instanceID, err)
	}

	// Ensure we got exactly one reservation with one instance
	if len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("EC2 instance not found: %s", instanceID)
	}

	instance := resp.Reservations[0].Instances[0]

	// Convert to domain model
	// I could maintain a separate model for the AWS SDK response, but I prefer to keep it simple
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

	// Add subnet IDs
	if instance.SubnetId != nil {
		details.SubnetID = aws.ToString(instance.SubnetId)
	}

	return details, nil
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
