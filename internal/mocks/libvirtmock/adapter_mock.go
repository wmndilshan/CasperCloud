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

// GracefulShutdown mocks base method.
func (m *MockAdapter) GracefulShutdown(ctx context.Context, instanceName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GracefulShutdown", ctx, instanceName)
	ret0, _ := ret[0].(error)
	return ret0
}

// GracefulShutdown indicates an expected call of GracefulShutdown.
func (mr *MockAdapterMockRecorder) GracefulShutdown(ctx, instanceName any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GracefulShutdown", reflect.TypeOf((*MockAdapter)(nil).GracefulShutdown), ctx, instanceName)
}

// HardPowerOff mocks base method.
func (m *MockAdapter) HardPowerOff(ctx context.Context, instanceName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HardPowerOff", ctx, instanceName)
	ret0, _ := ret[0].(error)
	return ret0
}

// HardPowerOff indicates an expected call of HardPowerOff.
func (mr *MockAdapterMockRecorder) HardPowerOff(ctx, instanceName any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HardPowerOff", reflect.TypeOf((*MockAdapter)(nil).HardPowerOff), ctx, instanceName)
}

// ListDomainRunning mocks base method.
func (m *MockAdapter) ListDomainRunning(ctx context.Context) (map[string]bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListDomainRunning", ctx)
	ret0, _ := ret[0].(map[string]bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDomainRunning indicates an expected call of ListDomainRunning.
func (mr *MockAdapterMockRecorder) ListDomainRunning(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDomainRunning", reflect.TypeOf((*MockAdapter)(nil).ListDomainRunning), ctx)
}

// GetInstanceStats mocks base method.
func (m *MockAdapter) GetInstanceStats(ctx context.Context, instanceName string) (*libvirt.InstanceMetrics, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInstanceStats", ctx, instanceName)
	ret0, _ := ret[0].(*libvirt.InstanceMetrics)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInstanceStats indicates an expected call of GetInstanceStats.
func (mr *MockAdapterMockRecorder) GetInstanceStats(ctx, instanceName any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInstanceStats", reflect.TypeOf((*MockAdapter)(nil).GetInstanceStats), ctx, instanceName)
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

// NextVirtioDiskTarget mocks base method.
func (m *MockAdapter) NextVirtioDiskTarget(ctx context.Context, instanceName string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextVirtioDiskTarget", ctx, instanceName)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NextVirtioDiskTarget indicates an expected call of NextVirtioDiskTarget.
func (mr *MockAdapterMockRecorder) NextVirtioDiskTarget(ctx, instanceName any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextVirtioDiskTarget", reflect.TypeOf((*MockAdapter)(nil).NextVirtioDiskTarget), ctx, instanceName)
}

// AttachVolume mocks base method.
func (m *MockAdapter) AttachVolume(ctx context.Context, instanceName, absPath, targetDev string, live bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AttachVolume", ctx, instanceName, absPath, targetDev, live)
	ret0, _ := ret[0].(error)
	return ret0
}

// AttachVolume indicates an expected call of AttachVolume.
func (mr *MockAdapterMockRecorder) AttachVolume(ctx, instanceName, absPath, targetDev, live any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AttachVolume", reflect.TypeOf((*MockAdapter)(nil).AttachVolume), ctx, instanceName, absPath, targetDev, live)
}

// DetachVolume mocks base method.
func (m *MockAdapter) DetachVolume(ctx context.Context, instanceName, absPath, targetDev string, live bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DetachVolume", ctx, instanceName, absPath, targetDev, live)
	ret0, _ := ret[0].(error)
	return ret0
}

// DetachVolume indicates an expected call of DetachVolume.
func (mr *MockAdapterMockRecorder) DetachVolume(ctx, instanceName, absPath, targetDev, live any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DetachVolume", reflect.TypeOf((*MockAdapter)(nil).DetachVolume), ctx, instanceName, absPath, targetDev, live)
}

// CreateInternalSnapshot mocks base method.
func (m *MockAdapter) CreateInternalSnapshot(ctx context.Context, instanceName, snapLibvirtName, description string, domainRunning bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateInternalSnapshot", ctx, instanceName, snapLibvirtName, description, domainRunning)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateInternalSnapshot indicates an expected call of CreateInternalSnapshot.
func (mr *MockAdapterMockRecorder) CreateInternalSnapshot(ctx, instanceName, snapLibvirtName, description, domainRunning any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateInternalSnapshot", reflect.TypeOf((*MockAdapter)(nil).CreateInternalSnapshot), ctx, instanceName, snapLibvirtName, description, domainRunning)
}

// RevertToInternalSnapshot mocks base method.
func (m *MockAdapter) RevertToInternalSnapshot(ctx context.Context, instanceName, snapLibvirtName string, domainWasRunningAtSnapshot, domainCurrentlyRunning bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RevertToInternalSnapshot", ctx, instanceName, snapLibvirtName, domainWasRunningAtSnapshot, domainCurrentlyRunning)
	ret0, _ := ret[0].(error)
	return ret0
}

// RevertToInternalSnapshot indicates an expected call of RevertToInternalSnapshot.
func (mr *MockAdapterMockRecorder) RevertToInternalSnapshot(ctx, instanceName, snapLibvirtName, domainWasRunningAtSnapshot, domainCurrentlyRunning any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RevertToInternalSnapshot", reflect.TypeOf((*MockAdapter)(nil).RevertToInternalSnapshot), ctx, instanceName, snapLibvirtName, domainWasRunningAtSnapshot, domainCurrentlyRunning)
}

// DeleteInternalSnapshot mocks base method.
func (m *MockAdapter) DeleteInternalSnapshot(ctx context.Context, instanceName, snapLibvirtName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteInternalSnapshot", ctx, instanceName, snapLibvirtName)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteInternalSnapshot indicates an expected call of DeleteInternalSnapshot.
func (mr *MockAdapterMockRecorder) DeleteInternalSnapshot(ctx, instanceName, snapLibvirtName any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteInternalSnapshot", reflect.TypeOf((*MockAdapter)(nil).DeleteInternalSnapshot), ctx, instanceName, snapLibvirtName)
}
