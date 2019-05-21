package impl

import (
	"codebase-operator/pkg/apis/edp/v1alpha1"
	"testing"
)

var (
	baseUrl = "https://github.com/epmd-edp"
)

func TestBuildRepoUrl_DatabaseIsNil(t *testing.T) {
	expectedUrl := "https://github.com/epmd-edp/java-maven-springboot.git"

	spec := v1alpha1.CodebaseSpec{
		Lang:      "Java",
		BuildTool: "Maven",
		Framework: "SpringBoot",
	}
	url := buildRepoUrl(baseUrl, spec)

	if url != expectedUrl {
		t.Fatalf("Expected: %v. Actual: %v", expectedUrl, url)
	}
}

func TestBuildRepoUrl_PostgresDatabase(t *testing.T) {
	expectedUrl := "https://github.com/epmd-edp/java-maven-springboot-postgresql.git"

	db := v1alpha1.Database{
		Kind: "PostgreSQL",
	}

	spec := v1alpha1.CodebaseSpec{
		Lang:      "Java",
		BuildTool: "Maven",
		Framework: "SpringBoot",
		Database:  &db,
	}
	url := buildRepoUrl(baseUrl, spec)

	if url != expectedUrl {
		t.Fatalf("Expected: %v. Actual: %v", expectedUrl, url)
	}
}
