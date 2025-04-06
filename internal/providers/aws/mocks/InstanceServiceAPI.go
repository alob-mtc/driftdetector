// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	context "context"
	models "driftdetector/internal/models"

	mock "github.com/stretchr/testify/mock"
)

// InstanceServiceAPI is an autogenerated mock type for the InstanceServiceAPI type
type InstanceServiceAPI struct {
	mock.Mock
}

// GetInstancesDetails provides a mock function with given fields: ctx, instanceIDs
func (_m *InstanceServiceAPI) GetInstancesDetails(ctx context.Context, instanceIDs []string) ([]*models.InstanceDetails, error) {
	ret := _m.Called(ctx, instanceIDs)

	if len(ret) == 0 {
		panic("no return value specified for GetInstancesDetails")
	}

	var r0 []*models.InstanceDetails
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]*models.InstanceDetails, error)); ok {
		return rf(ctx, instanceIDs)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []string) []*models.InstanceDetails); ok {
		r0 = rf(ctx, instanceIDs)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*models.InstanceDetails)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []string) error); ok {
		r1 = rf(ctx, instanceIDs)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewInstanceServiceAPI creates a new instance of InstanceServiceAPI. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewInstanceServiceAPI(t interface {
	mock.TestingT
	Cleanup(func())
}) *InstanceServiceAPI {
	mock := &InstanceServiceAPI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
