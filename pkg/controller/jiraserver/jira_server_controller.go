package jiraserver

import (
	"context"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira/adapter"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira/dto"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/jiraserver/chain"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

var (
	log                = logf.Log.WithName("controller_jira_server")
	schemeGroupVersion = schema.GroupVersion{Group: "v1.edp.epam.com", Version: "v1alpha1"}
)

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	addKnownTypes(scheme)
	return &ReconcileJiraServer{
		client: mgr.GetClient(),
		scheme: scheme,
	}
}

func addKnownTypes(scheme *runtime.Scheme) {
	scheme.AddKnownTypes(schemeGroupVersion,
		&v1alpha1.EDPComponent{},
		&v1alpha1.EDPComponentList{},
	)
	metav1.AddToGroupVersion(scheme, schemeGroupVersion)
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

const statusError = "error"

type ReconcileJiraServer struct {
	client client.Client
	scheme *runtime.Scheme
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

	c, err := r.initJiraClient(*i)
	if err != nil {
		i.Status.Available = false
		return reconcile.Result{}, err
	}

	jiraHandler := chain.CreateDefChain(c, r.client)
	if err := jiraHandler.ServeRequest(i); err != nil {
		i.Status.Status = statusError
		i.Status.DetailedMessage = err.Error()
		return reconcile.Result{}, err
	}
	rl.Info("Reconciling JiraServer has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileJiraServer) updateStatus(instance *edpv1alpha1.JiraServer) {
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		_ = r.client.Update(context.TODO(), instance)
	}
}

func (r *ReconcileJiraServer) initJiraClient(jira edpv1alpha1.JiraServer) (jira.Client, error) {
	s, err := util.GetSecretData(r.client, jira.Spec.CredentialName, jira.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't get secret %v", jira.Spec.CredentialName)
	}
	user := string(s.Data["username"])
	pwd := string(s.Data["password"])
	c, err := new(adapter.GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer(jira.Spec.ApiUrl, user, pwd))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create Jira client")
	}
	return c, nil
}
