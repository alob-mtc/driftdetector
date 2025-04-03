package aws

import (
	"context"
	"driftdetector/internal/aws/mocks"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetInstanceDetails_Success(t *testing.T) {
	mockClient := mocks.NewEC2ClientAPI(t)

	// expected instance ID
	instanceID := "i-1234567890abcdef0"

	// expected response
	expectedResponse := &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{
			{
				Instances: []types.Instance{
					{
						InstanceId:   aws.String(instanceID),
						InstanceType: types.InstanceTypeT2Micro,
						ImageId:      aws.String("ami-12345"),
						Tags: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String("test-instance"),
							},
							{
								Key:   aws.String("Environment"),
								Value: aws.String("testing"),
							},
						},
						SecurityGroups: []types.GroupIdentifier{
							{
								GroupId:   aws.String("sg-12345"),
								GroupName: aws.String("test-sg"),
							},
						},
						SubnetId: aws.String("subnet-12345"),
					},
				},
			},
		},
	}

	mockClient.On("DescribeInstances",
		mock.Anything,
		mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
			return len(input.InstanceIds) == 1 && input.InstanceIds[0] == instanceID
		}),
	).Return(expectedResponse, nil)

	service := NewInstanceServiceWithClient(mockClient)
	details, err := service.GetInstanceDetails(context.Background(), instanceID)

	assert.NoError(t, err)
	assert.NotNil(t, details)

	assert.Equal(t, instanceID, details.InstanceID)
	assert.Equal(t, "t2.micro", details.InstanceType)
	assert.Equal(t, "ami-12345", details.AMI)
	assert.Len(t, details.Tags, 2)
	assert.Equal(t, "test-instance", details.Tags["Name"])
	assert.Len(t, details.SecurityGroups, 1)
	assert.Equal(t, "sg-12345", details.SecurityGroups[0])
	assert.Equal(t, "subnet-12345", details.SubnetID)
}

func TestGetInstanceDetails_InstanceNotFound(t *testing.T) {
	mockClient := mocks.NewEC2ClientAPI(t)

	// nonexistent instance ID
	instanceID := "i-nonexistent"

	emptyResponse := &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{},
	}

	mockClient.On("DescribeInstances",
		mock.Anything,
		mock.MatchedBy(func(input *ec2.DescribeInstancesInput) bool {
			return len(input.InstanceIds) == 1 && input.InstanceIds[0] == instanceID
		}),
	).Return(emptyResponse, nil)

	service := NewInstanceServiceWithClient(mockClient)
	details, err := service.GetInstanceDetails(context.Background(), instanceID)

	assert.Error(t, err)
	assert.Nil(t, details)
	assert.Contains(t, err.Error(), "EC2 instance not found")
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
	details, err := service.GetInstanceDetails(context.Background(), instanceID)

	// Should return an error
	assert.Error(t, err)
	assert.Nil(t, details)
	assert.Contains(t, err.Error(), expectedError.Error())
}
