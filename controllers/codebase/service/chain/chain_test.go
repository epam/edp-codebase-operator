package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain/handler"
	handlermocks "github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain/handler/mocks"
)

func Test_chain_Use(t *testing.T) {
	t.Parallel()

	type fields struct {
		handlers []handler.CodebaseHandler
	}

	type args struct {
		handlers []handler.CodebaseHandler
	}

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "should successfully add handlers to the chain",
			fields: fields{
				handlers: []handler.CodebaseHandler{},
			},
			args: args{
				handlers: []handler.CodebaseHandler{handlermocks.NewMockCodebaseHandler(t)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ch := &chain{
				handlers: tt.fields.handlers,
			}

			want := make([]handler.CodebaseHandler, 0, len(tt.fields.handlers)+len(tt.args.handlers))
			want = append(want, tt.fields.handlers...)
			want = append(want, tt.args.handlers...)

			ch.Use(tt.args.handlers...)

			assert.Equal(t, want, ch.handlers)
		})
	}
}
