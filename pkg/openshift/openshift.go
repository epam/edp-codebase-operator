package openshift

import (
	"business-app-handler-controller/models"
	"bytes"
	"encoding/json"
	"fmt"
	buildV1 "github.com/openshift/api/build/v1"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
)

func getBuildConfig(clientSet ClientSet, appSettings models.AppSettings, bcName string) (*buildV1.BuildConfig, error) {
	bc, err := clientSet.
		BuildClient.
		BuildConfigs(appSettings.CicdNamespace).
		Get(bcName, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Build config %v in Openshift hasn't been found", bcName)
		return nil, err
	}

	return bc, nil
}

func updateBuildConfig(clientSet ClientSet, appSettings models.AppSettings, envSettings models.EnvSettings,
	bcName string) (*buildV1.BuildConfig, error) {
	triggers := make([]buildV1.BuildTriggerPolicy, len(envSettings.Triggers))

	bc, err := getBuildConfig(clientSet, appSettings, bcName)
	if err != nil {
		return nil, err
	}

	for _, value := range envSettings.Triggers{
		if value.Type == "ImageStreamChange" {
			triggers = append(triggers, buildV1.BuildTriggerPolicy{
				Type:             "ImageChange",
				ImageChange:      &buildV1.ImageChangeTrigger{
					From:                 &v1.ObjectReference{
						Kind:            "ImageStreamTag",
						Namespace:       fmt.Sprintf("%v-meta", envSettings.Name),
						Name:            fmt.Sprintf("%v:latest", appSettings.Name),
					},
				},
			})
		}
	}

	bc.Spec.Triggers = triggers

	log.Printf("Triggers inside build config object has been updated")

	return bc, nil
}

func PatchBuildConfig(clientSet ClientSet, appSettings models.AppSettings, envs models.EnvSettings) error {
	bcName := fmt.Sprintf("%v-deploy-pipeline", envs.Name)

	bc, err := updateBuildConfig(clientSet, appSettings, envs, bcName)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(bc)
	if err != nil {
		return err
	}

	_, err = clientSet.
		BuildClient.
		BuildConfigs(appSettings.CicdNamespace).
		Patch(bcName, types.StrategicMergePatchType, buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}
