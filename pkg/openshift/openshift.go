package openshift

import (
	"bytes"
	"codebase-operator/models"
	"encoding/json"
	"fmt"
	buildV1 "github.com/openshift/api/build/v1"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
)

func getBuildConfig(clientSet ClientSet, codebaseSettings models.CodebaseSettings, bcName string) (*buildV1.BuildConfig, error) {
	bc, err := clientSet.
		BuildClient.
		BuildConfigs(codebaseSettings.CicdNamespace).
		Get(bcName, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Build config %v in Openshift hasn't been found", bcName)
		return nil, err
	}

	return bc, nil
}

func updateBuildConfig(clientSet ClientSet, codebaseSettings models.CodebaseSettings, envSettings models.EnvSettings,
	bcName string) (*buildV1.BuildConfig, error) {

	bc, err := getBuildConfig(clientSet, codebaseSettings, bcName)
	if err != nil {
		return nil, err
	}

	for _, value := range envSettings.Triggers {
		if value.Type == "ImageStreamChange" {
			bc.Spec.Triggers = append(bc.Spec.Triggers, buildV1.BuildTriggerPolicy{
				Type: "ImageChange",
				ImageChange: &buildV1.ImageChangeTrigger{
					From: &v1.ObjectReference{
						Kind:      "ImageStreamTag",
						Namespace: fmt.Sprintf("%v-meta", envSettings.Name),
						Name:      fmt.Sprintf("%v-master:latest", codebaseSettings.Name),
					},
				},
			})
		}
	}

	log.Printf("Triggers inside build config object has been updated")

	return bc, nil
}

func PatchBuildConfig(clientSet ClientSet, codebaseSettings models.CodebaseSettings, env models.EnvSettings) error {
	bcName := fmt.Sprintf("%v-deploy-pipeline", env.Name)

	bc, err := updateBuildConfig(clientSet, codebaseSettings, env, bcName)
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
		BuildConfigs(codebaseSettings.CicdNamespace).
		Patch(bcName, types.StrategicMergePatchType, buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}
