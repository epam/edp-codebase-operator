package main

import (
	"flag"
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/cdstagedeploy"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebasebranch"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/gittag"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/imagestreamtag"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraissuemetadata"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	perfAPi "github.com/epam/edp-perf-operator/v2/pkg/apis/edp/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"
	"os"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const codebaseOperatorLock = "edp-codebase-operator-lock"

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(codebaseApi.AddToScheme(scheme))

	utilruntime.Must(cdPipeApi.AddToScheme(scheme))

	utilruntime.Must(edpCompApi.AddToScheme(scheme))

	utilruntime.Must(jenkinsApi.AddToScheme(scheme))

	utilruntime.Must(perfAPi.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", util.RunningInCluster(),
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	mode, err := util.GetDebugMode()
	if err != nil {
		setupLog.Error(err, "unable to get debug mode value")
		os.Exit(1)
	}

	opts := zap.Options{
		Development: mode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ns, err := util.GetWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get watch namespace")
		os.Exit(1)
	}

	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       codebaseOperatorLock,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return apiutil.NewDynamicRESTMapper(cfg)
		},
		Namespace: ns,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctrlLog := ctrl.Log.WithName("controllers")

	cdStageDeployCtrl := cdstagedeploy.NewReconcileCDStageDeploy(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err := cdStageDeployCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "cd-stage-deploy")
		os.Exit(1)
	}

	codebaseCtrl := codebase.NewReconcileCodebase(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err := codebaseCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "codebase")
		os.Exit(1)
	}

	cbCtrl := codebasebranch.NewReconcileCodebaseBranch(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err := cbCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "codebase-branch")
		os.Exit(1)
	}

	gitServerCtrl := gitserver.NewReconcileGitServer(mgr.GetClient(), ctrlLog)
	if err := gitServerCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "git-server")
		os.Exit(1)
	}

	gitTagCtrl := gittag.NewReconcileGitTag(mgr.GetClient(), ctrlLog)
	if err := gitTagCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "git-tag")
		os.Exit(1)
	}

	istCtrl := imagestreamtag.NewReconcileImageStreamTag(mgr.GetClient(), ctrlLog)
	if err := istCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "image-stream-tag")
		os.Exit(1)
	}

	jimCtrl := jiraissuemetadata.NewReconcileJiraIssueMetadata(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err := jimCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "jira-issue-metadata")
		os.Exit(1)
	}

	jsCtrl := jiraserver.NewReconcileJiraServer(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err := jsCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "jira-server")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
