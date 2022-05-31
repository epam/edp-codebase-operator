package handler

import (
	"github.com/stretchr/testify/mock"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) ServeRequest(c *codebaseApi.Codebase) error {
	return m.Called(c).Error(0)
}
