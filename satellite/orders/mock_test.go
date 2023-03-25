// Code generated by MockGen. DO NOT EDIT.
// Source: storx/storx/satellite/orders (interfaces: Overlay)

// Package orders is a generated GoMock package.
package orders

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	storx "common/storx"
	overlay "storx/satellite/overlay"
)

// MockOverlayForOrders is a mock of Overlay interface.
type MockOverlayForOrders struct {
	ctrl     *gomock.Controller
	recorder *MockOverlayForOrdersMockRecorder
}

// MockOverlayForOrdersMockRecorder is the mock recorder for MockOverlayForOrders.
type MockOverlayForOrdersMockRecorder struct {
	mock *MockOverlayForOrders
}

// NewMockOverlayForOrders creates a new mock instance.
func NewMockOverlayForOrders(ctrl *gomock.Controller) *MockOverlayForOrders {
	mock := &MockOverlayForOrders{ctrl: ctrl}
	mock.recorder = &MockOverlayForOrdersMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOverlayForOrders) EXPECT() *MockOverlayForOrdersMockRecorder {
	return m.recorder
}

// CachedGetOnlineNodesForGet mocks base method.
func (m *MockOverlayForOrders) CachedGetOnlineNodesForGet(arg0 context.Context, arg1 []storx.NodeID) (map[storx.NodeID]*overlay.SelectedNode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CachedGetOnlineNodesForGet", arg0, arg1)
	ret0, _ := ret[0].(map[storx.NodeID]*overlay.SelectedNode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CachedGetOnlineNodesForGet indicates an expected call of CachedGetOnlineNodesForGet.
func (mr *MockOverlayForOrdersMockRecorder) CachedGetOnlineNodesForGet(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CachedGetOnlineNodesForGet", reflect.TypeOf((*MockOverlayForOrders)(nil).CachedGetOnlineNodesForGet), arg0, arg1)
}

// Get mocks base method.
func (m *MockOverlayForOrders) Get(arg0 context.Context, arg1 storx.NodeID) (*overlay.NodeDossier, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(*overlay.NodeDossier)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockOverlayForOrdersMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockOverlayForOrders)(nil).Get), arg0, arg1)
}

// GetOnlineNodesForAuditRepair mocks base method.
func (m *MockOverlayForOrders) GetOnlineNodesForAuditRepair(arg0 context.Context, arg1 []storx.NodeID) (map[storx.NodeID]*overlay.NodeReputation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOnlineNodesForAuditRepair", arg0, arg1)
	ret0, _ := ret[0].(map[storx.NodeID]*overlay.NodeReputation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOnlineNodesForAuditRepair indicates an expected call of GetOnlineNodesForAuditRepair.
func (mr *MockOverlayForOrdersMockRecorder) GetOnlineNodesForAuditRepair(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOnlineNodesForAuditRepair", reflect.TypeOf((*MockOverlayForOrders)(nil).GetOnlineNodesForAuditRepair), arg0, arg1)
}

// IsOnline mocks base method.
func (m *MockOverlayForOrders) IsOnline(arg0 *overlay.NodeDossier) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsOnline", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsOnline indicates an expected call of IsOnline.
func (mr *MockOverlayForOrdersMockRecorder) IsOnline(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsOnline", reflect.TypeOf((*MockOverlayForOrders)(nil).IsOnline), arg0)
}