package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"driftdetector/internal/models"
)

// EC2ClientAPI defines the interface for EC2 operations we need to mock
//
//go:generate mockery --name=EC2ClientAPI --output=./mocks
type EC2ClientAPI interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}

// InstanceServiceAPI defines the interface for instance operations
//
//go:generate mockery --name=InstanceServiceAPI --output=./mocks
type InstanceServiceAPI interface {
	GetInstanceDetails(ctx context.Context, instanceID string) (*models.InstanceDetails, error)
}
