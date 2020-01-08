package openshift

import (
	imageV1Client "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientSet struct {
	CoreClient  *coreV1Client.CoreV1Client
	ImageClient *imageV1Client.ImageV1Client
	Client      client.Client
}

func CreateOpenshiftClients() *ClientSet {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := config.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	coreClient, err := coreV1Client.NewForConfig(restConfig)
	if err != nil {
		log.Fatal(err)
	}
	imageClient, err := imageV1Client.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("[ERROR] %s", err)
	}
	log.Print("Openshift clients was successfully created")
	return &ClientSet{
		CoreClient:  coreClient,
		ImageClient: imageClient,
	}
}
