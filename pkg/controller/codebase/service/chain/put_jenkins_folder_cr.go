package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/platform"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
)

type PutJenkinsFolder struct {
	client client.Client
}

func NewPutJenkinsFolder(client client.Client) *PutJenkinsFolder {
	return &PutJenkinsFolder{client: client}
}

func (h *PutJenkinsFolder) ServeRequest(ctx context.Context, c *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase_name", c.Name)
	jfn := fmt.Sprintf("%v-%v", c.Name, "codebase")
	jfr, err := h.getJenkinsFolder(jfn, c.Namespace)
	if err != nil {
		return err
	}

	if jfr != nil {
		rLog.Info("jenkins folder already exists in cluster", "name", jfn)
		return nil
	}

	gs, err := util.GetGitServer(h.client, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}

	path := getRepositoryPath(c.Name, string(c.Spec.Strategy), c.Spec.GitUrlPath)
	sshLink := generateSshLink(path, gs)
	jpm := map[string]string{
		"PARAM":                    "true",
		"NAME":                     c.Name,
		"LANGUAGE":                 c.Spec.Lang,
		"BUILD_TOOL":               strings.ToLower(c.Spec.BuildTool),
		"DEFAULT_BRANCH":           c.Spec.DefaultBranch,
		"GIT_SERVER_CR_NAME":       gs.Name,
		"GIT_SERVER_CR_VERSION":    "v2",
		"GIT_CREDENTIALS_ID":       gs.NameSshKeySecret,
		"REPOSITORY_PATH":          sshLink,
		"JIRA_INTEGRATION_ENABLED": strconv.FormatBool(isJiraIntegrationEnabled(c.Spec.JiraServer)),
		"PLATFORM_TYPE":            platform.GetPlatformType(),
	}
	jc, _ := json.Marshal(jpm)

	rLog.Info("start creating jenkins folder...")
	if err := h.putJenkinsFolder(c, string(jc), jfn); err != nil {
		setFailedFields(c, codebaseApi.PutJenkinsFolder, err.Error())
		return err
	}
	rLog.Info("end creating jenkins folder...")
	return nil
}

func (h *PutJenkinsFolder) putJenkinsFolder(c *codebaseApi.Codebase, jc, jfn string) error {
	jf := &jenkinsApi.JenkinsFolder{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: util.V2APIVersion,
			Kind:       util.JenkinsFolderKind,
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      jfn,
			Namespace: c.Namespace,
			Labels: map[string]string{
				util.CodebaseLabelKey: c.Name,
			},
		},
		Spec: jenkinsApi.JenkinsFolderSpec{
			Job: &jenkinsApi.Job{
				Name:   fmt.Sprintf("job-provisions/job/ci/job/%v", *c.Spec.JobProvisioning),
				Config: jc,
			},
		},
		Status: jenkinsApi.JenkinsFolderStatus{
			Available:       false,
			LastTimeUpdated: metaV1.Now(),
			Status:          util.StatusInProgress,
		},
	}
	if err := h.client.Create(context.TODO(), jf); err != nil {
		return errors.Wrapf(err, "couldn't create jenkins folder %v", "name")
	}
	return nil
}

func (h *PutJenkinsFolder) getJenkinsFolder(name, namespace string) (*jenkinsApi.JenkinsFolder, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	i := &jenkinsApi.JenkinsFolder{}
	if err := h.client.Get(context.TODO(), nsn, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get instance by owner %v", name)
	}
	return i, nil
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
	return server != nil
}
