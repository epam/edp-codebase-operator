package chain

import (
	edpV1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestParseTemplateMethod_1(t *testing.T) {
	ec := &v1alpha1.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
		},
		Spec: v1alpha1.EDPComponentSpec{
			Type:    "",
			Url:     "",
			Icon:    "",
			Visible: false,
		},
	}

	objs := []runtime.Object{ec}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, ec)

	ch := PutGitlabCiFile{
		client: fake.NewFakeClient(objs...),
	}

	c := &edpV1alpha1.Codebase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeCodebaseName,
			Namespace: fakeNamespace,
		},
		Spec: edpV1alpha1.CodebaseSpec{
			Framework: util.GetStringP("maven"),
			BuildTool: "maven",
			Lang:      goLang,
			Versioning: edpV1alpha1.Versioning{
				Type: edpV1alpha1.Default,
			},
		},
	}

	assert.Error(t, ch.parseTemplate(c))
}
