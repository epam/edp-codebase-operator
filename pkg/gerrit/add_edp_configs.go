package gerrit

import (
	"business-app-handler-controller/models"
	"business-app-handler-controller/pkg/apis/edp/v1alpha1"
	ClientSet "business-app-handler-controller/pkg/openshift"
	"errors"
	"fmt"
	imageV1 "github.com/openshift/api/image/v1"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"html/template"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type gerritConfigGoTemplating struct {
	Lang              string             `json:"lang"`
	Framework         string             `json:"framework"`
	BuildTool         string             `json:"build_tool"`
	RepositoryUrl     string             `json:"repository_url"`
	Route             v1alpha1.Route     `json:"route"`
	Database          v1alpha1.Database  `json:"database"`
	AppSettings       models.AppSettings `json:"app_settings"`
	DockerRegistryUrl string             `json:"docker_registry_url"`
	TemplatesDir      string             `json:"templates_dir"`
	CloneSshUrl       string             `json:"clone_ssh_url"`
	MessageHookCmd    string             `json:"message_hook_cmd"`
}

func ConfigInit(clientSet ClientSet.ClientSet, appSettings models.AppSettings,
	spec v1alpha1.BusinessApplicationSpec) (*gerritConfigGoTemplating, error) {
	dtrUrl, err := getOpenshiftDockerRegistryUrl(clientSet)
	if err != nil {
		return nil, err
	}

	templatesDir := fmt.Sprintf("%v/oc-templates", appSettings.WorkDir)
	cloneSshUrl := fmt.Sprintf("ssh://project-creator@gerrit.%v:%v/%v", appSettings.CicdNamespace,
		appSettings.GerritSettings.SshPort, appSettings.Name)
	messageHookCmd := fmt.Sprintf("project-creator@gerrit.%v:hooks/commit-msg %v/.git/hooks/",
		appSettings.CicdNamespace, appSettings.Name)

	config := gerritConfigGoTemplating{
		DockerRegistryUrl: *dtrUrl,
		Lang:              spec.Lang,
		Framework:         spec.Framework,
		BuildTool:         spec.BuildTool,
		RepositoryUrl:     spec.Repository.Url,
		Route:             *spec.Route,
		Database:          *spec.Database,
		TemplatesDir:      templatesDir,
		CloneSshUrl:       cloneSshUrl,
		MessageHookCmd:    messageHookCmd,
		AppSettings:       appSettings,
	}
	return &config, nil
}

func getOpenshiftDockerRegistryUrl(clientSet ClientSet.ClientSet) (*string, error) {
	dtrRegistry, err := clientSet.RouteClient.Routes("default").Get("docker-registry", metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get user settings configmap: %v", err)
		log.Println(errorMsg)
		return nil, errors.New(errorMsg)
	}
	return &dtrRegistry.Spec.Host, nil
}

func PushConfigs(config gerritConfigGoTemplating, appSettings models.AppSettings, clientSet ClientSet.ClientSet) error {
	err := cloneProjectRepoFromGerrit(config, appSettings)
	if err != nil {
		return err
	}

	appTemplatesDir := fmt.Sprintf("%v/%v/deploy-templates", config.TemplatesDir, appSettings.Name)
	err = createDirectory(appTemplatesDir)
	if err != nil {
		return err
	}

	templateBasePath := fmt.Sprintf("%v/%v/", config.AppSettings.Name, config.Lang)
	templateName := fmt.Sprintf("%v.tmpl", config.Framework)
	templatePath := fmt.Sprintf("%v/%v", templateBasePath, templateName)

	err = copyTemplate(templatePath, templateName, config, appSettings)
	if err != nil {
		return err
	}

	err = copyPipelines(appSettings, config)
	if err != nil {
		return nil
	}

	if strings.ToLower(config.Lang) == "javascript" {
		err = copySonarConfigs(config, appSettings)
		if err != nil {
			return err
		}
	}

	err = commitConfigs(config, appSettings.Name)
	if err != nil {
		return err
	}

	err = pushConfigsToGerrit(config, appSettings.Name)
	if err != nil {
		return err
	}

	appImageStream, err := getAppImageStream(config)
	if err != nil {
		return err
	}

	err = createS2IImageStream(clientSet, appSettings, appImageStream)
	if err != nil {
		return err
	}

	return nil
}

func cloneProjectRepoFromGerrit(config gerritConfigGoTemplating, appSettings models.AppSettings) error {
	cmd := exec.Command("git", "clone", config.CloneSshUrl)
	cmd.Dir = config.TemplatesDir
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	cmd = exec.Command("scp", "-p", "-P", string(appSettings.GerritSettings.SshPort), config.MessageHookCmd)
	return nil
}

func createDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func copyTemplate(templatePath string, templateName string, config gerritConfigGoTemplating, appSettings models.AppSettings) error {
	templatesDest := fmt.Sprintf("%v/%v/deploy-templates/%v.yaml", config.TemplatesDir, appSettings.Name,
		appSettings.Name)
	f, err := os.Create(templatesDest)
	if err != nil {
		return err
	}
	tmpl, err := template.New(templateName).ParseFiles(templatePath)
	if err != nil {
		return err
	}
	err = tmpl.Execute(f, config)
	if err != nil {
		log.Printf("Unable to render application deploy template: %v", err)
		return err
	}
	return nil
}

func copyPipelines(appSettings models.AppSettings, config gerritConfigGoTemplating) error {
	files, err := ioutil.ReadDir(appSettings.WorkDir + "/pipelines")
	pipelinesDest := fmt.Sprintf("%v/%v", config.TemplatesDir, appSettings.Name)
	if err != nil {
		return err
	}

	for _, f := range files {
		input, err := ioutil.ReadFile(f.Name())
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(pipelinesDest+"/"+f.Name(), input, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func copySonarConfigs(config gerritConfigGoTemplating, appSettings models.AppSettings) error {
	sonarConfigPath := fmt.Sprintf("%v/%v/sonar-project.properties", config.TemplatesDir, appSettings.Name)

	if _, err := os.Stat(sonarConfigPath); err == nil {
		return nil

	} else if os.IsNotExist(err) {
		f, err := os.Create(sonarConfigPath)
		if err != nil {
			return err
		}
		tmpl, err := template.New("sonar-project.properties.tmpl").
			ParseFiles("templates/sonar/sonar-project.properties.tmpl")
		if err != nil {
			return err
		}
		err = tmpl.Execute(f, config)
		if err != nil {
			log.Printf("Unable to render sonar configs fo JS app: %v", err)
			return err
		}
		defer f.Close()
	}

	return nil
}

func commitConfigs(config gerritConfigGoTemplating, appName string) error {
	commitMessage := fmt.Sprintf("Add template for %v", appName)
	r, err := git.PlainOpen(config.TemplatesDir + "/" + appName)
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	_, err = w.Add(".")
	if err != nil {
		return err
	}

	_, err = w.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "admin",
			Email: "admin@epam-edp.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func pushConfigsToGerrit(config gerritConfigGoTemplating, appName string) error {
	r, err := git.PlainOpen(config.TemplatesDir + "/" + appName)
	if err != nil {
		return err
	}

	err = r.Push(&git.PushOptions{})
	if err != nil {
		return err
	}
	return nil
}

func createS2IImageStream(clientSet ClientSet.ClientSet, appSettings models.AppSettings, is *imageV1.ImageStream) error {
	_, err := clientSet.ImageClient.ImageStreams(appSettings.CicdNamespace).Create(is)
	if err != nil {
		return err
	}
	return nil
}

func newS2IReact(config gerritConfigGoTemplating) *imageV1.ImageStream {
	return &imageV1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name: "s2i-" + config.Lang,
		},
		Spec: imageV1.ImageStreamSpec{
			LookupPolicy: imageV1.ImageLookupPolicy{
				Local: false,
			},
			Tags: []imageV1.TagReference{{
				Name:        "latest",
				Annotations: nil,
				From: &corev1.ObjectReference{
					Kind: "DockerImage",
					Name: "ibotty/s2i-nginx:latest",
				},
				ReferencePolicy: imageV1.TagReferencePolicy{
					Type: "Source",
				},
			}},
		},
	}
}

func newS2IJava(config gerritConfigGoTemplating) *imageV1.ImageStream {
	return &imageV1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name: "s2i-" + config.Lang,
		},
		Spec: imageV1.ImageStreamSpec{
			LookupPolicy: imageV1.ImageLookupPolicy{
				Local: false,
			},
			Tags: []imageV1.TagReference{{
				Name:        "2.1",
				Annotations: nil,
				From: &corev1.ObjectReference{
					Kind: "DockerImage",
					Name: "fabric8/s2i-java:2.1",
				},
				ReferencePolicy: imageV1.TagReferencePolicy{
					Type: "Source",
				},
			}},
		},
	}
}

