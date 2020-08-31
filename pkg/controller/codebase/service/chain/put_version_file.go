package chain

import (
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/helper"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	git "github.com/epmd-edp/codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type PutVersionFile struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
	cr        repository.CodebaseRepository
	git       git.Git
}

const (
	versionFileName = "VERSION"
	initVersion     = "0.0.1"
	goLang          = "go"
)

func (h PutVersionFile) ServeRequest(c *v1alpha1.Codebase) error {
	if strings.ToLower(c.Spec.Lang) != goLang ||
		(strings.ToLower(c.Spec.Lang) == goLang && c.Spec.Versioning.Type == "edp") {
		return nextServeOrNil(h.next, c)
	}

	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("start putting VERSION file...")

	name, err := helper.GetEDPName(h.clientSet.Client, c.Namespace)
	if err != nil {
		setFailedFields(c, v1alpha1.PutVersionFile, err.Error())
		return err
	}

	exists, err := h.versionFileExists(c.Name, *name)
	if err != nil {
		setFailedFields(c, v1alpha1.PutVersionFile, err.Error())
		return err
	}

	if exists {
		log.V(2).Info("skip pushing VERSION file to Git provider. file already exists",
			"name", c.Name)
		return nextServeOrNil(h.next, c)
	}

	if err := h.tryToPutVersionFile(c, util.GetWorkDir(c.Name, c.Namespace)); err != nil {
		setFailedFields(c, v1alpha1.PutVersionFile, err.Error())
		return err
	}

	if err := h.cr.UpdateProjectStatusValue(util.ProjectVersionGoFilePushedStatus, c.Name, *name); err != nil {
		err := errors.Wrapf(err, "couldn't set project_status %v value for %v codebase",
			util.ProjectVersionGoFilePushedStatus, c.Name)
		setFailedFields(c, v1alpha1.PutVersionFile, err.Error())
		return err
	}

	rLog.Info("end putting VERSION file...")
	return nextServeOrNil(h.next, c)
}

func (h PutVersionFile) versionFileExists(codebaseName, edpName string) (bool, error) {
	ps, err := h.cr.SelectProjectStatusValue(codebaseName, edpName)
	if err != nil {
		return false, errors.Wrapf(err, "couldn't get project_status value for %v codebase", codebaseName)
	}

	if *ps == util.ProjectVersionGoFilePushedStatus {
		return true, nil
	}

	return false, nil
}

func (h PutVersionFile) tryToPutVersionFile(c *v1alpha1.Codebase, projectPath string) error {
	path := fmt.Sprintf("%v/%v", projectPath, versionFileName)
	if err := createFile(path); err != nil {
		return errors.Wrapf(err, "couldn't create file %v", path)
	}

	if err := writeFile(path); err != nil {
		return errors.Wrapf(err, "couldn't write to file %v", path)
	}

	gs, err := util.GetGitServer(h.clientSet.Client, c.Name, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	secret, err := getSecret(h.clientSet.Client, gs.NameSshKeySecret, c.Namespace)
	if err != nil {
		return errors.Wrapf(err, "an error has occurred while getting %v secret", gs.NameSshKeySecret)
	}

	k := string(secret.Data[util.PrivateSShKeyName])
	u := gs.GitUser
	if err := h.pushChanges(projectPath, k, u); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing %v for %v codebase", versionFileName, c.Name)
	}

	return nil
}

func getSecret(c client.Client, name, namespace string) (*v1.Secret, error) {
	log.Info("Start fetching Secret resource from k8s", "secret name", name, "namespace", namespace)
	s := &v1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, s); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "secret %v doesn't exist in k8s.", name)
		}
		return nil, err
	}
	log.Info("Secret has been fetched", "secret name", name, "namespace", namespace)
	return s, nil
}

func (h PutVersionFile) pushChanges(projectPath, privateKey, user string) error {
	if err := h.git.CommitChanges(projectPath, fmt.Sprintf("Add %v file", versionFileName)); err != nil {
		return err
	}

	if err := h.git.PushChanges(privateKey, user, projectPath); err != nil {
		return errors.Wrapf(err, "an error has occurred while pushing changes for %v project", projectPath)
	}

	return nil
}

func createFile(filePath string) error {
	if _, err := os.Stat(filePath); os.IsExist(err) {
		log.Info("File already exists. skip creating.", "name", filePath)
		return nil
	}

	var file, err = os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	log.Info("File has been created.", "name", filePath)
	return nil
}

func writeFile(filePath string) error {
	var file, err = os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = file.WriteString(initVersion); err != nil {
		return err
	}

	if err = file.Sync(); err != nil {
		return err
	}

	log.Info("File has been updated.", "name", filePath)
	return nil
}
