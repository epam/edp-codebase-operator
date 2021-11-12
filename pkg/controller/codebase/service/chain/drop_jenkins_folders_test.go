package chain

import (
	"context"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	jenkinsv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDropJenkinsFolders_ServeRequest(t *testing.T) {
	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}

	cbl := &v1alpha1.CodebaseBranchList{
		Items: []v1alpha1.CodebaseBranch{
			{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{{
						Kind: "Codebase",
						Name: fakeName,
					}},
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
			},
		},
	}

	jfl := &jenkinsv1alpha1.JenkinsFolderList{
		Items: []jenkinsv1alpha1.JenkinsFolder{
			{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"codebase": fakeName,
					},
					Name:      fakeName,
					Namespace: fakeNamespace,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
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
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, cbl, &v1alpha1.CodebaseBranch{}, jfl)
	scheme.AddKnownTypes(jenkinsv1alpha1.SchemeGroupVersion, &jenkinsv1alpha1.JenkinsFolder{})
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, jfl, cbl).Build()

	djf := DropJenkinsFolders{
		k8sClient: fakeCl,
	}

	err := djf.ServeRequest(c)
	assert.NoError(t, err)

	jfr := &jenkinsv1alpha1.JenkinsFolder{}
	if err := fakeCl.Get(context.TODO(),
		types.NamespacedName{
			Name:      "another-jf",
			Namespace: fakeNamespace,
		},
		jfr); err != nil {
		t.Error("Unable to get JenkinsFolder")
	}
	assert.Equal(t, jfr.Labels["codebase"], "another-codebase")

	jflr := &jenkinsv1alpha1.JenkinsFolderList{}
	if err := fakeCl.List(context.TODO(), jflr); err != nil {
		t.Error("Unable to get JenkinsFolder")
	}
	assert.Equal(t, len(jflr.Items), 1)
}

func TestDropJenkinsFolders_ServeRequest_ShouldFailCodebaseExists(t *testing.T) {
	c := &v1alpha1.Codebase{
		TypeMeta: metav1.TypeMeta{
			Kind: "Codebase",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
			UID:       "xxx",
		},
	}

	cbl := &v1alpha1.CodebaseBranchList{
		Items: []v1alpha1.CodebaseBranch{
			{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{{
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
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c, cbl, &v1alpha1.CodebaseBranch{})
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, cbl).Build()

	djf := DropJenkinsFolders{
		k8sClient: fakeCl,
	}

	err := djf.ServeRequest(c)
	assert.ErrorIs(t, err, ErrorBranchesExists(err.Error()))
}
