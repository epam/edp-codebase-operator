package gerrit

import (
	"bytes"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	ClientSet "github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	imageV1 "github.com/openshift/api/image/v1"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

func ConfigInit(codebaseSettings model.CodebaseSettings,
	spec v1alpha1.CodebaseSpec) (*model.GerritConfigGoTemplating, error) {

	templatesDir := createTemplateDirectory(codebaseSettings.WorkDir, codebaseSettings.DeploymentScript)
	cloneSshUrl := fmt.Sprintf("ssh://project-creator@gerrit.%v:%v/%v", codebaseSettings.CicdNamespace,
		codebaseSettings.GerritSettings.SshPort, codebaseSettings.Name)

	config := model.GerritConfigGoTemplating{
		Lang:             spec.Lang,
		Framework:        spec.Framework,
		BuildTool:        spec.BuildTool,
		TemplatesDir:     templatesDir,
		CloneSshUrl:      cloneSshUrl,
		CodebaseSettings: codebaseSettings,
	}
	if spec.Repository != nil {
		config.RepositoryUrl = &spec.Repository.Url
	}
	if spec.Database != nil {
		config.Database = spec.Database
	}
	if spec.Route != nil {
		config.Route = spec.Route
	}

	log.Print("Gerrit config has been initialized")

	return &config, nil
}

func createTemplateDirectory(workDir string, deploymentScriptType string) string {
	if deploymentScriptType == util.OpenshiftTemplate {
		return fmt.Sprintf("%v/%v", workDir, util.OcTemplatesFolder)
	}
	return fmt.Sprintf("%v/%v", workDir, util.HelmChartTemplatesFolder)
}

func PushConfigs(config model.GerritConfigGoTemplating, codebaseSettings model.CodebaseSettings, clientSet ClientSet.ClientSet) error {
	appTemplatesDir := fmt.Sprintf("%v/%v/deploy-templates", config.TemplatesDir, codebaseSettings.Name)
	appConfigFilesDir := fmt.Sprintf("%v/%v/config-files", config.TemplatesDir, codebaseSettings.Name)

	err := createDirectory(config.TemplatesDir)
	if err != nil {
		return err
	}

	err = cloneProjectRepoFromGerrit(config, codebaseSettings)
	if err != nil {
		return err
	}

	err = createDirectory(appConfigFilesDir)
	if err != nil {
		return err
	}

	destinationPath := fmt.Sprintf("%v/%v/config-files", config.TemplatesDir, codebaseSettings.Name)
	sourcePath := "/usr/local/bin/templates/gerrit"
	fileName := "Readme.md"
	err = copyFile(destinationPath, sourcePath, fileName)
	if err != nil {
		return err
	}

	err = createDirectory(appTemplatesDir)
	if err != nil {
		return err
	}

	if codebaseSettings.Type == "application" {
		config.CodebaseSettings = codebaseSettings

		err = util.CopyTemplate(config)
		if err != nil {
			return err
		}
	}

	err = copyPipelines(codebaseSettings, config)
	if err != nil {
		return nil
	}

	if strings.ToLower(config.Lang) == "javascript" {
		err = copySonarConfigs(config, codebaseSettings)
		if err != nil {
			return err
		}
	}

	err = commitConfigs(config, codebaseSettings.Name)
	if err != nil {
		return err
	}

	err = pushConfigsToGerrit(config, codebaseSettings.Name, codebaseSettings.GerritKeyPath)
	if err != nil {
		return err
	}

	return tryCreateImageStream(clientSet, codebaseSettings)
}

func getTemplateName(deploymentScriptType, framework string) string {
	if deploymentScriptType == util.OpenshiftTemplate {
		return framework
	}
	return "values"
}

func tryCreateImageStream(cs ClientSet.ClientSet, c model.CodebaseSettings) error {
	if !isSupportedType(c) {
		log.Println("couldn't create image stream as type of codebase is not acceptable")
		return nil
	}

	appImageStream, err := GetAppImageStream(c.Lang)
	if err != nil {
		return err
	}
	return CreateS2IImageStream(cs, c.Name, c.CicdNamespace, appImageStream)
}

func isSupportedType(c model.CodebaseSettings) bool {
	return c.Type == "application" && c.Lang != "other"
}

func cloneProjectRepoFromGerrit(config model.GerritConfigGoTemplating, codebaseSettings model.CodebaseSettings) error {
	log.Printf("Cloning repo from gerrit using: %v", config.CloneSshUrl)
	var session *ssh.Session
	var connection *ssh.Client
	var out bytes.Buffer
	var stderr bytes.Buffer

	client, err := SshInit(codebaseSettings.GerritKeyPath, codebaseSettings.GerritHost, codebaseSettings.GerritSettings.SshPort)
	if err != nil {
		return err
	}

	if session, connection, err = client.newSession(); err != nil {
		return err
	}
	defer func() {
		if deferErr := session.Close(); deferErr != nil {
			err = deferErr
		}
		if deferErr := connection.Close(); deferErr != nil {
			err = deferErr
		}
	}()

	cmd := exec.Command("git", "clone", config.CloneSshUrl, fmt.Sprintf("%v/%v",
		config.TemplatesDir, codebaseSettings.Name))
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	log.Printf("Cloning repo %v to %v: Output: %v", config.CloneSshUrl, config.TemplatesDir, out.String())

	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return err
	}
	log.Print("Cloning repo has been finished")

	destinationPath := fmt.Sprintf("%v/%v/.git/hooks", config.TemplatesDir, codebaseSettings.Name)
	sourcePath := "/usr/local/bin/configs"
	fileName := "commit-msg"
	err = copyFile(destinationPath, sourcePath, fileName)
	if err != nil {
		return err
	}

	return nil
}

