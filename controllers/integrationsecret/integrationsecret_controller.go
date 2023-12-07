package integrationsecret

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	integrationSecretLabel                = "edp.epam.com/integration-secret"
	integrationSecretTypeLabel            = "app.edp.epam.com/secret-type"
	integrationSecretConnectionAnnotation = "edp.epam.com/integration-secret-connected"
	successConnectionRequeueTime          = time.Minute * 30
	failConnectionRequeueTime             = time.Minute * 1
)

type ReconcileIntegrationSecret struct {
	client client.Client
}

func NewReconcileIntegrationSecret(k8sClient client.Client) *ReconcileIntegrationSecret {
	return &ReconcileIntegrationSecret{client: k8sClient}
}

func (r *ReconcileIntegrationSecret) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return hasIntegrationSecretLabelLabel(event.Object)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return hasIntegrationSecretLabelLabel(updateEvent.ObjectNew)
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return hasIntegrationSecretLabelLabel(genericEvent.Object)
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build IntegrationSecret controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch;update;patch

// Reconcile reads secrets with integration-secret label and set connection status to the annotation.
func (r *ReconcileIntegrationSecret) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	secret := &corev1.Secret{}
	if err := r.client.Get(ctx, request.NamespacedName, secret); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get Secret: %w", err)
	}

	log := ctrl.LoggerFrom(ctx).WithValues("url", string(secret.Data["url"]))
	reachable := isReachable(ctx, secret)

	log.Info("Secret connection status", "reachable", reachable)

	if secret.GetAnnotations()[integrationSecretConnectionAnnotation] != strconv.FormatBool(reachable) {
		log.Info("Updating Secret connection status")

		if secret.GetAnnotations() == nil {
			secret.SetAnnotations(map[string]string{})
		}

		secret.GetAnnotations()[integrationSecretConnectionAnnotation] = strconv.FormatBool(reachable)

		if err := r.client.Update(ctx, secret); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update Secret: %w", err)
		}
	}

	requeue := successConnectionRequeueTime
	if !reachable {
		requeue = failConnectionRequeueTime
	}

	log.Info("Reconciling IntegrationSecret has been finished")

	return reconcile.Result{
		RequeueAfter: requeue,
	}, nil
}

func isReachable(ctx context.Context, secret *corev1.Secret) bool {
	log := ctrl.LoggerFrom(ctx).WithValues("url", string(secret.Data["url"]))

	var (
		url string
		req *resty.Request
	)

	switch secret.GetLabels()[integrationSecretTypeLabel] {
	case "sonarqube":
		url = "/api/system/ping"
		req = newRequestWithAuth(ctx, secret)
	case "nexus":
		url = "/service/rest/v1/status"
		req = newRequestWithAuth(ctx, secret)
	case "dependency-track":
		url = "/v1/team/self"
		req = newRequest(ctx, string(secret.Data["url"])).SetHeader("X-Api-Key", string(secret.Data["token"]))
	case "defectdojo":
		url = "/api/v2/user_profile"
		req = newRequest(ctx, string(secret.Data["url"])).SetHeader("Authorization", "Token "+string(secret.Data["token"]))
	default:
		url = "/"
		req = newRequest(ctx, string(secret.Data["url"]))
	}

	resp, err := req.Get(url)
	if err != nil {
		log.Info("Error during integration secret request", "error", err)
		return false
	}

	if !resp.IsSuccess() {
		log.Info("Integration secret is not reachable", "status", resp.Status())
		return false
	}

	return true
}

func newRequestWithAuth(ctx context.Context, secret *corev1.Secret) *resty.Request {
	r := newRequest(ctx, string(secret.Data["url"]))

	if _, ok := secret.Data["username"]; ok {
		return r.SetBasicAuth(string(secret.Data["username"]), string(secret.Data["password"]))
	}

	return r.SetAuthToken(string(secret.Data["token"]))
}

func newRequest(ctx context.Context, url string) *resty.Request {
	return resty.New().SetHostURL(url).SetTimeout(time.Second * 5).R().SetContext(ctx)
}

func hasIntegrationSecretLabelLabel(object client.Object) bool {
	_, ok := object.GetLabels()[integrationSecretLabel]
	return ok
}
