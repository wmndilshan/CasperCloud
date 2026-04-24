// Code generated manually to match go.uber.org/mock layout; interface: caspercloud/internal/libvirt.Adapter

package libvirtmock

import (
	"context"
	"reflect"

	"caspercloud/internal/libvirt"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// MockAdapter is a mock of libvirt.Adapter.
type MockAdapter struct {
	ctrl     *gomock.Controller
	recorder *MockAdapterMockRecorder
}

// MockAdapterMockRecorder is the mock recorder for MockAdapter.
type MockAdapterMockRecorder struct {
	mock *MockAdapter
}

// NewMockAdapter creates a new mock instance.
func NewMockAdapter(ctrl *gomock.Controller) *MockAdapter {
	mock := &MockAdapter{ctrl: ctrl}
	mock.recorder = &MockAdapterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAdapter) EXPECT() *MockAdapterMockRecorder {
	return m.recorder
}

// CreateVM mocks base method.
func (m *MockAdapter) CreateVM(ctx context.Context, cfg libvirt.VMConfig) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVM", ctx, cfg)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateVM indicates an expected call of CreateVM.
func (mr *MockAdapterMockRecorder) CreateVM(ctx, cfg any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVM", reflect.TypeOf((*MockAdapter)(nil).CreateVM), ctx, cfg)
}

// StartVM mocks base method.
func (m *MockAdapter) StartVM(ctx context.Context, instanceName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StartVM", ctx, instanceName)
	ret0, _ := ret[0].(error)
	return ret0
}

// StartVM indicates an expected call of StartVM.
func (mr *MockAdapterMockRecorder) StartVM(ctx, instanceName any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartVM", reflect.TypeOf((*MockAdapter)(nil).StartVM), ctx, instanceName)
}

// StopVM mocks base method.
func (m *MockAdapter) StopVM(ctx context.Context, instanceName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StopVM", ctx, instanceName)
	ret0, _ := ret[0].(error)
	return ret0
}

// StopVM indicates an expected call of StopVM.
func (mr *MockAdapterMockRecorder) StopVM(ctx, instanceName any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StopVM", reflect.TypeOf((*MockAdapter)(nil).StopVM), ctx, instanceName)
}

// RebootVM mocks base method.
func (m *MockAdapter) RebootVM(ctx context.Context, instanceName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RebootVM", ctx, instanceName)
	ret0, _ := ret[0].(error)
	return ret0
}

// RebootVM indicates an expected call of RebootVM.
func (mr *MockAdapterMockRecorder) RebootVM(ctx, instanceName any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RebootVM", reflect.TypeOf((*MockAdapter)(nil).RebootVM), ctx, instanceName)
}

// DeleteVM mocks base method.
func (m *MockAdapter) DeleteVM(ctx context.Context, instanceName string, instanceID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVM", ctx, instanceName, instanceID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVM indicates an expected call of DeleteVM.
func (mr *MockAdapterMockRecorder) DeleteVM(ctx, instanceName, instanceID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVM", reflect.TypeOf((*MockAdapter)(nil).DeleteVM), ctx, instanceName, instanceID)
}
