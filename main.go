package main

import (
	"context"
	"flag"
	"os"
	"strconv"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	//+kubebuilder:scaffold:imports
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	buildInfo "github.com/epam/edp-common/pkg/config"
	edpCompApi "github.com/epam/edp-component-operator/api/v1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	perfAPi "github.com/epam/edp-perf-operator/v2/api/v1"

	codebaseApiV1 "github.com/epam/edp-codebase-operator/v2/api/v1"
	codebaseApiV1Alpha1 "github.com/epam/edp-codebase-operator/v2/api/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebaseimagestream"
	"github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/controllers/gittag"
	"github.com/epam/edp-codebase-operator/v2/controllers/imagestreamtag"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraissuemetadata"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-codebase-operator/v2/pkg/webhook"
)

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	port                                     = 9443
	codebaseOperatorLock                     = "edp-codebase-operator-lock"
	codebaseBranchMaxConcurrentReconcilesEnv = "CODEBASE_BRANCH_MAX_CONCURRENT_RECONCILES"
	logFailCtrlCreateMessage                 = "failed to create controller"
)

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	mode, err := util.GetDebugMode()
	if err != nil {
		setupLog.Error(err, "failed to get debug mode value")
		os.Exit(1)
	}

	opts := zap.Options{
		Development: mode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	v := buildInfo.Get()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog.Info("Starting the Codebase Operator",
		"version", v.Version,
		"git-commit", v.GitCommit,
		"git-tag", v.GitTag,
		"build-date", v.BuildDate,
		"go-version", v.Go,
		"go-client", v.KubectlVersion,
		"platform", v.Platform,
	)

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(codebaseApiV1Alpha1.AddToScheme(scheme))
	utilruntime.Must(codebaseApiV1.AddToScheme(scheme))
	utilruntime.Must(cdPipeApi.AddToScheme(scheme))
	utilruntime.Must(edpCompApi.AddToScheme(scheme))
	utilruntime.Must(jenkinsApi.AddToScheme(scheme))
	utilruntime.Must(perfAPi.AddToScheme(scheme))
	utilruntime.Must(networkingV1.AddToScheme(scheme))

	ns, err := util.GetWatchNamespace()
	if err != nil {
		setupLog.Error(err, "failed to get watch namespace")
		os.Exit(1)
	}

	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		Port:                   port,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       codebaseOperatorLock,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return apiutil.NewDynamicRESTMapper(cfg)
		},
		Namespace: ns,
	})
	if err != nil {
		setupLog.Error(err, "failed to start manager")
		os.Exit(1)
	}

	ctrlLog := ctrl.Log.WithName("controllers")

	cdStageDeployCtrl := cdstagedeploy.NewReconcileCDStageDeploy(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err = cdStageDeployCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to create controller", "controller", "cd-stage-deploy")
		os.Exit(1)
	}

	codebaseCtrl := codebase.NewReconcileCodebase(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err = codebaseCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to create controller", "controller", "codebase")
		os.Exit(1)
	}

	cbCtrl := codebasebranch.NewReconcileCodebaseBranch(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err = cbCtrl.SetupWithManager(mgr,
		getMaxConcurrentReconciles(codebaseBranchMaxConcurrentReconcilesEnv)); err != nil {
		setupLog.Error(err, logFailCtrlCreateMessage, "controller", "codebase-branch")
		os.Exit(1)
	}

	cisCtrl := codebaseimagestream.NewReconcileCodebaseImageStream(mgr.GetClient(), ctrlLog)
	if err = cisCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, logFailCtrlCreateMessage, "controller", "codebase-image-stream")
		os.Exit(1)
	}

	gitServerCtrl := gitserver.NewReconcileGitServer(mgr.GetClient(), ctrlLog)
	if err = gitServerCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, logFailCtrlCreateMessage, "controller", "git-server")
		os.Exit(1)
	}

	gitTagCtrl := gittag.NewReconcileGitTag(mgr.GetClient(), ctrlLog)
	if err = gitTagCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, logFailCtrlCreateMessage, "controller", "git-tag")
		os.Exit(1)
	}

	istCtrl := imagestreamtag.NewReconcileImageStreamTag(mgr.GetClient(), ctrlLog)
	if err = istCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, logFailCtrlCreateMessage, "controller", "image-stream-tag")
		os.Exit(1)
	}

	jimCtrl := jiraissuemetadata.NewReconcileJiraIssueMetadata(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err = jimCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, logFailCtrlCreateMessage, "controller", "jira-issue-metadata")
		os.Exit(1)
	}

	jsCtrl := jiraserver.NewReconcileJiraServer(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err = jsCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, logFailCtrlCreateMessage, "controller", "jira-server")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhook.RegisterValidationWebHook(context.Background(), mgr, ns); err != nil {
			setupLog.Error(err, "failed to create webhook", "webhook", "Codebase")
			os.Exit(1)
		}
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "failed to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "failed to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func getMaxConcurrentReconciles(envVar string) int {
	val, exists := os.LookupEnv(envVar)
	if !exists {
		return 1
	}

	n, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return 1
	}

	return int(n)
}
