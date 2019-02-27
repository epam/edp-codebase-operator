package main

import (
	"business-app-handler-controller/pkg/apis"
	"business-app-handler-controller/pkg/controller"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	appsv1 "github.com/openshift/api/apps/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func printVersion() {
	log.Printf(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Printf(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Printf(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
	log.Printf(os.Getenv("KUBECONFIG"))
}

func main() {
	printVersion()
	flag.Parse()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Fatalf("Failed to get watch namespace. %v", err)
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Registering Components.")

	// Setup Scheme for all resources
	//Adding Openshift v1 resources to Scheme
	if err := appsv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal(err)
	}

	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Printf("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatalf("Manager exited non-zero: %v", err)
		os.Exit(1)
	}
}
