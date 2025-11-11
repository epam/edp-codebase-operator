package gitserver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

func TestReconcileGitServer_Reconcile_ShouldPassNotFound(t *testing.T) {
	gs := &codebaseApi.GitServer{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewMockGitServer",
			Namespace: "namespace",
		},
	}

	r := ReconcileGitServer{
		client: fakeCl,
	}

	res, err := r.Reconcile(ctrl.LoggerInto(context.Background(), logr.Discard()), req)

	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), res.RequeueAfter)
}

func TestReconcileGitServer_Reconcile_ShouldFailNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects().Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewMockGitServer",
			Namespace: "namespace",
		},
	}

	r := ReconcileGitServer{
		client: fakeCl,
	}

	res, err := r.Reconcile(ctrl.LoggerInto(context.Background(), logr.Discard()), req)

	assert.Error(t, err)

	if !strings.Contains(err.Error(), "no kind is registered for the type v1.GitServer") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	assert.Equal(t, time.Duration(0), res.RequeueAfter)
}

func TestReconcileGitServer_Reconcile_ShouldFailToGetSecret(t *testing.T) {
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewMockGitServer",
			Namespace: "namespace",
		},
		Spec: codebaseApi.GitServerSpec{
			GitHost:          "g-host",
			NameSshKeySecret: "ssh-secret",
		},
	}
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	fakeCl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gs).
		WithStatusSubresource(gs).
		Build()

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "NewMockGitServer",
			Namespace: "namespace",
		},
	}

	r := ReconcileGitServer{
		client: fakeCl,
	}

	logger := platform.NewLoggerMock()
	loggerSink, ok := logger.GetSink().(*platform.LoggerMock)
	require.True(t, ok)

	_, err := r.Reconcile(ctrl.LoggerInto(context.Background(), logger), req)

	require.NoError(t, err)
	require.Error(t, loggerSink.LastError())
	assert.Contains(t, loggerSink.LastError().Error(), "failed to get secret ssh-secret")
}

func TestReconcileGitServer_UpdateStatus_ShouldPassWithSuccess(t *testing.T) {
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewMockGitServer",
			Namespace: "namespace",
		},
		Spec: codebaseApi.GitServerSpec{
			GitHost:          "g-host",
			NameSshKeySecret: "ssh-secret",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, gs)
	fakeCl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gs).
		WithStatusSubresource(gs).
		Build()

	r := ReconcileGitServer{
		client: fakeCl,
	}

	err := r.updateGitServerStatus(
		ctrl.LoggerInto(context.Background(), logr.Discard()),
		gs,
		codebaseApi.GitServerStatus{
			Connected: true,
		},
	)

	assert.NoError(t, err)
}

func TestReconcileGitServer_UpdateStatus_Failure(t *testing.T) {
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewMockGitServer",
			Namespace: "namespace",
		},
		Spec: codebaseApi.GitServerSpec{
			GitHost:          "g-host",
			NameSshKeySecret: "ssh-secret",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, gs)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(gs).Build()

	r := ReconcileGitServer{
		client: fakeCl,
	}

	err := r.updateGitServerStatus(
		ctrl.LoggerInto(context.Background(), logr.Discard()),
		&codebaseApi.GitServer{},
		codebaseApi.GitServerStatus{
			Connected: true,
		},
	)

	assert.Error(t, err)
}

const testKey = `-----BEGIN OPENSSH PRIVATE KEY-----
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
-----END OPENSSH PRIVATE KEY-----`

