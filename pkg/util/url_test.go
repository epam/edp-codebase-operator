package util

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
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
