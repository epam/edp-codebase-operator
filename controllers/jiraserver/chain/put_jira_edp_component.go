package chain

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	edpComponentApi "github.com/epam/edp-component-operator/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraserver/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
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

	if err := h.createEDPComponentIfNotExists(jira); err != nil {
		return fmt.Errorf("failed to create EDP component %v: %w", jira.Name, err)
	}

	jira.Status.Status = statusFinished
	jira.Status.DetailedMessage = ""

	rl.Info("end putting Jira EDP component...")

	return nextServeOrNil(h.next, jira)
}

func (h PutJiraEDPComponent) createEDPComponentIfNotExists(js *codebaseApi.JiraServer) error {
	ctx := context.Background()

	icon, err := getIcon()
	if err != nil {
		return fmt.Errorf("failed to encode icon %v: %w", js.Name, err)
	}

	c := &edpComponentApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      js.Name,
			Namespace: js.Namespace,
		},
		Spec: edpComponentApi.EDPComponentSpec{
			Type:    edpComponentJiraType,
			Url:     js.Spec.RootUrl,
			Icon:    *icon,
			Visible: true,
		},
	}

	err = h.client.Create(ctx, c)
	if err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			log.V(2).Info("edp component already exists. skip creating...", "name", js.Name)

			return nil
		}

		return fmt.Errorf("fail to create EDPComponent resource %q: %w", js.Name, err)
	}

	return nil
}

func getIcon() (*string, error) {
	p := path.Join(util.GetAssetsDir(), "img/jira.svg")

	f, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	reader := bufio.NewReader(f)

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read all file content: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(content)

	return &encoded, nil
}
