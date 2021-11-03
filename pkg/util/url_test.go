package util

import (
	"testing"

	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestBuildRepoUrl_DatabaseIsNil(t *testing.T) {
	expectedURL := "https://github.com/epmd-edp/java-maven-java11.git"
	framework := "java11"
	spec := v1alpha1.CodebaseSpec{
		Lang:      "Java",
		BuildTool: "Maven",
		Type:      "application",
		Framework: &framework,
	}
	u := BuildRepoUrl(spec)
	assert.Equal(t, expectedURL, u)
}

func TestBuildRepoUrl_FrameworkIsNil(t *testing.T) {
	expectedURL := "https://github.com/epmd-edp/javascript-npm-react.git"
	spec := v1alpha1.CodebaseSpec{
		Lang:      "javascript",
		BuildTool: "npm",
		Type:      "library",
	}
	u := BuildRepoUrl(spec)
	assert.Equal(t, expectedURL, u)
}

func Test_tryGetRepoUrl_ShouldPass(t *testing.T) {
	c := v1alpha1.CodebaseSpec{
		Repository: &v1alpha1.Repository{
			Url: "test",
		},
	}
	s, err := tryGetRepoUrl(c)
	assert.NoError(t, err)
	assert.Equal(t, &c.Repository.Url, s)
}

func Test_tryGetRepoUrl_ShouldFail(t *testing.T) {
	c := v1alpha1.CodebaseSpec{}
	s, err := tryGetRepoUrl(c)
	assert.Error(t, err)
	assert.Nil(t, s)
}

func TestGetRepoUrl_ShouldPass(t *testing.T) {
	f := "java11"
	r := "https://github.com/epmd-edp/java11-maven-java11.git"
	c := v1alpha1.Codebase{
		Spec: v1alpha1.CodebaseSpec{
			Strategy:  v1alpha1.Create,
			Lang:      "java11",
			BuildTool: "maven",
			Framework: &f,
		},
	}
	url, err := GetRepoUrl(&c)
	assert.NoError(t, err)
	assert.Equal(t, url, &r)
}

func TestGetRepoUrl_CloneShouldPass(t *testing.T) {
	r := "link"
	c := v1alpha1.Codebase{
		Spec: v1alpha1.CodebaseSpec{
			Strategy: v1alpha1.Clone,
			Repository: &v1alpha1.Repository{
				Url: "link",
			},
		},
	}
	url, err := GetRepoUrl(&c)
	assert.NoError(t, err)
	assert.Equal(t, url, &r)
}
