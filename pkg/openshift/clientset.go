package openshift

import (
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

type OpenshiftClientSet struct {
	CoreClient *coreV1Client.CoreV1Client
}

func CreateOpenshiftClients() *OpenshiftClientSet {
	log.Print("Start creating openshift clients...")
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
	log.Print("Openshift clients was successfully created")
	return &OpenshiftClientSet{
		CoreClient: coreClient,
	}
}
