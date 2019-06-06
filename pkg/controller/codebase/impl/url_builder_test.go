package impl

import (
	"codebase-operator/pkg/apis/edp/v1alpha1"
	"fmt"
	"testing"
)

var (
	baseUrl = "https://github.com/epmd-edp"
)

func TestBuildRepoUrl_DatabaseIsNil(t *testing.T) {
	expectedUrl := "https://github.com/epmd-edp/java-maven-springboot.git"
	framework := "SpringBoot"

	spec := v1alpha1.CodebaseSpec{
		Lang:      "Java",
		BuildTool: "Maven",
		Type:      "application",
		Framework: &framework,
	}
	url := buildRepoUrl(baseUrl, spec)
	fmt.Printf("Expected: %v. Actual: %s", expectedUrl, url)

	if url != expectedUrl {
		t.Fatalf("Expected: %v. Actual: %s", expectedUrl, url)
	}
}

func TestBuildRepoUrl_FrameworkIsNil(t *testing.T) {
	expectedUrl := "https://github.com/epmd-edp/sample-library.git"
	var framework *string = nil

	spec := v1alpha1.CodebaseSpec{
		Lang:      "Java",
		BuildTool: "Maven",
		Type:      "library",
		Framework: framework,
	}
	url := buildRepoUrl(baseUrl, spec)
	fmt.Printf("Expected: %v. Actual: %v", expectedUrl, url)

	if url != expectedUrl {
		t.Fatalf("Expected: %v. Actual: %v", expectedUrl, url)
	}
}

func TestBuildRepoUrl_PostgresDatabase(t *testing.T) {
	expectedUrl := "https://github.com/epmd-edp/java-maven-springboot-postgresql.git"
	framework := "SpringBoot"

	db := v1alpha1.Database{
		Kind: "PostgreSQL",
	}

	spec := v1alpha1.CodebaseSpec{
		Lang:      "Java",
		Type:      "application",
		BuildTool: "Maven",
		Framework: &framework,
		Database:  &db,
	}
	url := buildRepoUrl(baseUrl, spec)
	fmt.Printf("Expected: %v. Actual: %v", expectedUrl, url)

	if url != expectedUrl {
		t.Fatalf("Expected: %v. Actual: %v", expectedUrl, url)
	}
}
