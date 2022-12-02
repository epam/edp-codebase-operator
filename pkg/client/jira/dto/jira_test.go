package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertSpecToJiraServer(t *testing.T) {
	t.Parallel()

	type args struct {
		url      string
		user     string
		password string
	}

	tests := []struct {
		name string
		args args
		want JiraServer
	}{
		{
			name: "should return correct jira server",
			args: args{
				url:      "test-url",
				user:     "test-user",
				password: "test-pass",
			},
			want: JiraServer{
				ApiUrl: "test-url",
				User:   "test-user",
				Pwd:    "test-pass",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ConvertSpecToJiraServer(tt.args.url, tt.args.user, tt.args.password)

			assert.Equal(t, tt.want, got)
		})
	}
}