func TestReconcileGitServer_ServerUnavailable(t *testing.T) {
	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewMockGitServer",
			Namespace: "namespace",
		},
		Spec: codebaseApi.GitServerSpec{
			GitHost:          "g-host",
			NameSshKeySecret: "ssh-secret",
		},
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, gs)
	err := corev1.AddToScheme(scheme)
	assert.NoError(t, err)

	secret := corev1.Secret{ObjectMeta: metaV1.ObjectMeta{Name: "ssh-secret", Namespace: gs.Namespace}, Data: map[string][]byte{
		util.PrivateSShKeyName: []byte(testKey),
	}}

	fakeCl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gs, &secret).
		WithStatusSubresource(gs).
		Build()

	logger := platform.NewLoggerMock()
	loggerSink, ok := logger.GetSink().(*platform.LoggerMock)
	require.True(t, ok)

	r := ReconcileGitServer{
		client: fakeCl,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      gs.Name,
			Namespace: gs.Namespace,
		},
	}

	res, err := r.Reconcile(ctrl.LoggerInto(context.Background(), logger), req)
	assert.NoError(t, err)
	assert.Equal(t, res.RequeueAfter, defaultRequeueTime)
	require.Error(t, loggerSink.LastError())
	assert.Contains(t, loggerSink.LastError().Error(), "failed to establish connection to Git Server")

	testedGitServer := &codebaseApi.GitServer{}
	require.NoError(t, fakeCl.Get(context.Background(), req.NamespacedName, testedGitServer))
	assert.Equal(t, "failed", testedGitServer.Status.Status)
}

func TestReconcileGitServer_InvalidSSHKey(t *testing.T) {
	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	require.NoError(t, err)

	err = codebaseApi.AddToScheme(scheme)
	require.NoError(t, err)

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewMockGitServer",
			Namespace: "namespace",
		},
		Spec: codebaseApi.GitServerSpec{
			GitHost:          "g-host",
			NameSshKeySecret: "ssh-secret",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "ssh-secret",
			Namespace: gs.Namespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte("invalid-ssh-key"),
		},
	}

	fakeCl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gs, secret).
		WithStatusSubresource(gs).
		Build()

	r := ReconcileGitServer{
		client: fakeCl,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      gs.Name,
			Namespace: gs.Namespace,
		},
	}

	logger := platform.NewLoggerMock()
	loggerSink, ok := logger.GetSink().(*platform.LoggerMock)
	require.True(t, ok)

	_, err = r.Reconcile(ctrl.LoggerInto(context.Background(), logger), req)
	require.NoError(t, err)
	require.Error(t, loggerSink.LastError())
	require.Contains(t, loggerSink.LastError().Error(), "failed to parse private key")

	gotGitServer := &codebaseApi.GitServer{}
	err = fakeCl.Get(context.Background(), req.NamespacedName, gotGitServer)
	require.NoError(t, err)
	assert.False(t, gotGitServer.Status.Connected)
}

func TestReconcileGitServer_EmptySSHKey(t *testing.T) {
	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	require.NoError(t, err)

	err = codebaseApi.AddToScheme(scheme)
	require.NoError(t, err)

	gs := &codebaseApi.GitServer{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "NewMockGitServer",
			Namespace: "namespace",
		},
		Spec: codebaseApi.GitServerSpec{
			GitHost:          "g-host",
			NameSshKeySecret: "ssh-secret",
			WebhookUrl:       "https://test-webhook-url",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "ssh-secret",
			Namespace: gs.Namespace,
		},
		Data: map[string][]byte{
			util.PrivateSShKeyName: []byte(""),
		},
	}

	fakeCl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(gs, secret).
		WithStatusSubresource(gs).
		Build()

	r := ReconcileGitServer{
		client: fakeCl,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      gs.Name,
			Namespace: gs.Namespace,
		},
	}

	logger := platform.NewLoggerMock()

	_, err = r.Reconcile(ctrl.LoggerInto(context.Background(), logger), req)
	require.NoError(t, err)

	gotGitServer := &codebaseApi.GitServer{}
	err = fakeCl.Get(context.Background(), req.NamespacedName, gotGitServer)
	require.NoError(t, err)
	assert.True(t, gotGitServer.Status.Connected)
	assert.True(t, gotGitServer.Status.IsSuccess())
}

func TestNewReconcileGitServer(t *testing.T) {
	t.Parallel()

	type args struct {
		c client.Client
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "should create new reconciler",
			args: args{
				c: fake.NewClientBuilder().
					WithScheme(runtime.NewScheme()).
					Build(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want := &ReconcileGitServer{
				client: tt.args.c,
			}

			got := NewReconcileGitServer(tt.args.c)

			assert.Equal(t, want, got)
		})
	}
}