func newS2IDotNet(config gerritConfigGoTemplating) *imageV1.ImageStream {
	return &imageV1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "s2i-" + config.Lang,
			Annotations: map[string]string{"openshift.io/display-name": ".NET Core Builder Images"},
		},
		Spec: imageV1.ImageStreamSpec{
			LookupPolicy: imageV1.ImageLookupPolicy{
				Local: false,
			},
			Tags: []imageV1.TagReference{{
				Name: "2.0",
				Annotations: map[string]string{
					"description": "Build and run .NET Core applications on CentOS 7. For more information about" +
						" using this builder image, including OpenShift considerations, see " +
						"https://github.com/redhat-developer/s2i-dotnetcore/tree/master/2.0/build/README.md. " +
						"WARNING: By selecting this tag, your application will automatically update to use the " +
						"latest version of .NET Core available on OpenShift, including major versions updates.",
					"iconClass":                 "icon-dotnet",
					"openshift.io/display-name": ".NET Core (Latest)",
					"sampleContextDir":          "app",
					"sampleRef":                 "dotnetcore-2.0",
					"sampleRepo":                "https://github.com/redhat-developer/s2i-dotnetcore-ex.git",
					"supports":                  "dotnet",
					"tags":                      "builder,.net,dotnet,dotnetcore",
				},
				From: &corev1.ObjectReference{
					Kind: "DockerImage",
					Name: "registry.centos.org/dotnet/dotnet-20-centos7:latest",
				},
				ImportPolicy: imageV1.TagImportPolicy{},
				ReferencePolicy: imageV1.TagReferencePolicy{
					Type: "Source",
				},
			}, {
				Name: "latest",
				Annotations: map[string]string{
					"description": "Build and run .NET Core 2.0 applications on CentOS 7. For more " +
						"information about using this builder image, including OpenShift considerations, " +
						"see https://github.com/redhat-developer/s2i-dotnetcore/tree/master/2.0/build/README.md.",
					"iconClass":                 "icon-dotnet",
					"openshift.io/display-name": ".NET Core 2.0",
					"sampleContextDir":          "app",
					"sampleRef":                 "dotnetcore-2.0",
					"sampleRepo":                "https://github.com/redhat-developer/s2i-dotnetcore-ex.git",
					"supports":                  "dotnet:2.0,dotnet",
					"tags":                      "builder,.net,dotnet,dotnetcore,rh-dotnet20",
					"version":                   "2.0",
				},
				From: &corev1.ObjectReference{
					Kind: "ImageStreamTag",
					Name: "2.0",
				},
				ImportPolicy: imageV1.TagImportPolicy{},
				ReferencePolicy: imageV1.TagReferencePolicy{
					Type: "Source",
				},
			}},
		},
	}
}

func getAppImageStream(config gerritConfigGoTemplating) (*imageV1.ImageStream, error) {
	switch strings.ToLower(config.Lang) {
	case models.JavaScript:
		return newS2IReact(config), nil
	case models.Java:
		return newS2IJava(config), nil
	case models.DotNet:
		return newS2IDotNet(config), nil
	}
	return nil, nil
}
