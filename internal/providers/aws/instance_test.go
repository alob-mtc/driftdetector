package aws

import (
	"context"
	"driftdetector/internal/providers/aws/mocks"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestGetInstancesDetails_Success tests successful retrieval of multiple EC2 instances
func TestGetInstancesDetails_Success(t *testing.T) {
	mockClient := mocks.NewEC2ClientAPI(t)

	// expected instance IDs
	instanceIDs := []string{"i-1234567890abcdef0", "i-0987654321fedcba0"}

	// expected response
	expectedResponse := &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{
			{
				Instances: []types.Instance{
					{
						InstanceId:   aws.String(instanceIDs[0]),
						InstanceType: types.InstanceTypeT2Micro,
						ImageId:      aws.String("ami-12345"),
					},
					{
						InstanceId:   aws.String(instanceIDs[1]),
						InstanceType: types.InstanceTypeT2Medium,
						ImageId:      aws.String("ami-67890"),
					},
				},
			},
		},
	}

	mockClient.On("DescribeInstances",
		mock.Anything,
		mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
			return len(input.InstanceIds) == 2 &&
				input.InstanceIds[0] == instanceIDs[0] &&
				input.InstanceIds[1] == instanceIDs[1]
		}),
	).Return(expectedResponse, nil)

	service := NewInstanceServiceWithClient(mockClient)
	results, err := service.GetInstancesDetails(context.Background(), instanceIDs)

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, instanceIDs[0], results[0].InstanceID)
	assert.Equal(t, string(types.InstanceTypeT2Micro), results[0].InstanceType)
	assert.Equal(t, instanceIDs[1], results[1].InstanceID)
	assert.Equal(t, string(types.InstanceTypeT2Medium), results[1].InstanceType)
}

func TestGetInstanceDetails_InstanceNotFound(t *testing.T) {
	mockClient := mocks.NewEC2ClientAPI(t)

	// nonexistent instance ID
	instanceID := "i-nonexistent"

	expectedError := errors.New("InvalidInstanceID.NotFound")

	mockClient.On("DescribeInstances",
		mock.Anything,
		mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
			return len(input.InstanceIds) == 1 && input.InstanceIds[0] == instanceID
		}),
	).Return(nil, expectedError)

	service := NewInstanceServiceWithClient(mockClient)
	results, err := service.GetInstancesDetails(context.Background(), []string{instanceID})

	// Should return an error
	assert.Error(t, err)
	assert.Nil(t, results)

	// Verify the error is an AWS error
	var awsErr *Error
	assert.True(t, errors.As(err, &awsErr))
	assert.Equal(t, ErrResourceNotFound, awsErr.Category)
	assert.Equal(t, EC2ResourceType, awsErr.ResourceType)
	assert.Equal(t, instanceID, awsErr.ResourceID)
}

func TestGetInstanceDetails_AWSError(t *testing.T) {
	mockClient := mocks.NewEC2ClientAPI(t)

	instanceID := "i-1234567890abcdef0"

	expectedError := errors.New("AWS API error")
	mockClient.On("DescribeInstances",
		mock.Anything,
		mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
			return len(input.InstanceIds) == 1 && input.InstanceIds[0] == instanceID
		}),
	).Return(nil, expectedError)

	service := NewInstanceServiceWithClient(mockClient)
	details, err := service.GetInstancesDetails(context.Background(), []string{instanceID})

	// Should return an error
	assert.Error(t, err)
	assert.Nil(t, details)

	// Verify the error is an AWS error
	var awsErr *Error
	assert.True(t, errors.As(err, &awsErr))
	assert.Equal(t, EC2ResourceType, awsErr.ResourceType)
	assert.Equal(t, instanceID, awsErr.ResourceID)
}
