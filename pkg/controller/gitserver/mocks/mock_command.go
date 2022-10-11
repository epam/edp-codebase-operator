package mocks

import "github.com/stretchr/testify/mock"

type CommandMock struct {
	mock.Mock
}

func (c *CommandMock) CombinedOutput() ([]byte, error) {
	called := c.Called()
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).([]byte), nil
}
