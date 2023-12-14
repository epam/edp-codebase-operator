package metrics

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestCollector_Start(t *testing.T) {
	t.Parallel()

	var counter uint32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint32(&counter, 1)
		w.WriteHeader(http.StatusOK)

		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			return
		}

		res := map[string]PlatformMetrics{}
		err = json.Unmarshal(body, &res)
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, 1, len(res))
		assert.Contains(t, res, "platformMetrics")
		if assert.Equal(t, 1, len(res["platformMetrics"].CodebaseMetrics)) {
			assert.Equal(t, "go", res["platformMetrics"].CodebaseMetrics[0].Lang)
			assert.Equal(t, "gin", res["platformMetrics"].CodebaseMetrics[0].Framework)
			assert.Equal(t, "go", res["platformMetrics"].CodebaseMetrics[0].BuildTool)
			assert.Equal(t, "create", res["platformMetrics"].CodebaseMetrics[0].Strategy)
			assert.Equal(t, "application", res["platformMetrics"].CodebaseMetrics[0].Type)
			assert.Equal(t, "default", res["platformMetrics"].CodebaseMetrics[0].Versioning)
		}
		if assert.Equal(t, 1, len(res["platformMetrics"].CdPipelineMetrics)) {
			assert.Equal(t, "Auto", res["platformMetrics"].CdPipelineMetrics[0].DeploymentType)
			assert.Equal(t, 1, res["platformMetrics"].CdPipelineMetrics[0].NumberOfStages)
		}
		if assert.Equal(t, 1, len(res["platformMetrics"].GitProviders)) {
			assert.Equal(t, "github", res["platformMetrics"].GitProviders[0])
		}
		assert.True(t, res["platformMetrics"].JiraEnabled)
		assert.Equal(t, "2.0.0", res["platformMetrics"].Version)
		assert.Equal(t, "harbor", res["platformMetrics"].RegistryType)

	}))
	defer server.Close()

	ns := "default"
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, pipelineApi.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.EdpConfigMap,
					Namespace: ns,
				},
				Data: map[string]string{
					"container_registry_type": "harbor",
					"edp_version":             "2.0.0",
				},
			},
			&codebaseApi.Codebase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "codebase",
					Namespace: ns,
				},
				Spec: codebaseApi.CodebaseSpec{
					Lang:      "go",
					Framework: "gin",
					BuildTool: "go",
					Strategy:  "create",
					Type:      "application",
					Versioning: codebaseApi.Versioning{
						Type: "default",
					},
				},
			},
			&codebaseApi.GitServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-server",
					Namespace: ns,
				},
				Spec: codebaseApi.GitServerSpec{
					GitProvider: codebaseApi.GitProviderGithub,
				},
			},
			&pipelineApi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipeline",
					Namespace: ns,
				},
				Spec: pipelineApi.CDPipelineSpec{
					DeploymentType: "container",
				},
			},
			&pipelineApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "stage",
					Namespace: ns,
				},
				Spec: pipelineApi.StageSpec{
					CdPipeline:  "pipeline",
					TriggerType: "Auto",
				},
			},
			&codebaseApi.JiraServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jira-server",
					Namespace: ns,
				},
			},
		).
		Build()

	collector := NewCollector(ns, server.URL, k8sClient)
	ctx, cancel := context.WithCancel(context.Background())

	go collector.Start(ctx, 0, time.Second*2)

	time.Sleep(time.Second * 3)
	cancel()

	assert.Equal(t, uint32(2), atomic.LoadUint32(&counter))
}
