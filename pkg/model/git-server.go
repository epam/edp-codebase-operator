package model

import (
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
)

var log = ctrl.Log.WithName("git-server-model")

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
	GitHost          string
	GitUser          string
	HttpsPort        int32
	SshPort          int32
	NameSshKeySecret string
	ActionLog        ActionLog
	Namespace        string
	Name             string
}

type RepositoryData struct {
	User          string
	Key           string
	Port          int32
	RepositoryUrl string
	FolderToClone string
}

func ConvertToGitServer(k8sObj *codebaseApi.GitServer) *GitServer {
	log.Info("Start converting GitServer", "data", k8sObj.Name)

	spec := k8sObj.Spec

	actionLog := convertGitServerActionLog(&k8sObj.Status)

	return &GitServer{
		GitHost:          spec.GitHost,
		GitUser:          spec.GitUser,
		HttpsPort:        spec.HttpsPort,
		SshPort:          spec.SshPort,
		NameSshKeySecret: spec.NameSshKeySecret,
		ActionLog:        *actionLog,
		Namespace:        k8sObj.Namespace,
		Name:             k8sObj.Name,
	}
}

func convertGitServerActionLog(status *codebaseApi.GitServerStatus) *ActionLog {
	return &ActionLog{
		Event:           formatStatus(status.Status),
		DetailedMessage: status.DetailedMessage,
		Username:        status.Username,
		UpdatedAt:       status.LastTimeUpdated.Time,
		Action:          status.Action,
		Result:          status.Result,
	}
}

func formatStatus(status string) string {
	return strings.ToLower(strings.ReplaceAll(status, " ", "_"))
}
