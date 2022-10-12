package chain

import (
	"context"
	"errors"
	"fmt"
	"strings"

	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-codebase-operator/v2/pkg/vcs"
)

const (
	webhookTokenLength = 20
	gitLabIngressName  = "el-gitlab-listener"
)

// PutGitlabWebHook is a chain element to create gitlab webhook.
type PutGitlabWebHook struct {
	client       client.Client
	gitLabClient *vcs.GitLabClient
}

// NewPutGitlabWebHook creates PutGitlabWebHook instance.
func NewPutGitlabWebHook(k8sClient client.Client, gitLabClient *vcs.GitLabClient) *PutGitlabWebHook {
	return &PutGitlabWebHook{client: k8sClient, gitLabClient: gitLabClient}
}

// ServeRequest creates gitlab webhook.
func (s *PutGitlabWebHook) ServeRequest(ctx context.Context, codebase *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase name", codebase.Name)
	rLog.Info("Start putting Gitlab webhook...")

	gitServer := &codebaseApi.GitServer{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: codebase.Spec.GitServer, Namespace: codebase.Namespace}, gitServer); err != nil {
		setFailedFields(codebase, codebaseApi.PutWebHook, err.Error())

		return fmt.Errorf("unable to get git server %s: %w", codebase.Spec.GitServer, err)
	}

	if strings.Contains(gitServer.Spec.GitHost, "github.com") {
		rLog.Info("Git server is GitHub. Skip putting webhook.")
		return nil
	}

	secret, err := s.getGitServerSecret(ctx, gitServer.Spec.NameSshKeySecret, gitServer.Namespace)
	if err != nil {
		setFailedFields(codebase, codebaseApi.PutWebHook, err.Error())

		return err
	}

	if codebase.Spec.GitUrlPath == nil {
		err = fmt.Errorf("unable to get project ID for codebase %s, git url path is empty", codebase.Name)

		setFailedFields(codebase, codebaseApi.PutWebHook, err.Error())

		return err
	}

	projectID := *codebase.Spec.GitUrlPath

	gitHost := getGitServerURL(gitServer)

	if codebase.Status.WebHookID != 0 {
		_, err = s.gitLabClient.GetWebHook(
			ctx,
			gitHost,
			string(secret.Data[util.GitServerSecretTokenField]),
			projectID,
			codebase.Status.WebHookID,
		)

		if err == nil {
			rLog.Info("Webhook already exists. Skip putting webhook.")

			return nil
		}

		if !errors.Is(err, vcs.ErrWebHookNotFound) {
			setFailedFields(codebase, codebaseApi.PutWebHook, err.Error())

			return fmt.Errorf("unable to get webhook: %w", err)
		}
	}

	webHookURL, err := s.getVCSUrl(ctx, codebase.Namespace)
	if err != nil {
		setFailedFields(codebase, codebaseApi.PutWebHook, err.Error())

		return err
	}

	webHook, err := s.gitLabClient.CreateWebHook(
		ctx,
		gitHost,
		string(secret.Data[util.GitServerSecretTokenField]),
		projectID,
		string(secret.Data[util.GitServerSecretWebhookSecretField]),
		webHookURL,
	)
	if err != nil {
		setFailedFields(codebase, codebaseApi.PutWebHook, err.Error())

		return fmt.Errorf("unable to create GitLab web hook: %w", err)
	}

	codebase.Status.WebHookID = webHook.ID

	if err = setIntermediateSuccessFields(ctx, s.client, codebase, codebaseApi.PutWebHook); err != nil {
		return fmt.Errorf("unable to update codebase %s status: %w", codebase.Name, err)
	}

	rLog.Info("Gitlab webhook has been created successfully.")

	return nil
}

func (s *PutGitlabWebHook) getGitServerSecret(ctx context.Context, secretName, namespace string) (*coreV1.Secret, error) {
	secret := &coreV1.Secret{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: namespace}, secret); err != nil {
		return nil, fmt.Errorf("unable to get %v secret: %w", secretName, err)
	}

	if _, ok := secret.Data[util.GitServerSecretTokenField]; !ok {
		return nil, fmt.Errorf("unable to get %s field from %s secret", util.GitServerSecretTokenField, secretName)
	}

	if token, ok := secret.Data[util.GitServerSecretWebhookSecretField]; !ok || len(token) == 0 {
		token, err := util.GenerateRandomString(webhookTokenLength)
		if err != nil {
			return nil, fmt.Errorf("unable to generate webhook secret: %w", err)
		}

		secret.Data[util.GitServerSecretWebhookSecretField] = []byte(token)

		if err = s.client.Update(ctx, secret); err != nil {
			return nil, fmt.Errorf("unable to update %s secret: %w", secretName, err)
		}
	}

	return secret, nil
}

func (s *PutGitlabWebHook) getVCSUrl(ctx context.Context, namespace string) (string, error) {
	ingress := &networkingV1.Ingress{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: gitLabIngressName, Namespace: namespace}, ingress); err != nil {
		return "", fmt.Errorf("unable to get %s ingress: %w", gitLabIngressName, err)
	}

	if len(ingress.Spec.Rules) == 0 {
		return "", fmt.Errorf("ingress %s doesn't have rules", gitLabIngressName)
	}

	return getHostWithProtocol(ingress.Spec.Rules[0].Host), nil
}
