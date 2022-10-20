package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-resty/resty/v2"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/gitprovider"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const (
	webhookTokenLength = 20
	gitLabIngressName  = "el-gitlab-listener"
	gitHubIngressName  = "el-github-listener"
)

// PutWebHook is a chain element to create webhook.
type PutWebHook struct {
	client      client.Client
	restyClient *resty.Client
}

// NewPutWebHook creates PutWebHook instance.
func NewPutWebHook(k8sClient client.Client, restyClient *resty.Client) *PutWebHook {
	return &PutWebHook{client: k8sClient, restyClient: restyClient}
}

// ServeRequest creates webhook.
func (s *PutWebHook) ServeRequest(ctx context.Context, codebase *codebaseApi.Codebase) error {
	rLog := log.WithValues("codebase name", codebase.Name)
	rLog.Info("Start putting webhook...")

	gitServer := &codebaseApi.GitServer{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: codebase.Spec.GitServer, Namespace: codebase.Namespace}, gitServer); err != nil {
		return s.processCodebaseError(
			codebase,
			fmt.Errorf("unable to get git server %s: %w", codebase.Spec.GitServer, err),
		)
	}

	if gitServer.Spec.GitProvider != codebaseApi.GitProviderGitlab &&
		gitServer.Spec.GitProvider != codebaseApi.GitProviderGithub {
		rLog.Info(fmt.Sprintf("Unsupported Git provider %s. Skip putting webhook.", gitServer.Spec.GitProvider))
		return nil
	}

	secret, err := s.getGitServerSecret(ctx, gitServer.Spec.NameSshKeySecret, gitServer.Namespace)
	if err != nil {
		return s.processCodebaseError(codebase, err)
	}

	if codebase.Spec.GitUrlPath == nil {
		return s.processCodebaseError(
			codebase,
			fmt.Errorf("unable to get project ID for codebase %s, git url path is empty", codebase.Name),
		)
	}

	gitProvider, err := gitprovider.NewProvider(gitServer, s.restyClient)
	if err != nil {
		return s.processCodebaseError(codebase, fmt.Errorf("unable to create git provider: %w", err))
	}

	projectID := codebase.Spec.GetProjectID()
	gitHost := getGitProviderAPIURL(gitServer)

	if codebase.Status.WebHookID != 0 {
		_, err = gitProvider.GetWebHook(
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

		if !errors.Is(err, gitprovider.ErrWebHookNotFound) {
			return s.processCodebaseError(codebase, fmt.Errorf("unable to get webhook: %w", err))
		}
	}

	webHookURL, err := s.getWebHookUrl(ctx, gitServer)
	if err != nil {
		return s.processCodebaseError(codebase, err)
	}

	webHook, err := gitProvider.CreateWebHook(
		ctx,
		gitHost,
		string(secret.Data[util.GitServerSecretTokenField]),
		projectID,
		string(secret.Data[util.GitServerSecretWebhookSecretField]),
		webHookURL,
	)
	if err != nil {
		return s.processCodebaseError(codebase, fmt.Errorf("unable to create web hook: %w", err))
	}

	codebase.Status.WebHookID = webHook.ID

	if err = setIntermediateSuccessFields(ctx, s.client, codebase, codebaseApi.PutWebHook); err != nil {
		return fmt.Errorf("unable to update codebase %s status: %w", codebase.Name, err)
	}

	rLog.Info("Webhook has been created successfully.")

	return nil
}

func (s *PutWebHook) getGitServerSecret(ctx context.Context, secretName, namespace string) (*coreV1.Secret, error) {
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

func (s *PutWebHook) getWebHookUrl(ctx context.Context, gitServer *codebaseApi.GitServer) (string, error) {
	var ingressName string

	switch gitServer.Spec.GitProvider {
	case codebaseApi.GitProviderGithub:
		ingressName = gitHubIngressName
	case codebaseApi.GitProviderGitlab:
		ingressName = gitLabIngressName
	default:
		return "", fmt.Errorf("unsupported git provider %s", gitServer.Spec.GitProvider)
	}

	ingress := &networkingV1.Ingress{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: ingressName, Namespace: gitServer.Namespace}, ingress); err != nil {
		return "", fmt.Errorf("unable to get %s ingress: %w", ingressName, err)
	}

	if len(ingress.Spec.Rules) == 0 {
		return "", fmt.Errorf("ingress %s doesn't have rules", ingressName)
	}

	return getHostWithProtocol(ingress.Spec.Rules[0].Host), nil
}

func (*PutWebHook) processCodebaseError(codebase *codebaseApi.Codebase, err error) error {
	setFailedFields(codebase, codebaseApi.PutWebHook, err.Error())

	return err
}
