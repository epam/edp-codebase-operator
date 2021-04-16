package jiraissuemetadata

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/adapter"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/dto"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraissuemetadata/chain"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

var log = logf.Log.WithName("controller_jira_issue_metadata")

const (
	reconcilationPeriod        = "RECONCILATION_PERIOD"
	defaultReconcilationPeriod = "360"
)

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileJiraIssueMetadata{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("jira-issue-metadata-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*edpv1alpha1.JiraIssueMetadata)
			no := e.ObjectNew.(*edpv1alpha1.JiraIssueMetadata)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			return false
		},
	}

	// Watch for changes to primary resource GitServer
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.JiraIssueMetadata{}}, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileJiraIssueMetadata{}

const (
	codebaseKind = "Codebase"
	errorStatus  = "error"
)

type ReconcileJiraIssueMetadata struct {
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileJiraIssueMetadata) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling JiraIssueMetadata")

	i := &edpv1alpha1.JiraIssueMetadata{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	defer r.updateStatus(i)

	if err := r.setOwnerRef(i); err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, err
	}

	js, err := r.getJiraServer(*i)
	if err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, err
	}
	if !js.Status.Available {
		log.Info("Waiting for Jira server become available.", "name", js.Name)
		return reconcile.Result{RequeueAfter: setFailureCount(i)}, nil
	}

	jc, err := r.initJiraClient(*js)
	if err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, err
	}

	ch, err := chain.CreateChain(i.Spec.Payload, jc, r.client)
	if err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, err
	}

	if err := ch.ServeRequest(i); err != nil {
		setErrorStatus(i, err.Error())
		timeout := setFailureCount(i)
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
	if value, ok := os.LookupEnv(reconcilationPeriod); ok {
		return value
	}
	return defaultReconcilationPeriod
}

// setFailureCount increments failure count and returns delay for next reconciliation
func setFailureCount(metadata *edpv1alpha1.JiraIssueMetadata) time.Duration {
	timeout := util.GetTimeout(metadata.Status.FailureCount, 500*time.Millisecond)
	log.V(2).Info("wait for next reconcilation", "next reconcilation in", timeout)
	metadata.Status.FailureCount += 1
	return timeout
}

func (r *ReconcileJiraIssueMetadata) setOwnerRef(metadata *edpv1alpha1.JiraIssueMetadata) error {
	c := &edpv1alpha1.Codebase{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: metadata.Namespace,
		Name:      metadata.Spec.CodebaseName,
	}, c)
	if err != nil {
		return err
	}

	if err := controllerutil.SetControllerReference(c, metadata, r.scheme); err != nil {
		return errors.Wrap(err, "cannot set owner ref for JiraIssueMetadata CR")
	}
	return nil
}

func setErrorStatus(metadata *edpv1alpha1.JiraIssueMetadata, msg string) {
	metadata.Status.Status = errorStatus
	metadata.Status.DetailedMessage = msg
}
func (r *ReconcileJiraIssueMetadata) updateStatus(instance *edpv1alpha1.JiraIssueMetadata) {
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		_ = r.client.Update(context.TODO(), instance)
	}
}

func (r *ReconcileJiraIssueMetadata) initJiraClient(js edpv1alpha1.JiraServer) (*jira.Client, error) {
	s, err := util.GetSecretData(r.client, js.Spec.CredentialName, js.Namespace)
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

func (r *ReconcileJiraIssueMetadata) getJiraServer(metadata edpv1alpha1.JiraIssueMetadata) (*edpv1alpha1.JiraServer, error) {
	ref, err := util.GetOwnerReference(codebaseKind, metadata.GetOwnerReferences())
	if err != nil {
		return nil, err
	}

	c := &edpv1alpha1.Codebase{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
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

	server := &edpv1alpha1.JiraServer{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: metadata.Namespace,
		Name:      *c.Spec.JiraServer,
	}, server)
	if err != nil {
		return nil, err
	}

	return server, nil
}
