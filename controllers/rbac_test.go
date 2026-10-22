package controllers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const managerUser = "manager"

// OwnerReferencesPermissionEnforcement, on by default on OpenShift, admits a blockOwnerDeletion
// ownerReference only from a user who may update the owner's finalizers, and an ownerReference
// change on an existing object only from a user who may delete that object.
func TestManagerRole_AdmitsControllerReferences(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, pipelineApi.AddToScheme(scheme))

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join("testdata", "stages.v2.edp.epam.com.yaml"),
		},
		ErrorIfCRDPathMissing: true,
	}
	testEnv.ControlPlane.GetAPIServer().Configure().
		Append("enable-admission-plugins", "OwnerReferencesPermissionEnforcement")

	cfg, err := testEnv.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, testEnv.Stop())
	})

	admin, err := client.New(cfg, client.Options{Scheme: scheme})
	require.NoError(t, err)

	user, err := testEnv.AddUser(envtest.User{Name: managerUser}, nil)
	require.NoError(t, err)

	manager, err := client.New(user.Config(), client.Options{Scheme: scheme})
	require.NoError(t, err)

	// Each Role is bound to the manager user in a namespace of its own.
	roles := []struct {
		namespace string
		path      string
	}{
		{namespace: "config-rbac", path: filepath.Join("..", "config", "rbac", "role.yaml")},
		{namespace: "chart", path: filepath.Join("..", "deploy-templates", "templates", "role.yaml")},
	}

	for _, role := range roles {
		t.Run(role.namespace, func(t *testing.T) {
			bindRole(t, admin, manager, role.namespace, roleRules(t, role.path))
			requireControllerReferences(t, admin, manager, scheme, role.namespace)
		})
	}
}

