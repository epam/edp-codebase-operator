package chain

import (
	"context"
	"fmt"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"strings"
	"time"
)

type PutCDStageDeploy struct {
	client client.Client
	log    logr.Logger
}

type cdStageDeployCommand struct {
	Name      string
	Namespace string
	Pipeline  string
	Stage     string
	Tag       jenkinsApi.Tag
}

const dateLayout = "2006-01-02T15:04:05"

func (h PutCDStageDeploy) ServeRequest(imageStream *codebaseApi.CodebaseImageStream) error {
	log := h.log.WithValues("name", imageStream.Name)
	log.Info("creating/updating CDStageDeploy.")
	if err := h.handleCodebaseImageStreamEnvLabels(imageStream); err != nil {
		return errors.Wrapf(err, "couldn't handle %v codebase image stream", imageStream.Name)
	}
	log.Info("creating/updating CDStageDeploy has been finished.")
	return nil
}

func (h PutCDStageDeploy) handleCodebaseImageStreamEnvLabels(imageStream *codebaseApi.CodebaseImageStream) error {
	if imageStream.ObjectMeta.Labels == nil || len(imageStream.ObjectMeta.Labels) == 0 {
		h.log.Info("codebase image stream doesnt contain env labels. skip CDStageDeploy creating...")
		return nil
	}

	for envLabel := range imageStream.ObjectMeta.Labels {
		if err := h.putCDStageDeploy(envLabel, imageStream.Namespace, imageStream.Spec); err != nil {
			return err
		}
	}
	return nil
}

func (h PutCDStageDeploy) putCDStageDeploy(envLabel, namespace string, spec codebaseApi.CodebaseImageStreamSpec) error {
	name := generateCdStageDeployName(envLabel, spec.Codebase)
	stageDeploy, err := h.getCDStageDeploy(name, namespace)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v cd stage deploy", name)
	}

	if stageDeploy != nil {
		h.log.Info("CDStageDeploy already exists. skip creating.", "name", stageDeploy.Name)
		return &util.CDStageDeployHasNotBeenProcessed{
			Message: fmt.Sprintf("%v has not been processed for previous version of application yet", name),
		}
	}

	command := h.getCreateCommand(envLabel, name, namespace, spec.Codebase, spec.Tags)
	if err := h.create(command); err != nil {
		return errors.Wrapf(err, "couldn't create %v cd stage deploy", name)
	}
	return nil
}

func generateCdStageDeployName(env, codebase string) string {
	env = strings.Replace(env, "/", "-", -1)
	return fmt.Sprintf("%v-%v", env, codebase)
}

func (h PutCDStageDeploy) getCDStageDeploy(name, namespace string) (*codebaseApi.CDStageDeploy, error) {
	h.log.Info("getting cd stage deploy", "name", name)
	i := &codebaseApi.CDStageDeploy{}
	nn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := h.client.Get(context.TODO(), nn, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return i, nil
}

func (h PutCDStageDeploy) getCreateCommand(envLabel, name, namespace, codebase string, tags []codebaseApi.Tag) cdStageDeployCommand {
	env := strings.Split(envLabel, "/")
	return cdStageDeployCommand{
		Name:      name,
		Namespace: namespace,
		Pipeline:  env[0],
		Stage:     env[1],
		Tag: jenkinsApi.Tag{
			Codebase: codebase,
			Tag:      h.getLastTag(tags).Name,
		},
	}
}

func (h PutCDStageDeploy) getLastTag(tags []codebaseApi.Tag) codebaseApi.Tag {
	sort.Slice(tags, func(i, j int) bool {
		prev, err := parseTime(tags[i].Created)
		if err != nil {
			h.log.Error(fmt.Errorf("couldn't parse time"), "time", tags[i].Created)
			return false
		}
		next, err := parseTime(tags[j].Created)
		if err != nil {
			h.log.Error(fmt.Errorf("couldn't parse time"), "time", tags[j].Created)
			return false
		}
		return (*prev).Before(*next)
	})
	return tags[len(tags)-1]
}

func parseTime(date string) (*time.Time, error) {
	t, err := time.Parse(dateLayout, date)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (h PutCDStageDeploy) create(command cdStageDeployCommand) error {
	log := h.log.WithValues("name", command.Name)
	log.Info("cd stage deploy is not present in cluster. start creating...")

	stageDeploy := &codebaseApi.CDStageDeploy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: util.V2APIVersion,
			Kind:       util.CDStageDeployKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      command.Name,
			Namespace: command.Namespace,
		},
		Spec: codebaseApi.CDStageDeploySpec{
			Pipeline: command.Pipeline,
			Stage:    command.Stage,
			Tag:      command.Tag,
		},
	}
	if err := h.client.Create(context.TODO(), stageDeploy); err != nil {
		return err
	}
	log.Info("cd stage deploy has been created.")
	return nil
}
