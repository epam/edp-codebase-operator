package chain

import (
	"context"
	"fmt"

	coreV1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-codebase-operator/v2/pkg/vcs"
)

// DeleteGitlabWebHook is a chain element to delete webhook.
type DeleteGitlabWebHook struct {
	client       client.Client
	gitLabClient *vcs.GitLabClient
}

// NewDeleteGitlabWebHook creates DeleteGitlabWebHook instance.
func NewDeleteGitlabWebHook(k8sClient client.Client, gitLabClient *vcs.GitLabClient) *DeleteGitlabWebHook {
	return &DeleteGitlabWebHook{client: k8sClient, gitLabClient: gitLabClient}
}

// ServeRequest deletes Gitlab webhook.
func (s *DeleteGitlabWebHook) ServeRequest(ctx context.Context, codebase *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase name", codebase.Name)
	rLog.Info("Start deleting Gitlab webhook...")

	if codebase.Status.WebHookID == 0 {
		rLog.Info("Webhook ID is empty. Skip deleting webhook.")

		return nil
	}

	gitServer := &codebaseApi.GitServer{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: codebase.Spec.GitServer, Namespace: codebase.Namespace}, gitServer); err != nil {
		rLog.Error(err, "Failed to delete webhook: unable to get git server", "git server name", codebase.Spec.GitServer)

		return nil
	}

	secret, err := s.getGitServerSecret(ctx, gitServer.Spec.NameSshKeySecret, codebase.Namespace)
	if err != nil {
		rLog.Error(err, "Failed to delete webhook: unable to get git server secret.")

		return nil
	}

	if codebase.Spec.GitUrlPath == nil {
		err = fmt.Errorf("unable to get project ID for codebase %s, git url path is empty", codebase.Name)
		rLog.Error(err, "Failed to delete webhook.")

		return nil
	}

	projectID := *codebase.Spec.GitUrlPath

	gitHost := getGitServerURL(gitServer)

	err = s.gitLabClient.DeleteWebHook(
		ctx,
		gitHost,
		string(secret.Data[util.GitServerSecretTokenField]),
		projectID,
		codebase.Status.WebHookID,
	)
	if err != nil {
		rLog.Error(err, "Failed to delete webhook.")

		return nil
	}

	rLog.Info("Gitlab webhook has been deleted successfully.")

	return nil
}

func (s *DeleteGitlabWebHook) getGitServerSecret(ctx context.Context, secretName, namespace string) (*coreV1.Secret, error) {
	secret := &coreV1.Secret{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: namespace}, secret); err != nil {
		return nil, fmt.Errorf("unable to get %v secret: %w", secretName, err)
	}

	if _, ok := secret.Data[util.GitServerSecretTokenField]; !ok {
		return nil, fmt.Errorf("unable to get %s field from %s secret", util.GitServerSecretTokenField, secretName)
	}

	return secret, nil
}
