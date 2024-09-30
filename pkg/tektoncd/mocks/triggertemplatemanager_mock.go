// Code generated by mockery v2.43.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockTriggerTemplateManager is an autogenerated mock type for the TriggerTemplateManager type
type MockTriggerTemplateManager struct {
	mock.Mock
}

type MockTriggerTemplateManager_Expecter struct {
	mock *mock.Mock
}

func (_m *MockTriggerTemplateManager) EXPECT() *MockTriggerTemplateManager_Expecter {
	return &MockTriggerTemplateManager_Expecter{mock: &_m.Mock}
}

// CreatePendingPipelineRun provides a mock function with given fields: ctx, ns, cdStageDeployName, rawPipeRun, appPayload, stage, pipeline, clusterSecret
func (_m *MockTriggerTemplateManager) CreatePendingPipelineRun(ctx context.Context, ns string, cdStageDeployName string, rawPipeRun []byte, appPayload []byte, stage []byte, pipeline []byte, clusterSecret []byte) error {
	ret := _m.Called(ctx, ns, cdStageDeployName, rawPipeRun, appPayload, stage, pipeline, clusterSecret)

	if len(ret) == 0 {
		panic("no return value specified for CreatePendingPipelineRun")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, []byte, []byte, []byte, []byte, []byte) error); ok {
		r0 = rf(ctx, ns, cdStageDeployName, rawPipeRun, appPayload, stage, pipeline, clusterSecret)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockTriggerTemplateManager_CreatePendingPipelineRun_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreatePendingPipelineRun'
type MockTriggerTemplateManager_CreatePendingPipelineRun_Call struct {
	*mock.Call
}

// CreatePendingPipelineRun is a helper method to define mock.On call
//   - ctx context.Context
//   - ns string
//   - cdStageDeployName string
//   - rawPipeRun []byte
//   - appPayload []byte
//   - stage []byte
//   - pipeline []byte
//   - clusterSecret []byte
func (_e *MockTriggerTemplateManager_Expecter) CreatePendingPipelineRun(ctx interface{}, ns interface{}, cdStageDeployName interface{}, rawPipeRun interface{}, appPayload interface{}, stage interface{}, pipeline interface{}, clusterSecret interface{}) *MockTriggerTemplateManager_CreatePendingPipelineRun_Call {
	return &MockTriggerTemplateManager_CreatePendingPipelineRun_Call{Call: _e.mock.On("CreatePendingPipelineRun", ctx, ns, cdStageDeployName, rawPipeRun, appPayload, stage, pipeline, clusterSecret)}
}

func (_c *MockTriggerTemplateManager_CreatePendingPipelineRun_Call) Run(run func(ctx context.Context, ns string, cdStageDeployName string, rawPipeRun []byte, appPayload []byte, stage []byte, pipeline []byte, clusterSecret []byte)) *MockTriggerTemplateManager_CreatePendingPipelineRun_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].([]byte), args[4].([]byte), args[5].([]byte), args[6].([]byte), args[7].([]byte))
	})
	return _c
}

func (_c *MockTriggerTemplateManager_CreatePendingPipelineRun_Call) Return(_a0 error) *MockTriggerTemplateManager_CreatePendingPipelineRun_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockTriggerTemplateManager_CreatePendingPipelineRun_Call) RunAndReturn(run func(context.Context, string, string, []byte, []byte, []byte, []byte, []byte) error) *MockTriggerTemplateManager_CreatePendingPipelineRun_Call {
	_c.Call.Return(run)
	return _c
}

