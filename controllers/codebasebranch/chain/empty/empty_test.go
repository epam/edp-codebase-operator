package empty

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestChain_ServeRequest(t *testing.T) {
	t.Parallel()

	type fields struct {
		logMessage  string
		returnError bool
	}

	type args struct {
		in0 *codebaseApi.CodebaseBranch
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return no error",
			fields: fields{
				logMessage:  "log message",
				returnError: false,
			},
			args: args{
				in0: nil,
			},
			wantErr: assert.NoError,
		},
		{
			name: "should return an error",
			fields: fields{
				logMessage:  "log message",
				returnError: true,
			},
			args: args{
				in0: nil,
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := Chain{
				logMessage:  tt.fields.logMessage,
				returnError: tt.fields.returnError,
			}

			gotErr := e.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.args.in0)

			tt.wantErr(t, gotErr)
		})
	}
}

func TestMakeChain(t *testing.T) {
	t.Parallel()

	type args struct {
		logMessage  string
		returnError bool
	}

	tests := []struct {
		name string
		args args
		want Chain
	}{
		{
			name: "should create chain",
			args: args{
				logMessage:  "chain log",
				returnError: true,
			},
			want: Chain{
				logMessage:  "chain log",
				returnError: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := MakeChain(tt.args.logMessage, tt.args.returnError)

			assert.Equal(t, tt.want, got)
		})
	}
}
