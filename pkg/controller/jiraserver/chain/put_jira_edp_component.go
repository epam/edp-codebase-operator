package chain

import (
	"bufio"
	"context"
	"encoding/base64"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/jiraserver/chain/handler"
	edpApi "github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	"io/ioutil"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	jiraIconPath         = "/usr/local/bin/img/jira.svg"
	edpComponentJiraType = "jira"
)

type PutJiraEDPComponent struct {
	next   handler.JiraServerHandler
	client client.Client
}

const statusFinished = "finished"

func (h PutJiraEDPComponent) ServeRequest(jira *v1alpha1.JiraServer) error {
	rl := log.WithValues("jira server name", jira.Name)
	rl.V(2).Info("start putting Jira EDP component...")
	if err := h.createEDPComponentIfNotExists(*jira); err != nil {
		return errors.Wrapf(err, "couldn't create EDP component %v", jira.Name)
	}
	jira.Status.Status = statusFinished
	jira.Status.DetailedMessage = ""
	rl.Info("end putting Jira EDP component...")
	return nextServeOrNil(h.next, jira)
}

func (h PutJiraEDPComponent) createEDPComponentIfNotExists(jira v1alpha1.JiraServer) error {
	icon, err := getIcon()
	if err != nil {
		return errors.Wrapf(err, "couldn't encode icon %v", jira.Name)
	}

	c := &edpApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jira.Name,
			Namespace: jira.Namespace,
		},
		Spec: edpApi.EDPComponentSpec{
			Type: edpComponentJiraType,
			Url:  jira.Spec.RootUrl,
			Icon: *icon,
		},
	}
	if err := h.client.Create(context.TODO(), c); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.V(2).Info("edp component already exists. skip creating...", "name", jira.Name)
			return nil
		}
		return err
	}
	return nil
}

func getIcon() (*string, error) {
	f, err := os.Open(jiraIconPath)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(content)
	return &encoded, nil
}
