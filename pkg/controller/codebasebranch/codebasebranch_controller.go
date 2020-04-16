package codebasebranch

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/service"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
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

var log = logf.Log.WithName("codebase-branch-controller")

type CodebaseBranchService interface {
	Create(cr *edpv1alpha1.CodebaseBranch)
	Update(cr *edpv1alpha1.CodebaseBranch)
	Delete(cr *edpv1alpha1.CodebaseBranch)
}

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	c := mgr.GetClient()
	oc := openshift.CreateOpenshiftClients()
	oc.Client = c
	return &ReconcileCodebaseBranch{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		codebaseBranchService: service.CodebaseBranchService{
			Cs: *oc,
		},
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("codebasebranch-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*edpv1alpha1.CodebaseBranch)
			no := e.ObjectNew.(*edpv1alpha1.CodebaseBranch)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			return false
		},
	}

	err = c.Watch(&source.Kind{Type: &edpv1alpha1.CodebaseBranch{}}, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCodebaseBranch{}

type ReconcileCodebaseBranch struct {
	client                client.Client
	scheme                *runtime.Scheme
	codebaseBranchService service.CodebaseBranchService
}

func (r *ReconcileCodebaseBranch) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rl := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	rl.Info("Reconciling CodebaseBranch")

	i := &edpv1alpha1.CodebaseBranch{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	c, err := r.getCodebaseByBranch(*i)
	if err != nil {
		return reconcile.Result{RequeueAfter: 5 * time.Second},
			errors.Wrapf(err, "couldn't get codebase for branch %s", request.Name)
	}

	jfn := fmt.Sprintf("%v-%v", c.Name, "codebase")
	jf, err := r.getJenkinsFolder(jfn, c.Namespace)
	if err != nil {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, err
	}

	a := jf == nil || !jf.Status.Available
	if !c.Status.Available && a {
		log.Info("can't start reconcile for branch", "codebase", c.Name,
			"codebase status", c.Status.Available, "branch", request.Name, "jenkins folder", a)
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if c.Spec.Versioning.Type == "edp" && hasNewVersion(i) {
		err := r.processNewVersion(i)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	if err := r.codebaseBranchService.TriggerReleaseJob(i); err != nil {
		return reconcile.Result{}, err
	}

	rl.Info("Reconciling CodebaseBranch has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileCodebaseBranch) getCodebaseByBranch(branch edpv1alpha1.CodebaseBranch) (*edpv1alpha1.Codebase, error) {
	instance := &edpv1alpha1.Codebase{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: branch.Namespace,
		Name:      branch.Spec.CodebaseName,
	}, instance)

	if err != nil {
		return nil, err
	}

	return instance, nil
}

func (r *ReconcileCodebaseBranch) getJenkinsFolder(name, namespace string) (*v1alpha1.JenkinsFolder, error) {
	i := &v1alpha1.JenkinsFolder{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, i)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get jenkins folder %v", name)
	}
	return i, nil
}

func hasNewVersion(b *edpv1alpha1.CodebaseBranch) bool {
	return !util.SearchVersion(b.Status.VersionHistory, *b.Spec.Version)
}

func (r *ReconcileCodebaseBranch) processNewVersion(b *edpv1alpha1.CodebaseBranch) error {
	if err := r.codebaseBranchService.ResetBranchBuildCounter(b); err != nil {
		return err
	}

	if err := r.codebaseBranchService.ResetBranchSuccessBuildCounter(b); err != nil {
		return err
	}

	return r.codebaseBranchService.AppendVersionToTheHistorySlice(b)
}