// CreatePipelineRun provides a mock function with given fields: ctx, ns, cdStageDeployName, rawPipeRun, appPayload, stage, pipeline, clusterSecret
func (_m *MockTriggerTemplateManager) CreatePipelineRun(ctx context.Context, ns string, cdStageDeployName string, rawPipeRun []byte, appPayload []byte, stage []byte, pipeline []byte, clusterSecret []byte) error {
	ret := _m.Called(ctx, ns, cdStageDeployName, rawPipeRun, appPayload, stage, pipeline, clusterSecret)

	if len(ret) == 0 {
		panic("no return value specified for CreatePipelineRun")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, []byte, []byte, []byte, []byte, []byte) error); ok {
		r0 = rf(ctx, ns, cdStageDeployName, rawPipeRun, appPayload, stage, pipeline, clusterSecret)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockTriggerTemplateManager_CreatePipelineRun_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreatePipelineRun'
type MockTriggerTemplateManager_CreatePipelineRun_Call struct {
	*mock.Call
}

// CreatePipelineRun is a helper method to define mock.On call
//   - ctx context.Context
//   - ns string
//   - cdStageDeployName string
//   - rawPipeRun []byte
//   - appPayload []byte
//   - stage []byte
//   - pipeline []byte
//   - clusterSecret []byte
func (_e *MockTriggerTemplateManager_Expecter) CreatePipelineRun(ctx interface{}, ns interface{}, cdStageDeployName interface{}, rawPipeRun interface{}, appPayload interface{}, stage interface{}, pipeline interface{}, clusterSecret interface{}) *MockTriggerTemplateManager_CreatePipelineRun_Call {
	return &MockTriggerTemplateManager_CreatePipelineRun_Call{Call: _e.mock.On("CreatePipelineRun", ctx, ns, cdStageDeployName, rawPipeRun, appPayload, stage, pipeline, clusterSecret)}
}

func (_c *MockTriggerTemplateManager_CreatePipelineRun_Call) Run(run func(ctx context.Context, ns string, cdStageDeployName string, rawPipeRun []byte, appPayload []byte, stage []byte, pipeline []byte, clusterSecret []byte)) *MockTriggerTemplateManager_CreatePipelineRun_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].([]byte), args[4].([]byte), args[5].([]byte), args[6].([]byte), args[7].([]byte))
	})
	return _c
}

func (_c *MockTriggerTemplateManager_CreatePipelineRun_Call) Return(_a0 error) *MockTriggerTemplateManager_CreatePipelineRun_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockTriggerTemplateManager_CreatePipelineRun_Call) RunAndReturn(run func(context.Context, string, string, []byte, []byte, []byte, []byte, []byte) error) *MockTriggerTemplateManager_CreatePipelineRun_Call {
	_c.Call.Return(run)
	return _c
}

// GetRawResourceFromTriggerTemplate provides a mock function with given fields: ctx, triggerTemplateName, ns
func (_m *MockTriggerTemplateManager) GetRawResourceFromTriggerTemplate(ctx context.Context, triggerTemplateName string, ns string) ([]byte, error) {
	ret := _m.Called(ctx, triggerTemplateName, ns)

	if len(ret) == 0 {
		panic("no return value specified for GetRawResourceFromTriggerTemplate")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) ([]byte, error)); ok {
		return rf(ctx, triggerTemplateName, ns)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) []byte); ok {
		r0 = rf(ctx, triggerTemplateName, ns)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, triggerTemplateName, ns)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockTriggerTemplateManager_GetRawResourceFromTriggerTemplate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetRawResourceFromTriggerTemplate'
type MockTriggerTemplateManager_GetRawResourceFromTriggerTemplate_Call struct {
	*mock.Call
}

// GetRawResourceFromTriggerTemplate is a helper method to define mock.On call
//   - ctx context.Context
//   - triggerTemplateName string
//   - ns string
func (_e *MockTriggerTemplateManager_Expecter) GetRawResourceFromTriggerTemplate(ctx interface{}, triggerTemplateName interface{}, ns interface{}) *MockTriggerTemplateManager_GetRawResourceFromTriggerTemplate_Call {
	return &MockTriggerTemplateManager_GetRawResourceFromTriggerTemplate_Call{Call: _e.mock.On("GetRawResourceFromTriggerTemplate", ctx, triggerTemplateName, ns)}
}

func (_c *MockTriggerTemplateManager_GetRawResourceFromTriggerTemplate_Call) Run(run func(ctx context.Context, triggerTemplateName string, ns string)) *MockTriggerTemplateManager_GetRawResourceFromTriggerTemplate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockTriggerTemplateManager_GetRawResourceFromTriggerTemplate_Call) Return(_a0 []byte, _a1 error) *MockTriggerTemplateManager_GetRawResourceFromTriggerTemplate_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockTriggerTemplateManager_GetRawResourceFromTriggerTemplate_Call) RunAndReturn(run func(context.Context, string, string) ([]byte, error)) *MockTriggerTemplateManager_GetRawResourceFromTriggerTemplate_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockTriggerTemplateManager creates a new instance of MockTriggerTemplateManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockTriggerTemplateManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockTriggerTemplateManager {
	mock := &MockTriggerTemplateManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
