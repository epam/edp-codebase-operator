package handler

import (
	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) ServeRequest(c *edpv1alpha1.Codebase) error {
	return m.Called(c).Error(0)
}
