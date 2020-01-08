package chain

import (
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/jenkins"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/service/codebase/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"
)

type TriggerJobProvisioning struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
}

const ImportStrategy = "import"

func (h TriggerJobProvisioning) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start triggering job provisioning...")

	gs, err := util.GetGitServer(h.clientSet.Client, c.Name, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	path := getRepositoryPath(c.Name, string(c.Spec.Strategy), c.Spec.GitUrlPath)
	sshLink := generateSshLink(path, gs)
	j, err := h.getJenkinsData(c.Spec.JobProvisioning, c.Namespace)
	if err != nil {
		return errors.Wrap(err, "couldn't get Jenkins info")
	}

	err = h.triggerJobProvisioning(*j,
		map[string]string{
			"PARAM":                 "true",
			"NAME":                  c.Name,
			"BUILD_TOOL":            strings.ToLower(c.Spec.BuildTool),
			"GIT_SERVER_CR_NAME":    gs.Name,
			"GIT_SERVER_CR_VERSION": "v2",
			"GIT_CREDENTIALS_ID":    gs.NameSshKeySecret,
			"REPOSITORY_PATH":       sshLink,
		})
	if err != nil {
		setFailedFields(*c, edpv1alpha1.JenkinsConfiguration, err.Error())
		return errors.Wrap(err, "an error has been occurred while triggering job provisioning")
	}

	if err := h.setIntermediateSuccessFields(c, edpv1alpha1.JenkinsConfiguration); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}
	rLog.Info("Job provisioning has been triggered")

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", c.Namespace, c.Name)
	if err := os.RemoveAll(wd); err != nil {
		return errors.Wrapf(err, "couldn't remove work directory %v", wd)
	}
	rLog.Info("work directory has been cleaned", "directory", wd)

	if err := h.updateFinishStatus(c); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}
	return nextServeOrNil(h.next, c)
}

func generateSshLink(repoPath string, gs *model.GitServer) string {
	l := fmt.Sprintf("ssh://%v@%v:%v%v", gs.GitUser, gs.GitHost, gs.SshPort, repoPath)
	log.Info("generated SSH link", "link", l)
	return l
}

func getRepositoryPath(codebaseName, strategy string, gitUrlPath *string) string {
	if strategy == ImportStrategy {
		return *gitUrlPath
	}
	return "/" + codebaseName
}

func (h TriggerJobProvisioning) getJenkinsData(jobProvision, namespace string) (*model.Jenkins, error) {
	jen, err := jenkins.GetJenkins(h.clientSet.Client, namespace)
	if err != nil {
		return nil, err
	}
	jt, ju, err := jenkins.GetJenkinsCreds(*jen, h.clientSet, namespace)
	if err != nil {
		return nil, err
	}
	jurl := jenkins.GetJenkinsUrl(*jen, namespace)
	return &model.Jenkins{
		JenkinsUrl:      jurl,
		JenkinsUsername: ju,
		JenkinsToken:    jt,
		JobName:         jobProvision,
	}, nil
}

func (h TriggerJobProvisioning) triggerJobProvisioning(data model.Jenkins, parameters map[string]string) error {
	log.Info("start triggering job provision", "codebase name", parameters["NAME"])
	js, err := jenkins.Init(data.JenkinsUrl, data.JenkinsUsername, data.JenkinsToken)
	if err != nil {
		return err
	}

	if err := js.TriggerJobProvisioning(data.JobName, parameters, 10*time.Second, 12); err != nil {
		return err
	}
	log.Info("end triggering job provision", "codebase name", parameters["NAME"])
	return nil
}

func (h TriggerJobProvisioning) setIntermediateSuccessFields(c *edpv1alpha1.Codebase, action edpv1alpha1.ActionType) error {
	c.Status = edpv1alpha1.CodebaseStatus{
		Status:          util.StatusInProgress,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Action:          action,
		Result:          edpv1alpha1.Success,
		Username:        "system",
		Value:           "inactive",
	}
	return h.updateStatus(c)
}

func (h TriggerJobProvisioning) updateFinishStatus(c *edpv1alpha1.Codebase) error {
	c.Status = edpv1alpha1.CodebaseStatus{
		Status:          util.StatusFinished,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          edpv1alpha1.SetupDeploymentTemplates,
		Result:          edpv1alpha1.Success,
		Value:           "active",
	}
	return h.updateStatus(c)
}

func (h TriggerJobProvisioning) updateStatus(c *edpv1alpha1.Codebase) error {
	if err := h.clientSet.Client.Status().Update(context.TODO(), c); err != nil {
		if err := h.clientSet.Client.Update(context.TODO(), c); err != nil {
			return err
		}
	}
	return nil
}
