package jiraissuemetadata

import (
	"context"
	"fmt"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/adapter"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/dto"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraissuemetadata/chain"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const (
	reconcilePeriod        = "RECONCILATION_PERIOD"
	defaultReconcilePeriod = "360"
	codebaseKind           = "Codebase"
	errorStatus            = "error"
)

type ReconcileJiraIssueMetadata struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func (r *ReconcileJiraIssueMetadata) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*codebaseApi.JiraIssueMetadata)
			no := e.ObjectNew.(*codebaseApi.JiraIssueMetadata)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.JiraIssueMetadata{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileJiraIssueMetadata) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling JiraIssueMetadata")

	i := &codebaseApi.JiraIssueMetadata{}
	if err := r.Client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	defer r.updateStatus(ctx, i)

	if err := r.setOwnerRef(ctx, i); err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, err
	}

	js, err := r.getJiraServer(ctx, *i)
	if err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, err
	}
	if !js.Status.Available {
		log.Info("Waiting for Jira server become available.", "name", js.Name)
		return reconcile.Result{RequeueAfter: r.setFailureCount(i)}, nil
	}

	jc, err := r.initJiraClient(*js)
	if err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, err
	}

	ch, err := chain.CreateChain(i.Spec.Payload, jc, r.Client)
	if err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, err
	}

	if err := ch.ServeRequest(i); err != nil {
		setErrorStatus(i, err.Error())
		timeout := r.setFailureCount(i)
		log.Error(err, "couldn't set jira issue metadata", "name", i.Name)
		return reconcile.Result{RequeueAfter: timeout}, nil
	}

	duration, err := time.ParseDuration(lookup() + "m")
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{RequeueAfter: duration * time.Minute}, nil
}

func lookup() string {
	if value, ok := os.LookupEnv(reconcilePeriod); ok {
		return value
	}
	return defaultReconcilePeriod
}

// setFailureCount increments failure count and returns delay for next reconciliation
func (r *ReconcileJiraIssueMetadata) setFailureCount(metadata *codebaseApi.JiraIssueMetadata) time.Duration {
	timeout := util.GetTimeout(metadata.Status.FailureCount, 500*time.Millisecond)
	r.Log.V(2).Info("wait for next reconcilation", "next reconcilation in", timeout)
	metadata.Status.FailureCount += 1
	return timeout
}

func (r *ReconcileJiraIssueMetadata) setOwnerRef(ctx context.Context, metadata *codebaseApi.JiraIssueMetadata) error {
	c := &codebaseApi.Codebase{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: metadata.Namespace,
		Name:      metadata.Spec.CodebaseName,
	}, c)
	if err != nil {
		return err
	}

	if err := controllerutil.SetControllerReference(c, metadata, r.Scheme); err != nil {
		return errors.Wrap(err, "cannot set owner ref for JiraIssueMetadata CR")
	}
	return nil
}

func setErrorStatus(metadata *codebaseApi.JiraIssueMetadata, msg string) {
	metadata.Status.Status = errorStatus
	metadata.Status.DetailedMessage = msg
}
func (r *ReconcileJiraIssueMetadata) updateStatus(ctx context.Context, instance *codebaseApi.JiraIssueMetadata) {
	instance.Status.LastTimeUpdated = time.Now()
	err := r.Client.Status().Update(ctx, instance)
	if err != nil {
		_ = r.Client.Update(ctx, instance)
	}
}

func (r *ReconcileJiraIssueMetadata) initJiraClient(js codebaseApi.JiraServer) (*jira.Client, error) {
	s, err := util.GetSecretData(r.Client, js.Spec.CredentialName, js.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't get secret %v", js.Spec.CredentialName)
	}

	user := string(s.Data["username"])
	pwd := string(s.Data["password"])
	c, err := new(adapter.GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer(js.Spec.ApiUrl, user, pwd))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create Jira client")
	}
	return &c, nil
}

func (r *ReconcileJiraIssueMetadata) getJiraServer(ctx context.Context, metadata codebaseApi.JiraIssueMetadata) (*codebaseApi.JiraServer, error) {
	ref, err := util.GetOwnerReference(codebaseKind, metadata.GetOwnerReferences())
	if err != nil {
		return nil, err
	}

	c := &codebaseApi.Codebase{}
	err = r.Client.Get(ctx, types.NamespacedName{
		Namespace: metadata.Namespace,
		Name:      ref.Name,
	}, c)
	if err != nil {
		return nil, err
	}

	if c.Spec.JiraServer == nil {
		return nil, fmt.Errorf("codebase %v has disabled jira integration. skip JiraIssueMetadata %v reconcilation",
			c.Name, metadata.Name)
	}

	server := &codebaseApi.JiraServer{}
	err = r.Client.Get(ctx, types.NamespacedName{
		Namespace: metadata.Namespace,
		Name:      *c.Spec.JiraServer,
	}, server)
	if err != nil {
		return nil, err
	}

	return server, nil
}
