package chain

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/jiraserver/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	edpComponentApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"
)

const (
	edpComponentJiraType = "jira"
)

type PutJiraEDPComponent struct {
	next   handler.JiraServerHandler
	client client.Client
}

const statusFinished = "finished"

func (h PutJiraEDPComponent) ServeRequest(jira *codebaseApi.JiraServer) error {
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

func (h PutJiraEDPComponent) createEDPComponentIfNotExists(jira codebaseApi.JiraServer) error {
	icon, err := getIcon()
	if err != nil {
		return errors.Wrapf(err, "couldn't encode icon %v", jira.Name)
	}

	c := &edpComponentApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      jira.Name,
			Namespace: jira.Namespace,
		},
		Spec: edpComponentApi.EDPComponentSpec{
			Type:    edpComponentJiraType,
			Url:     jira.Spec.RootUrl,
			Icon:    *icon,
			Visible: true,
		},
	}
	if err := h.client.Create(context.TODO(), c); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			log.V(2).Info("edp component already exists. skip creating...", "name", jira.Name)
			return nil
		}
		return err
	}
	return nil
}

func getIcon() (*string, error) {
	f, err := os.Open(fmt.Sprintf("%v/img/jira.svg", util.GetAssetsDir()))
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(f)
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(content)
	return &encoded, nil
}
