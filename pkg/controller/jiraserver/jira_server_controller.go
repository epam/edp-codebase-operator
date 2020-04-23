package jiraserver

import (
	"context"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira/adapter"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira/dto"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

var log = logf.Log.WithName("controller_jira_server")

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileJiraServer{
		client:  mgr.GetClient(),
		scheme:  mgr.GetScheme(),
		factory: new(adapter.GoJiraAdapterFactory),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("jiraserver-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*edpv1alpha1.JiraServer)
			newObject := e.ObjectNew.(*edpv1alpha1.JiraServer)
			if oldObject.Status != newObject.Status {
				return false
			}
			return true
		},
	}

	if err = c.Watch(&source.Kind{Type: &edpv1alpha1.JiraServer{}}, &handler.EnqueueRequestForObject{}, p); err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileJiraServer{}

type ReconcileJiraServer struct {
	client  client.Client
	scheme  *runtime.Scheme
	factory jira.ClientFactory
}

func (r *ReconcileJiraServer) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rl := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	rl.V(2).Info("Reconciling JiraServer")

	i := &edpv1alpha1.JiraServer{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	defer r.updateStatus(i)

	connected, err := r.checkConnection(*i)
	i.Status.Available = err == nil && connected == true
	if err != nil {
		rl.Error(err, "couldn't establish connection to Jira server", "url", i.Spec.ApiUrl)
		return reconcile.Result{RequeueAfter: time.Minute}, nil
	}

	rl.Info("Reconciling JiraServer has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileJiraServer) updateStatus(instance *edpv1alpha1.JiraServer) {
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		_ = r.client.Update(context.TODO(), instance)
	}
}

func (r ReconcileJiraServer) checkConnection(jira edpv1alpha1.JiraServer) (bool, error) {
	s, err := r.getSecretData(jira.Spec.CredentialName, jira.Namespace)
	if err != nil {
		return false, err
	}
	user := string(s.Data["username"])
	pwd := string(s.Data["password"])
	jiraClient, err := r.factory.New(dto.ConvertSpecToJiraServer(jira.Spec.ApiUrl, user, pwd))
	if err != nil {
		return false, err
	}
	connected, err := jiraClient.Connected()
	if err != nil {
		return false, errors.Wrap(err, "couldn't connect to Jira server")
	}
	log.Info("connection to Jira server", "established", connected)
	return connected, nil
}

func (r ReconcileJiraServer) getSecretData(name, namespace string) (*coreV1.Secret, error) {
	s := &coreV1.Secret{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}
