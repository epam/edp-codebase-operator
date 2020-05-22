package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	jenkinsv1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/util/consts"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"strconv"
	"strings"
	"time"
)

type PutJenkinsFolder struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
}

func (h PutJenkinsFolder) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)

	gs, err := util.GetGitServer(h.clientSet.Client, c.Name, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	log.Info("GIT server has been retrieved", "name", gs.Name)
	path := getRepositoryPath(c.Name, string(c.Spec.Strategy), c.Spec.GitUrlPath)
	sshLink := generateSshLink(path, gs)
	jpm := map[string]string{
		"PARAM":                    "true",
		"NAME":                     c.Name,
		"BUILD_TOOL":               strings.ToLower(c.Spec.BuildTool),
		"GIT_SERVER_CR_NAME":       gs.Name,
		"GIT_SERVER_CR_VERSION":    "v2",
		"GIT_CREDENTIALS_ID":       gs.NameSshKeySecret,
		"REPOSITORY_PATH":          sshLink,
		"JIRA_INTEGRATION_ENABLED": strconv.FormatBool(isJiraIntegrationEnabled(c.Spec.JiraServer)),
	}

	jc, err := json.Marshal(jpm)
	if err != nil {
		return errors.Wrapf(err, "Can't marshal parameters %v into json string", jpm)
	}

	rLog.Info("start creating jenkins folder...")
	if err := h.putJenkinsFolder(c, string(jc)); err != nil {
		setFailedFields(c, v1alpha1.PutJenkinsFolder, err.Error())
		return err
	}
	if err := h.updateFinishStatus(c); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v Codebase status", c.Name)
	}
	rLog.Info("end creating jenkins folder...")
	return nextServeOrNil(h.next, c)
}

func (h PutJenkinsFolder) putJenkinsFolder(c *v1alpha1.Codebase, jc string) error {
	jfn := fmt.Sprintf("%v-%v", c.Name, "codebase")
	jfr, err := h.getJenkinsFolder(jfn, c.Namespace)
	if err != nil {
		return err
	}

	if jfr != nil {
		log.Info("jenkins folder already exists in cluster", "name", jfn)
		return nil
	}

	jf := &jenkinsv1alpha1.JenkinsFolder{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "JenkinsFolder",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jfn,
			Namespace: c.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v2.edp.epam.com/v1alpha1",
					Kind:               util.CodebaseKind,
					Name:               c.Name,
					UID:                c.UID,
					BlockOwnerDeletion: newTrue(),
				},
			},
		},
		Spec: jenkinsv1alpha1.JenkinsFolderSpec{
			JobName: &c.Spec.JobProvisioning,
			Job: jenkinsv1alpha1.Job{
				Name:   fmt.Sprintf("job-provisions/job/ci/job/%v", c.Spec.JobProvisioning),
				Config: jc,
			},
		},
		Status: jenkinsv1alpha1.JenkinsFolderStatus{
			Available:       false,
			LastTimeUpdated: time.Time{},
			Status:          util.StatusInProgress,
		},
	}
	if err := h.clientSet.Client.Create(context.TODO(), jf); err != nil {
		return errors.Wrapf(err, "couldn't create jenkins folder %v", "name")
	}
	return nil
}

func (h PutJenkinsFolder) getJenkinsFolder(name, namespace string) (*jenkinsv1alpha1.JenkinsFolder, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	i := &jenkinsv1alpha1.JenkinsFolder{}
	if err := h.clientSet.Client.Get(context.TODO(), nsn, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get instance by owner %v", name)
	}
	return i, nil
}

func newTrue() *bool {
	b := true
	return &b
}

func (h PutJenkinsFolder) updateFinishStatus(c *v1alpha1.Codebase) error {
	c.Status = v1alpha1.CodebaseStatus{
		Status:          util.StatusFinished,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          v1alpha1.SetupDeploymentTemplates,
		Result:          v1alpha1.Success,
		Value:           "active",
		FailureCount:    0,
	}
	return h.updateStatus(c)
}

func (h PutJenkinsFolder) updateStatus(c *v1alpha1.Codebase) error {
	if err := h.clientSet.Client.Status().Update(context.TODO(), c); err != nil {
		if err := h.clientSet.Client.Update(context.TODO(), c); err != nil {
			return err
		}
	}
	return nil
}
func getRepositoryPath(codebaseName, strategy string, gitUrlPath *string) string {
	if strategy == consts.ImportStrategy {
		return *gitUrlPath
	}
	return "/" + codebaseName
}

func generateSshLink(repoPath string, gs *model.GitServer) string {
	l := fmt.Sprintf("ssh://%v@%v:%v%v", gs.GitUser, gs.GitHost, gs.SshPort, repoPath)
	log.Info("generated SSH link", "link", l)
	return l
}

func isJiraIntegrationEnabled(server *string) bool {
	if server != nil {
		return true
	}
	return false
}