// requireControllerReferences writes, as the manager user, each controller reference the controllers set.
func requireControllerReferences(t *testing.T, admin, manager client.Client, scheme *runtime.Scheme, ns string) {
	t.Helper()

	stage := &pipelineApi.Stage{ObjectMeta: objectMeta(ns, "app-dev")}
	codebase := &codebaseApi.Codebase{
		ObjectMeta: objectMeta(ns, "app"),
		Spec: codebaseApi.CodebaseSpec{
			Lang:          "go",
			Framework:     "gin",
			BuildTool:     "go",
			Strategy:      codebaseApi.Create,
			Type:          "application",
			GitServer:     "github",
			GitUrlPath:    "/org/app",
			DefaultBranch: "main",
			Versioning:    codebaseApi.Versioning{Type: codebaseApi.VersioningTypDefault},
			CiTool:        util.CITekton,
		},
	}
	codebaseBranch := &codebaseApi.CodebaseBranch{
		ObjectMeta: objectMeta(ns, "app-main"),
		Spec:       codebaseApi.CodebaseBranchSpec{CodebaseName: "app", BranchName: "main"},
	}
	gitServer := &codebaseApi.GitServer{
		ObjectMeta: objectMeta(ns, "github"),
		Spec: codebaseApi.GitServerSpec{
			GitHost:          "github.com",
			HttpsPort:        443,
			SshPort:          22,
			NameSshKeySecret: "ci-github",
		},
	}
	uncoveredOwner := &corev1.ConfigMap{ObjectMeta: objectMeta(ns, "uncovered-owner")}

	for _, owner := range []client.Object{stage, codebase, codebaseBranch, gitServer, uncoveredOwner} {
		require.NoError(t, admin.Create(t.Context(), owner))
	}

	t.Run("rejects an owner without update on its finalizers", func(t *testing.T) {
		dependent := &codebaseApi.CodebaseImageStream{
			ObjectMeta: objectMeta(ns, "rejected"),
			Spec:       codebaseApi.CodebaseImageStreamSpec{Codebase: "app", ImageName: "registry.example.com/app"},
		}
		require.NoError(t, controllerutil.SetControllerReference(uncoveredOwner, dependent, scheme))

		err := manager.Create(t.Context(), dependent)
		require.True(t, k8sErrors.IsForbidden(err), "want Forbidden, got %v", err)
		require.ErrorContains(t, err, "blockOwnerDeletion")
	})

	// One row per owner relation a controller writes with SetControllerReference;
	// a controller that starts writing a new one adds its row here.
	tests := []struct {
		name      string
		owner     client.Object
		dependent client.Object
		// existing dependents get the controller reference by update.
		existing bool
	}{
		{
			name:  "Stage controls a new CDStageDeploy",
			owner: stage,
			dependent: &codebaseApi.CDStageDeploy{
				ObjectMeta: objectMeta(ns, "app-dev"),
				Spec: codebaseApi.CDStageDeploySpec{
					Pipeline: "app",
					Stage:    "dev",
					Tag:      codebaseApi.CodebaseTag{Codebase: "app", Tag: "0.1.0"},
					Tags:     []codebaseApi.CodebaseTag{{Codebase: "app", Tag: "0.1.0"}},
				},
			},
		},
		{
			name:  "CodebaseBranch controls a new CodebaseImageStream",
			owner: codebaseBranch,
			dependent: &codebaseApi.CodebaseImageStream{
				ObjectMeta: objectMeta(ns, "app-main"),
				Spec:       codebaseApi.CodebaseImageStreamSpec{Codebase: "app", ImageName: "registry.example.com/app"},
			},
		},
		{
			// Also stands for the EventListener, HTTPRoute and Route the GitServer controller creates:
			// for a new object the plugin checks only the owner's finalizers.
			name:  "GitServer controls a new Ingress",
			owner: gitServer,
			dependent: &networkingv1.Ingress{
				ObjectMeta: objectMeta(ns, "event-listener-github"),
				Spec: networkingv1.IngressSpec{
					DefaultBackend: &networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: "el-github",
							Port: networkingv1.ServiceBackendPort{Number: 8080},
						},
					},
				},
			},
		},
		{
			name:     "Codebase controls an existing CodebaseBranch",
			owner:    codebase,
			existing: true,
			dependent: &codebaseApi.CodebaseBranch{
				ObjectMeta: objectMeta(ns, "app-feature"),
				Spec:       codebaseApi.CodebaseBranchSpec{CodebaseName: "app", BranchName: "feature"},
			},
		},
		{
			name:     "Codebase controls an existing JiraIssueMetadata",
			owner:    codebase,
			existing: true,
			dependent: &codebaseApi.JiraIssueMetadata{
				ObjectMeta: objectMeta(ns, "app-ticket"),
				Spec:       codebaseApi.JiraIssueMetadataSpec{CodebaseName: "app"},
			},
		},
		{
			name:      "GitServer controls an existing Secret",
			owner:     gitServer,
			existing:  true,
			dependent: &corev1.Secret{ObjectMeta: objectMeta(ns, "ci-github")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.existing {
				require.NoError(t, admin.Create(t.Context(), tt.dependent))
			}

			require.NoError(t, controllerutil.SetControllerReference(tt.owner, tt.dependent, scheme))

			if tt.existing {
				require.NoError(t, manager.Update(t.Context(), tt.dependent))

				return
			}

			require.NoError(t, manager.Create(t.Context(), tt.dependent))
		})
	}
}

// roleRules returns the rules of the Role in path. Every line holding a Helm expression is dropped:
// in the chart Role that removes the if/else/end markers, so the rules of both platform branches are kept.
func roleRules(t *testing.T, path string) []rbacv1.PolicyRule {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var kept []string

	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, "{{") {
			kept = append(kept, line)
		}
	}

	role := &rbacv1.Role{}
	_, _, err = clientgoscheme.Codecs.UniversalDeserializer().Decode([]byte(strings.Join(kept, "\n")), nil, role)
	require.NoError(t, err)

	return role.Rules
}

// bindRole binds rules to the manager user in a new namespace.
func bindRole(t *testing.T, admin, manager client.Client, ns string, rules []rbacv1.PolicyRule) {
	t.Helper()

	require.NotEmpty(t, rules)
	require.NoError(t, admin.Create(t.Context(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	require.NoError(t, admin.Create(t.Context(), &rbacv1.Role{ObjectMeta: objectMeta(ns, managerUser), Rules: rules}))
	require.NoError(t, admin.Create(t.Context(), &rbacv1.RoleBinding{
		ObjectMeta: objectMeta(ns, managerUser),
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: managerUser},
		Subjects:   []rbacv1.Subject{{APIGroup: rbacv1.GroupName, Kind: rbacv1.UserKind, Name: managerUser}},
	}))

	require.Eventually(t, func() bool {
		return manager.List(t.Context(), &codebaseApi.CodebaseList{}, client.InNamespace(ns)) == nil
	}, 10*time.Second, 100*time.Millisecond, "the RBAC authorizer did not serve the binding")
}

func objectMeta(ns, name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: ns}
}
