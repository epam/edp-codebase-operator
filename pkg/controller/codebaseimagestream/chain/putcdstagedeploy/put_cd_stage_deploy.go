package putcdstagedeploy

import (
	"context"
	"fmt"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sort"
	"strings"
	"time"
)

type PutCDStageDeploy struct {
	Client client.Client
}

type cdStageDeployDTO struct {
	Pipeline string
	Stage    string
	Tags     []v1alpha1.CodebaseTag
}

const dateLayout = "2006-01-02T15:04:05"

var log = logf.Log.WithName("put-cd-stage-deploy-controller")

func (h PutCDStageDeploy) ServeRequest(imageStream *v1alpha1.CodebaseImageStream) error {
	vLog := log.WithValues("name", imageStream.Name)
	vLog.Info("creating/updating CDStageDeploy.")
	if err := h.handleCodebaseImageStreamEnvLabels(imageStream); err != nil {
		return errors.Wrapf(err, "couldn't handle %v codebase image stream", imageStream.Name)
	}
	vLog.Info("creating/updating CDStageDeploy has been finished.")
	return nil
}

func (h PutCDStageDeploy) handleCodebaseImageStreamEnvLabels(imageStream *v1alpha1.CodebaseImageStream) error {
	if imageStream.ObjectMeta.Labels == nil || len(imageStream.ObjectMeta.Labels) == 0 {
		log.Info("codebase image stream doesnt contain env labels. skip CDStageDeploy creating...")
		return nil
	}

	for key := range imageStream.ObjectMeta.Labels {
		if err := h.putCDStageDeploy(key, imageStream.Namespace, imageStream.Spec); err != nil {
			return err
		}
	}
	return nil
}

func (h PutCDStageDeploy) putCDStageDeploy(envLabel, namespace string, spec v1alpha1.CodebaseImageStreamSpec) error {
	name := strings.Replace(envLabel, "/", "-", -1)
	stageDeploy, err := h.getCDStageDeploy(name, namespace)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v cd stage deploy", name)
	}

	if stageDeploy == nil {
		createCommand := getCreateCommand(envLabel, spec.Codebase, spec.Tags)
		if err := h.create(name, namespace, createCommand); err != nil {
			return errors.Wrapf(err, "couldn't create %v cd stage deploy", name)
		}
		return nil
	}
	if err := h.update(stageDeploy, v1alpha1.CodebaseTag{
		Codebase: spec.Codebase,
		Tag:      getLastTag(spec.Tags).Name,
	}); err != nil {
		return errors.Wrapf(err, "couldn't update %v cd stage deploy", name)
	}
	return nil
}

func (h PutCDStageDeploy) getCDStageDeploy(name, namespace string) (*v1alpha1.CDStageDeploy, error) {
	log.Info("getting cd stage deploy", "name", name)
	i := &v1alpha1.CDStageDeploy{}
	nn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := h.Client.Get(context.TODO(), nn, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return i, nil
}

func getCreateCommand(envLabel, codebase string, tags []v1alpha1.Tag) cdStageDeployDTO {
	env := strings.Split(envLabel, "/")
	return cdStageDeployDTO{
		Pipeline: env[0],
		Stage:    env[1],
		Tags: []v1alpha1.CodebaseTag{
			{
				Codebase: codebase,
				Tag:      getLastTag(tags).Name,
			},
		},
	}
}

func getLastTag(tags []v1alpha1.Tag) v1alpha1.Tag {
	sort.Slice(tags, func(i, j int) bool {
		prev, err := parseTime(tags[i].Created)
		if err != nil {
			log.Error(fmt.Errorf("couldn't parse time"), "time", tags[i].Created)
			return false
		}
		next, err := parseTime(tags[j].Created)
		if err != nil {
			log.Error(fmt.Errorf("couldn't parse time"), "time", tags[j].Created)
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

func (h PutCDStageDeploy) create(name, namespace string, stageDeploy cdStageDeployDTO) error {
	vLog := log.WithValues("name", name)
	vLog.Info("cd stage deploy is not present in cluster. start creating...")

	stageDeployCommand := &v1alpha1.CDStageDeploy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: util.V2APIVersion,
			Kind:       util.CDStageDeployKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.CDStageDeploySpec{
			Pipeline: stageDeploy.Pipeline,
			Stage:    stageDeploy.Stage,
			Tags:     stageDeploy.Tags,
		},
	}
	if err := h.Client.Create(context.TODO(), stageDeployCommand); err != nil {
		return err
	}
	vLog.Info("cd stage deploy has been created.")
	return nil
}

func (h PutCDStageDeploy) update(stageDeploy *v1alpha1.CDStageDeploy, latestTag v1alpha1.CodebaseTag) error {
	vLog := log.WithValues("name", stageDeploy.Name)
	vLog.Info("cd stage deploy is present in cluster. start updating...")
	for i, targetTag := range stageDeploy.Spec.Tags {
		if targetTag.Codebase == latestTag.Codebase {
			stageDeploy.Spec.Tags[i].Tag = latestTag.Tag
			if err := h.Client.Update(context.TODO(), stageDeploy); err != nil {
				return err
			}
			vLog.Info("cd stage deploy has been updated.")
			return nil
		}
	}
	stageDeploy.Spec.Tags = append(stageDeploy.Spec.Tags, latestTag)
	if err := h.Client.Update(context.TODO(), stageDeploy); err != nil {
		return err
	}
	vLog.Info("cd stage deploy has been updated.")
	return nil
}
