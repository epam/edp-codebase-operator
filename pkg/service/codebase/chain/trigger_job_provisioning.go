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
	"strings"
	"time"
)

type TriggerJobProvisioning struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
}

func (h TriggerJobProvisioning) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start triggering job provisioning...")

	if err := h.setIntermediateSuccessFields(c, edpv1alpha1.JenkinsConfiguration); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	if err := h.tryToTriggerJobProvisioning(c); err != nil {
		setFailedFields(*c, edpv1alpha1.JenkinsConfiguration, err.Error())
		return err
	}
	rLog.Info("Job provisioning has been triggered")

	if err := h.updateFinishStatus(c); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}

	wd := fmt.Sprintf("/home/codebase-operator/edp/%v/%v", c.Namespace, c.Name)
	if err := util.RemoveDirectory(wd); err != nil {
		return err
	}
	return nextServeOrNil(h.next, c)
}

func (h TriggerJobProvisioning) initJenkinsClient(namespace string) (*jenkins.JenkinsClient, error) {
	j, err := h.getJenkinsData(namespace)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get Jenkins info")
	}
	log.Info("jenkins data", "url", j.JenkinsUrl, "username", j.JenkinsUsername)

	js, err := jenkins.Init(j.JenkinsUrl, j.JenkinsUsername, j.JenkinsToken)
	if err != nil {
		return nil, err
	}
	return js, nil
}

func (h TriggerJobProvisioning) tryToTriggerJobProvisioning(c *v1alpha1.Codebase) error {
	log.Info("start triggering job provisioning", "name", c.Spec.JobProvisioning)
	js, err := h.initJenkinsClient(c.Namespace)
	if err != nil {
		return err
	}

	success, err := js.IsBuildSuccessful(c.Spec.JobProvisioning, c.Status.JenkinsJobProvisionBuildNumber)
	if err != nil {
		return errors.Wrapf(err, "couldn't check build status for job %v", c.Spec.JobProvisioning)
	}

	if success {
		log.Info("las build was successful. triggering of job provision is skipped")
		return nil
	}

	gs, err := util.GetGitServer(h.clientSet.Client, c.Name, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	path := getRepositoryPath(c.Name, string(c.Spec.Strategy), c.Spec.GitUrlPath)
	sshLink := generateSshLink(path, gs)
	jpm := map[string]string{
		"PARAM":                 "true",
		"NAME":                  c.Name,
		"BUILD_TOOL":            strings.ToLower(c.Spec.BuildTool),
		"GIT_SERVER_CR_NAME":    gs.Name,
		"GIT_SERVER_CR_VERSION": "v2",
		"GIT_CREDENTIALS_ID":    gs.NameSshKeySecret,
		"REPOSITORY_PATH":       sshLink,
	}

	bn, err := js.TriggerJobProvisioning(c.Spec.JobProvisioning, jpm)
	if err != nil {
		return errors.Wrap(err, "an error has been occurred while triggering job provisioning")
	}
	c.Status.JenkinsJobProvisionBuildNumber = *bn
	return nil
}

func generateSshLink(repoPath string, gs *model.GitServer) string {
	l := fmt.Sprintf("ssh://%v@%v:%v%v", gs.GitUser, gs.GitHost, gs.SshPort, repoPath)
	log.Info("generated SSH link", "link", l)
	return l
}

func getRepositoryPath(codebaseName, strategy string, gitUrlPath *string) string {
	if strategy == util.ImportStrategy {
		return *gitUrlPath
	}
	return "/" + codebaseName
}

func (h TriggerJobProvisioning) getJenkinsData(namespace string) (*model.Jenkins, error) {
	jen, err := jenkins.GetJenkins(h.clientSet.Client, namespace)
	if err != nil {
		return nil, err
	}
	jt, ju, err := jenkins.GetJenkinsCreds(*jen, h.clientSet, namespace)
	if err != nil {
		return nil, err
	}
	return &model.Jenkins{
		JenkinsUrl:      jenkins.GetJenkinsUrl(*jen, namespace),
		JenkinsUsername: ju,
		JenkinsToken:    jt,
	}, nil
}

func (h TriggerJobProvisioning) setIntermediateSuccessFields(c *edpv1alpha1.Codebase, action edpv1alpha1.ActionType) error {
	c.Status = edpv1alpha1.CodebaseStatus{
		Status:                         util.StatusInProgress,
		Available:                      false,
		LastTimeUpdated:                time.Now(),
		Action:                         action,
		Result:                         edpv1alpha1.Success,
		Username:                       "system",
		Value:                          "inactive",
		JenkinsJobProvisionBuildNumber: c.Status.JenkinsJobProvisionBuildNumber,
	}
	return h.updateStatus(c)
}

func (h TriggerJobProvisioning) updateFinishStatus(c *edpv1alpha1.Codebase) error {
	c.Status = edpv1alpha1.CodebaseStatus{
		Status:                         util.StatusFinished,
		Available:                      true,
		LastTimeUpdated:                time.Now(),
		Username:                       "system",
		Action:                         edpv1alpha1.SetupDeploymentTemplates,
		Result:                         edpv1alpha1.Success,
		Value:                          "active",
		JenkinsJobProvisionBuildNumber: c.Status.JenkinsJobProvisionBuildNumber,
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
