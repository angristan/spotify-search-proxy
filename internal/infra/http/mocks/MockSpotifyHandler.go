// Code generated by mockery v2.38.0. DO NOT EDIT.

package mocks

import (
	gin "github.com/gin-gonic/gin"
	mock "github.com/stretchr/testify/mock"
)

// MockSpotifyHandler is an autogenerated mock type for the SpotifyHandler type
type MockSpotifyHandler struct {
	mock.Mock
}

type MockSpotifyHandler_Expecter struct {
	mock *mock.Mock
}

func (_m *MockSpotifyHandler) EXPECT() *MockSpotifyHandler_Expecter {
	return &MockSpotifyHandler_Expecter{mock: &_m.Mock}
}

// Search provides a mock function with given fields: ctx
func (_m *MockSpotifyHandler) Search(ctx *gin.Context) {
	_m.Called(ctx)
}

// MockSpotifyHandler_Search_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Search'
type MockSpotifyHandler_Search_Call struct {
	*mock.Call
}

// Search is a helper method to define mock.On call
//   - ctx *gin.Context
func (_e *MockSpotifyHandler_Expecter) Search(ctx interface{}) *MockSpotifyHandler_Search_Call {
	return &MockSpotifyHandler_Search_Call{Call: _e.mock.On("Search", ctx)}
}

func (_c *MockSpotifyHandler_Search_Call) Run(run func(ctx *gin.Context)) *MockSpotifyHandler_Search_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*gin.Context))
	})
	return _c
}

func (_c *MockSpotifyHandler_Search_Call) Return() *MockSpotifyHandler_Search_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockSpotifyHandler_Search_Call) RunAndReturn(run func(*gin.Context)) *MockSpotifyHandler_Search_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockSpotifyHandler creates a new instance of MockSpotifyHandler. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockSpotifyHandler(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockSpotifyHandler {
	mock := &MockSpotifyHandler{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
