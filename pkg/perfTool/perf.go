package perfTool

import (
	ClientSet "business-app-handler-controller/pkg/openshift"
	"bytes"
	"fmt"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"net/http"
	"os"
)

func GetPerfCredentials(clientSet ClientSet.OpenshiftClientSet, namespace string) (map[string][]byte) {
	secret, err := clientSet.CoreClient.Secrets(namespace).Get("perf-user", metav1.GetOptions{})
	if err != nil {
		return nil
	}
	return secret.Data
}

func GetPerfSettings(clientSet ClientSet.OpenshiftClientSet, namespace string) (map[string]string) {
	perfSettings, err := clientSet.CoreClient.ConfigMaps(namespace).Get("perf-settings", metav1.GetOptions{})
	if err != nil {
		return nil
	}
	return perfSettings.Data
}

func GetPerfToken(perfWebUrl string, perfUserLogin string, perfUserPassword string) (string) {

	payload := fmt.Sprintf(`username=%v&password=%v`, perfUserLogin, perfUserPassword)

	req, err := http.NewRequest("POST", perfWebUrl+"/api/v2/sso/token", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)

	if err != nil || resp.StatusCode != 200 || resp.Body == nil {
		log.Printf("Failed to get perf token. %v", err)
		os.Exit(1)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Printf("Cannot read response body. %v", err)
		os.Exit(1)
	}
	perfToken := string(bodyBytes)
	return perfToken
}
