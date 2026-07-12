package stalecheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git"
	gitmocks "github.com/epam/edp-codebase-operator/v2/pkg/git/mocks"
)

const testNamespace = "default"

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	return scheme
}

func newCodebase() *codebaseApi.Codebase {
	return &codebaseApi.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: testNamespace,
		},
		Spec: codebaseApi.CodebaseSpec{
			GitServer:     "gitlab",
			GitUrlPath:    "/owner/app",
			DefaultBranch: "main",
		},
	}
}

func newBranch(name, branchName, gitStatus string) *codebaseApi.CodebaseBranch {
	return &codebaseApi.CodebaseBranch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: codebaseApi.CodebaseBranchSpec{
			CodebaseName: "app",
			BranchName:   branchName,
		},
		Status: codebaseApi.CodebaseBranchStatus{
			Git: gitStatus,
		},
	}
}

func newGitServerWithSecret() (*codebaseApi.GitServer, *corev1.Secret) {
	gitServer := &codebaseApi.GitServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gitlab", Namespace: testNamespace},
		Spec: codebaseApi.GitServerSpec{
			GitHost:          "gitlab.example.com",
			GitProvider:      codebaseApi.GitProviderGitlab,
			NameSshKeySecret: "gitlab-secret",
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "gitlab-secret", Namespace: testNamespace},
		Data:       map[string][]byte{"token": []byte("token"), "username": []byte("user")},
	}

	return gitServer, secret
}

func newChecker(
	t *testing.T,
	k8sClient client.Client,
	gitClient gitproviderv2.Git,
	recorder record.EventRecorder,
) *Checker {
	t.Helper()

	factory := func(_ *codebaseApi.GitServer, _ *corev1.Secret) gitproviderv2.Git {
		return gitClient
	}

	mark := NewMarkAction(k8sClient, recorder)

	return NewChecker(k8sClient, testNamespace, 0, factory, mark, NewCleanupAction(k8sClient, recorder, mark))
}

func getBranch(t *testing.T, k8sClient client.Client, name string) *codebaseApi.CodebaseBranch {
	t.Helper()

	branch := &codebaseApi.CodebaseBranch{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: name}, branch))

	return branch
}

func TestChecker_MarksMissingBranchStale(t *testing.T) {
	codebase := newCodebase()
	mainBranch := newBranch("app-main", "main", codebaseApi.CodebaseBranchGitStatusBranchCreated)
	featureBranch := newBranch("app-feature", "feature", codebaseApi.CodebaseBranchGitStatusBranchCreated)
	gitServer, secret := newGitServerWithSecret()

	k8sClient := fake.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithObjects(codebase, mainBranch, featureBranch, gitServer, secret).
		WithStatusSubresource(mainBranch, featureBranch).
		Build()

	gitClient := gitmocks.NewMockGit(t)
	gitClient.On("ListRemoteBranches", mock.Anything, mock.Anything).Return([]string{"main"}, nil)

	recorder := record.NewFakeRecorder(10)

	newChecker(t, k8sClient, gitClient, recorder).sweep(context.Background())

	stale := getBranch(t, k8sClient, "app-feature")
	assert.True(t, meta.IsStatusConditionTrue(stale.Status.Conditions, codebaseApi.ConditionStale))
	assert.Equal(t, "true", stale.Labels[codebaseApi.StaleLabel])

	condition := meta.FindStatusCondition(stale.Status.Conditions, codebaseApi.ConditionStale)
	require.NotNil(t, condition)
	assert.Equal(t, codebaseApi.ReasonBranchNotFoundInGit, condition.Reason)

	select {
	case event := <-recorder.Events:
		assert.Contains(t, event, EventReasonBranchStale)
	default:
		t.Fatal("expected BranchStale event")
	}
}

