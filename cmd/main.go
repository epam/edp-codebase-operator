package main

import (
	"context"
	"crypto/tls"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	routeApi "github.com/openshift/api/route/v1"
	tektonpipelineApi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonTriggersApi "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	buildInfo "github.com/epam/edp-common/pkg/config"

	codebaseApiV1 "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy"
	"github.com/epam/edp-codebase-operator/v2/controllers/cdstagedeploy/chain"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebasebranch"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebaseimagestream"
	"github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/controllers/integrationsecret"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraissuemetadata"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/telemetry"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-codebase-operator/v2/pkg/webhook"
)

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	codebaseOperatorLock                     = "edp-codebase-operator-lock"
	codebaseBranchMaxConcurrentReconcilesEnv = "CODEBASE_BRANCH_MAX_CONCURRENT_RECONCILES"
	logFailCtrlCreateMessage                 = "failed to create controller"
	telemetryDefaultDelay                    = time.Hour
	telemetrySendEvery                       = time.Hour * 24
	telemetryUrl                             = "https://telemetry.kuberocketci.io"
)

func main() {
	var (
		metricsAddr                                      string
		metricsCertPath, metricsCertName, metricsCertKey string
		webhookCertPath, webhookCertName, webhookCertKey string
		enableLeaderElection                             bool
		probeAddr                                        string
		secureMetrics                                    bool
		enableHTTP2                                      bool
		tlsOpts                                          []func(*tls.Config)
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.StringVar(&webhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate.")
	flag.StringVar(&webhookCertName, "webhook-cert-name", "tls.crt", "The name of the webhook certificate file.")
	flag.StringVar(&webhookCertKey, "webhook-cert-key", "tls.key", "The name of the webhook key file.")
	flag.StringVar(&metricsCertPath, "metrics-cert-path", "",
		"The directory that contains the metrics server certificate.")
	flag.StringVar(&metricsCertName, "metrics-cert-name", "tls.crt", "The name of the metrics server certificate file.")
	flag.StringVar(&metricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	v := buildInfo.Get()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")

		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	// Create watchers for metrics and webhooks certificates
	var metricsCertWatcher, webhookCertWatcher *certwatcher.CertWatcher

	// Initial webhook TLS options
	webhookTLSOpts := tlsOpts

	if len(webhookCertPath) > 0 {
		setupLog.Info("Initializing webhook certificate watcher using provided certificates",
			"webhook-cert-path", webhookCertPath, "webhook-cert-name", webhookCertName, "webhook-cert-key", webhookCertKey)

		var err error

		webhookCertWatcher, err = certwatcher.New(
			filepath.Join(webhookCertPath, webhookCertName),
			filepath.Join(webhookCertPath, webhookCertKey),
		)
		if err != nil {
			setupLog.Error(err, "Failed to initialize webhook certificate watcher")
			os.Exit(1)
		}

		webhookTLSOpts = append(webhookTLSOpts, func(config *tls.Config) {
			config.GetCertificate = webhookCertWatcher.GetCertificate
		})
	}

	webhookServer := ctrlwebhook.NewServer(ctrlwebhook.Options{
		TLSOpts: webhookTLSOpts,
	})

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	// If the certificate is not specified, controller-runtime will automatically
	// generate self-signed certificates for the metrics server. While convenient for development and testing,
	// this setup is not recommended for production.
	//
	// TODO(user): If you enable certManager, uncomment the following lines:
	// - [METRICS-WITH-CERTS] at config/default/kustomization.yaml to generate and use certificates
	// managed by cert-manager for the metrics server.
	// - [PROMETHEUS-WITH-CERTS] at config/prometheus/kustomization.yaml for TLS certification.
	if len(metricsCertPath) > 0 {
		setupLog.Info("Initializing metrics certificate watcher using provided certificates",
			"metrics-cert-path", metricsCertPath, "metrics-cert-name", metricsCertName, "metrics-cert-key", metricsCertKey)

		var err error

		metricsCertWatcher, err = certwatcher.New(
			filepath.Join(metricsCertPath, metricsCertName),
			filepath.Join(metricsCertPath, metricsCertKey),
		)
		if err != nil {
			setupLog.Error(err, "to initialize metrics certificate watcher", "error", err)
			os.Exit(1)
		}

		metricsServerOptions.TLSOpts = append(metricsServerOptions.TLSOpts, func(config *tls.Config) {
			config.GetCertificate = metricsCertWatcher.GetCertificate
		})
	}

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
	utilruntime.Must(codebaseApiV1.AddToScheme(scheme))
	utilruntime.Must(cdPipeApi.AddToScheme(scheme))
	utilruntime.Must(networkingV1.AddToScheme(scheme))
	utilruntime.Must(routeApi.AddToScheme(scheme))
	utilruntime.Must(tektonTriggersApi.AddToScheme(scheme))
	utilruntime.Must(tektonpipelineApi.AddToScheme(scheme))

	ns, err := util.GetWatchNamespace()
	if err != nil {
		setupLog.Error(err, "failed to get watch namespace")
		os.Exit(1)
	}

	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       codebaseOperatorLock,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{ns: {}},
		},
	})
	if err != nil {
		setupLog.Error(err, "failed to start manager")
		os.Exit(1)
	}

	ctrlLog := ctrl.Log.WithName("controllers")

	cdStageDeployCtrl := cdstagedeploy.NewReconcileCDStageDeploy(mgr.GetClient(), ctrlLog, chain.CreateChain)
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

	if err = gitserver.NewReconcileGitServer(mgr.GetClient()).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, logFailCtrlCreateMessage, "controller", "git-server")
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

	if err = integrationsecret.NewReconcileIntegrationSecret(mgr.GetClient()).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, logFailCtrlCreateMessage, "controller", "integration-secret")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhook.RegisterValidationWebHook(context.Background(), mgr, ns); err != nil {
			setupLog.Error(err, "failed to create webhook", "webhook", "Codebase")
			os.Exit(1)
		}
	}

	// +kubebuilder:scaffold:builder

	if metricsCertWatcher != nil {
		setupLog.Info("Adding metrics certificate watcher to manager")

		if err := mgr.Add(metricsCertWatcher); err != nil {
			setupLog.Error(err, "Unable to add metrics certificate watcher to manager")
			os.Exit(1)
		}
	}

	if webhookCertWatcher != nil {
		setupLog.Info("Adding webhook certificate watcher to manager")

		if err := mgr.Add(webhookCertWatcher); err != nil {
			setupLog.Error(err, "Unable to add webhook certificate watcher to manager")
			os.Exit(1)
		}
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "failed to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "failed to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")

	ctx := ctrl.SetupSignalHandler()

	telemetryEnabled, _ := strconv.ParseBool(os.Getenv("TELEMETRY_ENABLED"))
	if telemetryEnabled {
		go telemetry.NewCollector(ns, telemetryUrl, mgr.GetClient()).Start(ctx, getTelemetryDelay(), telemetrySendEvery)
	}

	if err := mgr.Start(ctx); err != nil {
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

func getTelemetryDelay() time.Duration {
	val, exists := os.LookupEnv("TELEMETRY_DELAY")
	if !exists {
		return telemetryDefaultDelay
	}

	d, err := strconv.Atoi(val)
	if err != nil {
		return telemetryDefaultDelay
	}

	return time.Duration(d) * time.Minute
}
