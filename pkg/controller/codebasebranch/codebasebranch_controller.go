package codebasebranch

import (
	"context"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebasebranch/chain"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
	return &ReconcileCodebaseBranch{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
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
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileCodebaseBranch) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rl := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	rl.Info("Reconciling CodebaseBranch")

	cb := &edpv1alpha1.CodebaseBranch{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, cb); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	c, err := util.GetCodebase(r.client, cb.Spec.CodebaseName, cb.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}

	cbChain := chain.GetChain(c.Spec.CiTool, r.client)
	if err := cbChain.ServeRequest(cb); err != nil {
		log.Error(err, "an error has occurred while handling codebase branch", "name", cb.Name)
		switch err.(type) {
		case *util.CodebaseBranchReconcileError:
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		default:
			return reconcile.Result{}, err
		}
	}

	if err := r.setSuccessStatus(cb, edpv1alpha1.JenkinsConfiguration); err != nil {
		return reconcile.Result{},
			errors.Wrapf(err, "an error has been occurred while updating %v Codebase branch status", cb.Name)
	}

	rl.Info("Reconciling CodebaseBranch has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileCodebaseBranch) setSuccessStatus(cb *edpv1alpha1.CodebaseBranch, action edpv1alpha1.ActionType) error {
	cb.Status = edpv1alpha1.CodebaseBranchStatus{
		LastTimeUpdated:     time.Now(),
		Username:            "system",
		Action:              cb.Status.Action,
		Result:              edpv1alpha1.Success,
		Value:               "active",
		Status:              model.StatusFinished,
		VersionHistory:      cb.Status.VersionHistory,
		LastSuccessfulBuild: cb.Status.LastSuccessfulBuild,
		Build:               cb.Status.Build,
	}
	return r.updateStatus(cb)
}

func (r *ReconcileCodebaseBranch) updateStatus(cb *edpv1alpha1.CodebaseBranch) error {
	if err := r.client.Status().Update(context.TODO(), cb); err != nil {
		if err := r.client.Update(context.TODO(), cb); err != nil {
			return errors.Wrap(err, "couldn't update codebase branch status")
		}
	}
	log.V(2).Info("codebase branch status has been updated", "name", cb.Name)
	return nil
}
