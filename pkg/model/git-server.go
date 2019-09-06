package model

import (
	"errors"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strconv"
	"strings"
	"time"
)

var log = logf.Log.WithName("git-server-model")

type ActionLog struct {
	Id              int
	Event           string
	DetailedMessage string
	Username        string
	UpdatedAt       time.Time
	Action          string
	ActionMessage   string
	Result          string
}

type GitServer struct {
	GitHost                  string
	GitUser                  string
	HttpsPort                int64
	SshPort                  int64
	NameSshKeySecret         string
	CreateCodeReviewPipeline bool
	ActionLog                ActionLog
	Tenant                   string
	Name                     string
}

type RepositoryData struct {
	User          string
	Key           string
	Port          int64
	RepositoryUrl string
	FolderToClone string
}

func ConvertToGitServer(k8sObj v1alpha1.GitServer) (*GitServer, error) {
	log.Info("Start converting GitServer", "data", k8sObj.Name)

	if &k8sObj == nil {
		return nil, errors.New("k8s git server object should not be nil")
	}
	spec := k8sObj.Spec

	actionLog := convertGitServerActionLog(k8sObj.Status)

	gitServer := GitServer{
		GitHost:                  spec.GitHost,
		GitUser:                  spec.GitUser,
		HttpsPort:                convertToInt(spec.HttpsPort),
		SshPort:                  convertToInt(spec.SshPort),
		NameSshKeySecret:         spec.NameSshKeySecret,
		CreateCodeReviewPipeline: spec.CreateCodeReviewPipeline,
		ActionLog:                *actionLog,
		Tenant:                   strings.TrimSuffix(k8sObj.Namespace, "-edp-cicd"),
		Name:                     k8sObj.Name,
	}

	return &gitServer, nil
}

func convertToInt(val string) int64 {
	res, _ := strconv.ParseInt(val, 10, 64)
	return res
}
func convertGitServerActionLog(status v1alpha1.GitServerStatus) *ActionLog {
	if &status == nil {
		return nil
	}

	return &ActionLog{
		Event:           formatStatus(status.Status),
		DetailedMessage: status.DetailedMessage,
		Username:        status.Username,
		UpdatedAt:       status.LastTimeUpdated,
		Action:          status.Action,
		Result:          status.Result,
	}
}

func formatStatus(status string) string {
	return strings.ToLower(strings.Replace(status, " ", "_", -1))
}
