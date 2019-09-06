package openshift

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	errWrap "github.com/pkg/errors"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("openshift-service")

type OpenshiftService struct {
	ClientSet *openshift.ClientSet
	Client    client.Client
}

func (s OpenshiftService) GetGitServer(name, namespace string) (*edpv1alpha1.GitServer, error) {
	log.Info("Start fetching GitServer resource from k8s", "name", name, "namespace", namespace)

	instance := &edpv1alpha1.GitServer{}
	err := s.Client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, instance)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, errWrap.Wrap(err, fmt.Sprintf("Git Server %v doesn't exist in k8s.", name))
		}
		return nil, err
	}

	log.Info("Git Server instance has been received", "name", name)

	return instance, nil
}

func (s OpenshiftService) GetSecret(secretName, namespace string) (*v1.Secret, error) {
	log.Info("Start fetching Secret resource from k8s", "secret name", secretName, "namespace", namespace)

	secret, err := s.ClientSet.CoreClient.
		Secrets(namespace).
		Get(secretName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) || k8serrors.IsForbidden(err) {
		return nil, err
	}

	log.Info("Secret has been fetched", "secret name", secretName, "namespace", namespace)

	return secret, nil
}
