package util

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"testing"
)

func TestBuildRepoUrl_DatabaseIsNil(t *testing.T) {
	expectedUrl := "https://github.com/epmd-edp/java-maven-java11.git"
	framework := "java11"
	spec := v1alpha1.CodebaseSpec{
		Lang:      "Java",
		BuildTool: "Maven",
		Type:      "application",
		Framework: &framework,
	}
	url := buildRepoUrl(spec)
	fmt.Printf("Expected: %v. Actual: %s", expectedUrl, url)

	if url != expectedUrl {
		t.Fatalf("Expected: %v. Actual: %s", expectedUrl, url)
	}
}

func TestBuildRepoUrl_FrameworkIsNil(t *testing.T) {
	expectedUrl := "https://github.com/epmd-edp/javascript-npm-react.git"
	spec := v1alpha1.CodebaseSpec{
		Lang:      "javascript",
		BuildTool: "npm",
		Type:      "library",
	}
	url := buildRepoUrl(spec)
	fmt.Printf("Expected: %v. Actual: %v", expectedUrl, url)

	if url != expectedUrl {
		t.Fatalf("Expected: %v. Actual: %v", expectedUrl, url)
	}
}

func TestBuildRepoUrl_PostgresDatabase(t *testing.T) {
	expectedUrl := "https://github.com/epmd-edp/javascript-npm-react-postgresql.git"
	spec := v1alpha1.CodebaseSpec{
		Lang:      "javascript",
		BuildTool: "npm",
		Type:      "library",
		Database: &v1alpha1.Database{
			Kind: "PostgreSQL",
		},
	}
	url := buildRepoUrl(spec)
	fmt.Printf("Expected: %v. Actual: %v", expectedUrl, url)

	if url != expectedUrl {
		t.Fatalf("Expected: %v. Actual: %v", expectedUrl, url)
	}
}
