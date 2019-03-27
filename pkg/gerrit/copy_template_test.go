package gerrit

import (
	"business-app-handler-controller/models"
	"business-app-handler-controller/pkg/apis/edp/v1alpha1"
	"fmt"
	"testing"
)

func TestExampleAddEdpConfigs_CopyTemplate_Test(t *testing.T) {
	dtrUrl := "https://docker.com"
	lang := "Java"
	framework := "springboot"
	buildTool := "maven"
	templatesDir := "/oc-templates"
	cloneSshUrl := "ssh://gerrit.svc"
	appSettings := models.AppSettings{
		Name: "test",
	}

	route := v1alpha1.Route{
		Site: "site",
		Path: "/my_path",
	}

	database := v1alpha1.Database{
		Kind:     "PostgreSQL",
		Version:  "9.6",
		Capacity: "1Gi",
		Storage:  "efs",
	}

	templatePath := "/templates/applications/java/springboot.tmpl"
	templateName := "springboot.tmpl"

	config := gerritConfigGoTemplating{
		DockerRegistryUrl: dtrUrl,
		Lang:              lang,
		Framework:         framework,
		BuildTool:         buildTool,
		TemplatesDir:      templatesDir,
		CloneSshUrl:       cloneSshUrl,
		AppSettings:       appSettings,
		Database:          &database,
		Route:             &route,
	}

	err := copyTemplate(templatePath, templateName, config, appSettings)
	if err != nil {
		fmt.Println(err)
	}
	if err != nil {
		t.Error("Actual: error. Expected: true")
	}
}
