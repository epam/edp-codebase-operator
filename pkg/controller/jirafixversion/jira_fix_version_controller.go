package jirafixversion

import (
	"context"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira/adapter"
	"github.com/epmd-edp/codebase-operator/v2/pkg/client/jira/dto"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/jirafixversion/chain"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
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

var log = logf.Log.WithName("controller_jira_fix_version")

const (
	reconcilationPeriod        = "RECONCILATION_PERIOD"
	defaultReconcilationPeriod = "360"
)

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileJiraFixVersion{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("jira-fix-version-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*edpv1alpha1.JiraFixVersion)
			no := e.ObjectNew.(*edpv1alpha1.JiraFixVersion)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			return false
		},
	}

	// Watch for changes to primary resource GitServer
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.JiraFixVersion{}}, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileJiraFixVersion{}

const (
	codebaseKind = "Codebase"
	errorStatus  = "error"
)

type ReconcileJiraFixVersion struct {
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileJiraFixVersion) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling JiraFixVersion")

	i := &edpv1alpha1.JiraFixVersion{}
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

	jc, err := r.initJiraClient(*i)
	if err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, err
	}

	ch := chain.CreateDefChain(jc, r.client)
	if err := ch.ServeRequest(i); err != nil {
		setErrorStatus(i, err.Error())
		timeout := setFailureCount(i)
		log.Error(err, "couldn't set fix version", "name", i.Name)
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
func setFailureCount(version *edpv1alpha1.JiraFixVersion) time.Duration {
	timeout := util.GetTimeout(version.Status.FailureCount)
	log.V(2).Info("wait for next reconcilation", "next reconcilation in", timeout)
	version.Status.FailureCount += 1
	return timeout
}

func (r *ReconcileJiraFixVersion) setOwnerRef(version *edpv1alpha1.JiraFixVersion) error {
	c := &edpv1alpha1.Codebase{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: version.Namespace,
		Name:      version.Spec.CodebaseName,
	}, c)
	if err != nil {
		return err
	}

	if err := controllerutil.SetControllerReference(c, version, r.scheme); err != nil {
		return errors.Wrap(err, "cannot set owner ref for JiraFixVersion CR")
	}
	return nil
}

func setErrorStatus(version *edpv1alpha1.JiraFixVersion, msg string) {
	version.Status.Status = errorStatus
	version.Status.DetailedMessage = msg
}
func (r *ReconcileJiraFixVersion) updateStatus(instance *edpv1alpha1.JiraFixVersion) {
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		_ = r.client.Update(context.TODO(), instance)
	}
}

func (r *ReconcileJiraFixVersion) initJiraClient(version edpv1alpha1.JiraFixVersion) (*jira.Client, error) {
	server, err := r.getJiraServer(version)
	if err != nil {
		return nil, err
	}

	s, err := util.GetSecretData(r.client, server.Spec.CredentialName, server.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't get secret %v", server.Spec.CredentialName)
	}
	user := string(s.Data["username"])
	pwd := string(s.Data["password"])
	c, err := new(adapter.GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer(server.Spec.ApiUrl, user, pwd))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create Jira client")
	}
	return &c, nil
}

func (r *ReconcileJiraFixVersion) getJiraServer(version edpv1alpha1.JiraFixVersion) (*edpv1alpha1.JiraServer, error) {
	ref, err := util.GetOwnerReference(codebaseKind, version.GetOwnerReferences())
	if err != nil {
		return nil, err
	}

	c := &edpv1alpha1.Codebase{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: version.Namespace,
		Name:      ref.Name,
	}, c)
	if err != nil {
		return nil, err
	}

	server := &edpv1alpha1.JiraServer{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: version.Namespace,
		Name:      *c.Spec.JiraServer,
	}, server)
	if err != nil {
		return nil, err
	}

	return server, nil
}
