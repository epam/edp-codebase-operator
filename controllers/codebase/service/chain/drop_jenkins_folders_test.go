package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestDropJenkinsFolders_ServeRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	c := &codebaseApi.Codebase{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}

	cbl := &codebaseApi.CodebaseBranchList{
		Items: []codebaseApi.CodebaseBranch{
			{
				ObjectMeta: metaV1.ObjectMeta{
					OwnerReferences: []metaV1.OwnerReference{{
						Kind: "Codebase",
						Name: fakeName,
					}},
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
			},
		},
	}

	jfl := &jenkinsApi.JenkinsFolderList{
		Items: []jenkinsApi.JenkinsFolder{
			{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"codebase": fakeName,
					},
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
			},
			{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"codebase": "another-codebase",
					},
					Name:      "another-jf",
					Namespace: fakeNamespace,
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, cbl, &codebaseApi.CodebaseBranch{})
	scheme.AddKnownTypes(jenkinsApi.SchemeGroupVersion, &jenkinsApi.JenkinsFolder{}, jfl)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, jfl, cbl).Build()

	djf := NewDropJenkinsFolders(
		fakeCl,
	)

	err := djf.ServeRequest(ctx, c)
	assert.NoError(t, err)

	jfr := &jenkinsApi.JenkinsFolder{}
	if err := fakeCl.Get(ctx,
		types.NamespacedName{
			Name:      "another-jf",
			Namespace: fakeNamespace,
		},
		jfr); err != nil {
		t.Error("failed to get JenkinsFolder")
	}

	assert.Equal(t, jfr.Labels["codebase"], "another-codebase")

	jflr := &jenkinsApi.JenkinsFolderList{}
	if err := fakeCl.List(ctx, jflr); err != nil {
		t.Error("failed to get JenkinsFolder")
	}

	assert.Equal(t, len(jflr.Items), 1)
}

func TestDropJenkinsFolders_ServeRequest_ShouldFailCodebaseExists(t *testing.T) {
	ctx := context.Background()
	c := &codebaseApi.Codebase{
		TypeMeta: metaV1.TypeMeta{
			Kind: "Codebase",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
			UID:       "xxx",
		},
	}

	cbl := &codebaseApi.CodebaseBranchList{
		Items: []codebaseApi.CodebaseBranch{
			{
				ObjectMeta: metaV1.ObjectMeta{
					OwnerReferences: []metaV1.OwnerReference{{
						Kind: "Codebase",
						Name: fakeName,
						UID:  "xxx",
					}},
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(codebaseApi.GroupVersion, c, cbl, &codebaseApi.CodebaseBranch{})
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cbl).Build()

	djf := NewDropJenkinsFolders(
		fakeCl,
	)

	err := djf.ServeRequest(ctx, c)
	assert.ErrorIs(t, err, BranchesExistsError(err.Error()))
}
