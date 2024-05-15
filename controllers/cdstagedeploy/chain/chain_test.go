package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy/chain/mocks"
)

func Test_chain_ServeRequest(t *testing.T) {
	tests := []struct {
		name     string
		c        *codebaseApi.CDStageDeploy
		handlers func(t *testing.T) []CDStageDeployHandler
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name: "should handle request successfully",
			c:    &codebaseApi.CDStageDeploy{},
			handlers: func(t *testing.T) []CDStageDeployHandler {
				h := mocks.NewMockCDStageDeployHandler(t)
				h.On("ServeRequest", mock.Anything, mock.Anything).
					Return(nil)

				return []CDStageDeployHandler{h}
			},
			wantErr: require.NoError,
		},
		{
			name: "should fail to handle request",
			c:    &codebaseApi.CDStageDeploy{},
			handlers: func(t *testing.T) []CDStageDeployHandler {
				h := mocks.NewMockCDStageDeployHandler(t)
				h.On("ServeRequest", mock.Anything, mock.Anything).
					Return(errors.New("failed to handle request"))

				return []CDStageDeployHandler{h}
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to serve handler")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &chain{
				handlers: tt.handlers(t),
			}

			tt.wantErr(t, ch.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.c))
		})
	}
}