func TestChecker_ClearsStaleMarkWhenBranchReappears(t *testing.T) {
	codebase := newCodebase()

	branch := newBranch("app-feature", "feature", codebaseApi.CodebaseBranchGitStatusBranchCreated)
	branch.Labels = map[string]string{codebaseApi.StaleLabel: "true"}
	branch.Status.Conditions = []metav1.Condition{{
		Type:               codebaseApi.ConditionStale,
		Status:             metav1.ConditionTrue,
		Reason:             codebaseApi.ReasonBranchNotFoundInGit,
		LastTransitionTime: metav1.Now(),
	}}

	gitServer, secret := newGitServerWithSecret()

	k8sClient := fake.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithObjects(codebase, branch, gitServer, secret).
		WithStatusSubresource(branch).
		Build()

	gitClient := gitmocks.NewMockGit(t)
	gitClient.On("ListRemoteBranches", mock.Anything, mock.Anything).Return([]string{"main", "feature"}, nil)

	recorder := record.NewFakeRecorder(10)

	newChecker(t, k8sClient, gitClient, recorder).sweep(context.Background())

	updated := getBranch(t, k8sClient, "app-feature")
	assert.False(t, meta.IsStatusConditionTrue(updated.Status.Conditions, codebaseApi.ConditionStale))
	assert.NotContains(t, updated.Labels, codebaseApi.StaleLabel)

	select {
	case event := <-recorder.Events:
		assert.Contains(t, event, EventReasonBranchStaleResolved)
	default:
		t.Fatal("expected BranchStaleResolved event")
	}
}

func TestChecker_DoesNotMarkOnGitError(t *testing.T) {
	codebase := newCodebase()
	branch := newBranch("app-feature", "feature", codebaseApi.CodebaseBranchGitStatusBranchCreated)
	gitServer, secret := newGitServerWithSecret()

	k8sClient := fake.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithObjects(codebase, branch, gitServer, secret).
		WithStatusSubresource(branch).
		Build()

	gitClient := gitmocks.NewMockGit(t)
	gitClient.On("ListRemoteBranches", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	newChecker(t, k8sClient, gitClient, record.NewFakeRecorder(10)).sweep(context.Background())

	updated := getBranch(t, k8sClient, "app-feature")
	assert.Empty(t, updated.Status.Conditions)
	assert.NotContains(t, updated.Labels, codebaseApi.StaleLabel)
}

func TestChecker_SkipsDefaultAndUnpushedBranches(t *testing.T) {
	codebase := newCodebase()
	defaultBranch := newBranch("app-main", "main", codebaseApi.CodebaseBranchGitStatusBranchCreated)
	unpushedBranch := newBranch("app-new", "new", "")
	gitServer, secret := newGitServerWithSecret()

	k8sClient := fake.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithObjects(codebase, defaultBranch, unpushedBranch, gitServer, secret).
		WithStatusSubresource(defaultBranch, unpushedBranch).
		Build()

	// Remote listing is empty: both branches are missing in git,
	// but neither is eligible for marking.
	gitClient := gitmocks.NewMockGit(t)
	gitClient.On("ListRemoteBranches", mock.Anything, mock.Anything).Return([]string{}, nil)

	newChecker(t, k8sClient, gitClient, record.NewFakeRecorder(10)).sweep(context.Background())

	for _, name := range []string{"app-main", "app-new"} {
		updated := getBranch(t, k8sClient, name)
		assert.Empty(t, updated.Status.Conditions, name)
		assert.NotContains(t, updated.Labels, codebaseApi.StaleLabel, name)
	}
}

func TestMarkAction_IdempotentWhenAlreadyStale(t *testing.T) {
	branch := newBranch("app-feature", "feature", codebaseApi.CodebaseBranchGitStatusBranchCreated)
	branch.Labels = map[string]string{codebaseApi.StaleLabel: "true"}
	branch.Status.Conditions = []metav1.Condition{{
		Type:               codebaseApi.ConditionStale,
		Status:             metav1.ConditionTrue,
		Reason:             codebaseApi.ReasonBranchNotFoundInGit,
		Message:            "Branch was not found in the git repository",
		LastTransitionTime: metav1.Now(),
	}}

	k8sClient := fake.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithObjects(branch).
		WithStatusSubresource(branch).
		Build()

	recorder := record.NewFakeRecorder(10)
	action := NewMarkAction(k8sClient, recorder)

	require.NoError(t, action.Apply(context.Background(), branch, Verdict{ExistsInGit: false}))

	select {
	case event := <-recorder.Events:
		t.Fatalf("expected no events for already-stale branch, got %s", event)
	default:
	}
}
