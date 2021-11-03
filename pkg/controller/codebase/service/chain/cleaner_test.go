package chain

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCleaner_ShouldPass(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(dir)

	os.Setenv("WORKING_DIR", dir)

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}
	ssh := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repository-codebase-fake-name-temp",
			Namespace: fakeNamespace,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh)
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, ssh).Build()

	cl := Cleaner{
		client: fakeCl,
	}

	if err := cl.ServeRequest(c); err != nil {
		t.Error("ServeRequest failed")
	}
}

func TestCleaner_ShouldNotFailedIfSecretNotFound(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(dir)

	os.Setenv("WORKING_DIR", dir)

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}
	ssh := &coreV1.Secret{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion, ssh)
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c, ssh).Build()

	cl := Cleaner{
		client: fakeCl,
	}

	if err := cl.ServeRequest(c); err != nil {
		t.Error("ServeRequest failed")
	}
}

func TestCleaner_ShouldFail(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "codebase")
	if err != nil {
		t.Fatalf("unable to create temp directory for testing")
	}
	defer os.RemoveAll(dir)

	os.Setenv("WORKING_DIR", dir)

	c := &v1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeName,
			Namespace: fakeNamespace,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreV1.SchemeGroupVersion)
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, c)

	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c).Build()

	cl := Cleaner{
		client: fakeCl,
	}

	err = cl.ServeRequest(c)
	if err == nil {
		t.Error("ServeRequest MUST fail")
	}
	if !strings.Contains(err.Error(), "unable to delete secret repository-codebase-fake-name-temp") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
