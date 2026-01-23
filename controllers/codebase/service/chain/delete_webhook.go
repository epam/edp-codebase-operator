package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	coreV1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/gitprovider"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// DeleteWebHook is a chain element to delete webhook.
type DeleteWebHook struct {
	client      client.Client
	restyClient *resty.Client
	log         logr.Logger
}

// NewDeleteWebHook creates DeleteWebHook instance.
func NewDeleteWebHook(k8sClient client.Client, restyClient *resty.Client, log logr.Logger) *DeleteWebHook {
	return &DeleteWebHook{client: k8sClient, restyClient: restyClient, log: log}
}

// ServeRequest deletes webhook.
func (s *DeleteWebHook) ServeRequest(ctx context.Context, codebase *codebaseApi.Codebase) error {
	log := ctrl.LoggerFrom(ctx)

	if codebase.Spec.CiTool != util.CITekton {
		log.Info("Skip deleting webhook for non-Tekton CI tool")
		return nil
	}

	log.Info("Start deleting webhook...")

	if codebase.Status.GetWebHookRef() == "" {
		log.Info("Webhook ref is empty. Skip deleting webhook.")

		return nil
	}

	gitServer := &codebaseApi.GitServer{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: codebase.Spec.GitServer, Namespace: codebase.Namespace},
		gitServer,
	); err != nil {
		log.Error(err, "Failed to delete webhook: unable to get GitServer", "gitServer", codebase.Spec.GitServer)

		return nil
	}

	secret, err := s.getGitServerSecret(ctx, gitServer.Spec.NameSshKeySecret, codebase.Namespace)
	if err != nil {
		log.Error(err, "Failed to delete webhook: unable to get GitServer secret")

		return nil
	}

	gitProvider, err := gitprovider.NewProvider(
		gitServer,
		s.restyClient,
		string(secret.Data[util.GitServerSecretTokenField]),
	)
	if err != nil {
		log.Error(err, "Failed to delete webhook: unable to create git provider")

		return nil
	}

	projectID := codebase.Spec.GetProjectID()
	gitHost := gitprovider.GetGitProviderAPIURL(gitServer)

	err = gitProvider.DeleteWebHook(
		ctx,
		gitHost,
		string(secret.Data[util.GitServerSecretTokenField]),
		projectID,
		codebase.Status.GetWebHookRef(),
	)
	if err != nil {
		if errors.Is(err, gitprovider.ErrWebHookNotFound) {
			log.Info("Webhook was not found. Skip deleting webhook")

			return nil
		}

		log.Error(err, "Failed to delete webhook")

		return nil
	}

	log.Info("Webhook has been deleted successfully")

	return nil
}

func (s *DeleteWebHook) getGitServerSecret(ctx context.Context, secretName, namespace string) (*coreV1.Secret, error) {
	secret := &coreV1.Secret{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: namespace}, secret); err != nil {
		return nil, fmt.Errorf("failed to get %v secret: %w", secretName, err)
	}

	if _, ok := secret.Data[util.GitServerSecretTokenField]; !ok {
		return nil, fmt.Errorf("failed to get field %s from %s secret", util.GitServerSecretTokenField, secretName)
	}

	return secret, nil
}
