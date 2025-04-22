package chain

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraissuemetadata/chain/handler"
	jiraMock "github.com/epam/edp-codebase-operator/v2/pkg/client/jira/mocks"
)

func TestApplyTagsToIssues_ServeRequest(t *testing.T) {
	t.Parallel()

	type fields struct {
		next handler.JiraIssueMetadataHandler
	}

	type args struct {
		name      string
		namespace string
		payload   map[string]any
		ticket    string
	}

	type configs struct {
		client func(ticket string) *jiraMock.MockClient
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		configs configs
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should complete without errors",
			fields: fields{
				next: nil,
			},
			args: args{
				payload: map[string]any{
					"issuesLinks": map[string]string{"ticket": "fake-issueId", "title": "fake-title", "url": "fake-url"},
					"labels":      "fake-label",
					"some-string": "some-value",
					"some-object": map[string]int{"some-number": 5},
				},
				ticket: "fake-ticket",
			},
			configs: configs{
				client: func(ticket string) *jiraMock.MockClient {
					client := jiraMock.NewMockClient(t)

					client.On(
						"ApplyTagsToIssue",
						ticket,
						map[string]interface{}{
							"update": map[string]interface{}{
								"labels": []map[string]interface{}{
									{"add": "fake-label"},
								},
								"some-string": []map[string]interface{}{
									{
										"add": struct {
											Name string `json:"name"`
										}{
											Name: "some-value",
										},
									},
								},
							},
						},
					).Return(nil)

					return client
				},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload, err := json.Marshal(tt.args.payload)
			assert.NoError(t, err)

			client := tt.configs.client(tt.args.ticket)

			h := ApplyTagsToIssues{
				next:   tt.fields.next,
				client: client,
			}

			metadata := &codebaseApi.JiraIssueMetadata{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      tt.args.name,
					Namespace: tt.args.namespace,
				},
				Spec: codebaseApi.JiraIssueMetadataSpec{
					Payload: string(payload),
					Tickets: []string{tt.args.ticket},
				},
			}

			err = h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), metadata)

			tt.wantErr(t, err)
		})
	}
}