func copyFile(destinationPath string, sourcePath string, fileName string) error {
	fullDestinationPath := fmt.Sprintf("%v/%v", destinationPath, fileName)
	fullSourcePath := fmt.Sprintf("%v/%v", sourcePath, fileName)
	log.Printf("Copying %v to config maps", fullSourcePath)
	copyFrom, err := os.Open(fullSourcePath)
	if err != nil {
		log.Fatal(err)
	}
	defer copyFrom.Close()

	copyTo, err := os.OpenFile(fullDestinationPath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer copyTo.Close()

	_, err = io.Copy(copyTo, copyFrom)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func createDirectory(path string) error {
	log.Printf("Creating directory for oc templates: %v", path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}
	log.Printf("Directory %v has been created", path)
	return nil
}

func copyPipelines(codebaseSettings model.CodebaseSettings, config model.GerritConfigGoTemplating) error {
	pipelinesPath := "/usr/local/bin/pipelines"
	files, err := ioutil.ReadDir(pipelinesPath)
	if err != nil {
		return err
	}

	pipelinesDest := fmt.Sprintf("%v/%v", config.TemplatesDir, codebaseSettings.Name)
	log.Printf("Start copying pipelines to %v", pipelinesDest)

	for _, f := range files {
		if codebaseSettings.Type == "autotests" && f.Name() == "build.groovy" {
			continue
		}

		input, err := ioutil.ReadFile(pipelinesPath + "/" + f.Name())
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(pipelinesDest+"/"+f.Name(), input, 0755)
		if err != nil {
			return err
		}
	}

	log.Printf("Jenkins pipelines for codebase %v has been copied", codebaseSettings.Name)
	return nil
}

func copySonarConfigs(config model.GerritConfigGoTemplating, codebaseSettings model.CodebaseSettings) error {
	sonarConfigPath := fmt.Sprintf("%v/%v/sonar-project.properties", config.TemplatesDir, codebaseSettings.Name)

	if _, err := os.Stat(sonarConfigPath); err == nil {
		return nil

	} else if os.IsNotExist(err) {
		f, err := os.Create(sonarConfigPath)
		if err != nil {
			return err
		}
		tmpl, err := template.New("sonar-project.properties.tmpl").
			ParseFiles("/usr/local/bin/templates/sonar/sonar-project.properties.tmpl")
		if err != nil {
			return err
		}
		err = tmpl.Execute(f, config)
		if err != nil {
			log.Printf("Unable to render sonar configs fo JS app: %v", err)
			return err
		}
		log.Printf("Sonar configs for codebase %v has been copied", codebaseSettings.Name)
		defer f.Close()
	}

	return nil
}

func commitConfigs(config model.GerritConfigGoTemplating, appName string) error {
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
	log.Printf("Commit changes has been completed for application %v", appName)
	return nil
}

func pushConfigsToGerrit(gerritConfig model.GerritConfigGoTemplating, appName string, keyPath string) error {
	auth, err := Auth(keyPath)
	if err != nil {
		return err
	}

	r, err := git.PlainOpen(gerritConfig.TemplatesDir + "/" + appName)
	if err != nil {
		return err
	}

	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			"refs/heads/*:refs/heads/*",
			"refs/tags/*:refs/tags/*",
		},
		Auth: auth,
	})
	if err != nil {
		return err
	}
	log.Printf("Configs has been pushed successfully for application %v", appName)

	return nil
}

func CreateS2IImageStream(clientSet ClientSet.ClientSet, codebaseName string, namespace string, is *imageV1.ImageStream) error {
	log.Printf("Trying to create s2i image stream for %v codebase in %v namespace", codebaseName, namespace)

	_, err := clientSet.ImageClient.ImageStreams(namespace).Get(is.Name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		_, err := clientSet.ImageClient.ImageStreams(namespace).Create(is)
		if err != nil {
			return err
		}
		log.Printf("Image stream in Openshift has been created for application %v", codebaseName)
	} else {
		log.Printf("Image stream in Openshift for application %v already exist. Creation skipped", codebaseName)
	}

	return nil
}

func newS2IReact(lang string) *imageV1.ImageStream {
	return &imageV1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name: "s2i-" + strings.ToLower(lang),
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
					Name: "epamedp/s2i-nginx:latest",
				},
				ReferencePolicy: imageV1.TagReferencePolicy{
					Type: "Source",
				},
			}},
		},
	}
}

func newS2IJava(lang string) *imageV1.ImageStream {
	return &imageV1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name: "s2i-" + strings.ToLower(lang),
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
					Name: "epamedp/s2i-java:latest",
				},
				ReferencePolicy: imageV1.TagReferencePolicy{
					Type: "Source",
				},
			}},
		},
	}
}

func newS2IDotNet(lang string) *imageV1.ImageStream {
	return &imageV1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "s2i-" + strings.ToLower(lang),
			Annotations: map[string]string{"openshift.io/display-name": ".NET Core Builder Images"},
		},
		Spec: imageV1.ImageStreamSpec{
			LookupPolicy: imageV1.ImageLookupPolicy{
				Local: false,
			},
			Tags: []imageV1.TagReference{{
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
					Kind: "DockerImage",
					Name: "epamedp/dotnet-20-centos7:latest",
				},
				ImportPolicy: imageV1.TagImportPolicy{},
				ReferencePolicy: imageV1.TagReferencePolicy{
					Type: "Source",
				},
			}},
		},
	}
}

func GetAppImageStream(lang string) (*imageV1.ImageStream, error) {
	log.Printf("Trying to get image stream %v", lang)

	switch strings.ToLower(lang) {
	case model.JavaScript:
		return newS2IReact(lang), nil
	case model.Java:
		return newS2IJava(lang), nil
	case model.DotNet:
		return newS2IDotNet(lang), nil
	}
	return nil, nil
}
