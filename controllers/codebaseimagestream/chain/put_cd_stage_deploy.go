package chain

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
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
	Tags      []jenkinsApi.Tag
}

const (
	dateLayout = "2006-01-02T15:04:05"
	logNameKey = "name"
)

func (h PutCDStageDeploy) ServeRequest(imageStream *codebaseApi.CodebaseImageStream) error {
	log := h.log.WithValues(logNameKey, imageStream.Name)
	log.Info("creating/updating CDStageDeploy.")

	if err := h.handleCodebaseImageStreamEnvLabels(imageStream); err != nil {
		return fmt.Errorf("failed to handle %v codebase image stream: %w", imageStream.Name, err)
	}

	log.Info("creating/updating CDStageDeploy has been finished.")

	return nil
}

func (h PutCDStageDeploy) handleCodebaseImageStreamEnvLabels(imageStream *codebaseApi.CodebaseImageStream) error {
	if imageStream.ObjectMeta.Labels == nil || len(imageStream.ObjectMeta.Labels) == 0 {
		h.log.Info("codebase image stream doesnt contain env labels. skip CDStageDeploy creating...")
		return nil
	}

	labelValueRegexp := regexp.MustCompile("^[-A-Za-z0-9_.]+/[-A-Za-z0-9_.]+$")

	for envLabel := range imageStream.ObjectMeta.Labels {
		if errs := validateCbis(imageStream, envLabel, labelValueRegexp); len(errs) != 0 {
			return errors.New(strings.Join(errs, "; "))
		}

		if err := h.putCDStageDeploy(envLabel, imageStream.Namespace, imageStream.Spec); err != nil {
			return err
		}
	}

	return nil
}

func validateCbis(imageStream *codebaseApi.CodebaseImageStream, envLabel string, labelValueRegexp *regexp.Regexp) []string {
	var errs []string

	if imageStream.Spec.Codebase == "" {
		errs = append(errs, "codebase is not defined in spec ")
	}

	if len(imageStream.Spec.Tags) == 0 {
		errs = append(errs, "tags are not defined in spec ")
	}

	if !labelValueRegexp.MatchString(envLabel) {
		errs = append(errs, "Label must be in format cd-pipeline-name/stage-name")
	}

	return errs
}

func (h PutCDStageDeploy) putCDStageDeploy(envLabel, namespace string, spec codebaseApi.CodebaseImageStreamSpec) error {
	name := generateCdStageDeployName(envLabel, spec.Codebase)

	stageDeploy, err := h.getCDStageDeploy(name, namespace)
	if err != nil {
		return fmt.Errorf("failed to get %v cd stage deploy: %w", name, err)
	}

	if stageDeploy != nil {
		h.log.Info("CDStageDeploy already exists. skip creating.", logNameKey, stageDeploy.Name)

		return &util.CDStageDeployHasNotBeenProcessedError{
			Message: fmt.Sprintf("%v has not been processed for previous version of application yet", name),
		}
	}

	cdsd, err := getCreateCommand(envLabel, name, namespace, spec.Codebase, spec.Tags)
	if err != nil {
		return fmt.Errorf("failed to construct command to create %v cd stage deploy: %w", name, err)
	}

	if err := h.create(cdsd); err != nil {
		return fmt.Errorf("failed to create %v cd stage deploy: %w", name, err)
	}

	return nil
}

func generateCdStageDeployName(env, codebase string) string {
	env = strings.ReplaceAll(env, "/", "-")
	return fmt.Sprintf("%v-%v", env, codebase)
}

func (h PutCDStageDeploy) getCDStageDeploy(name, namespace string) (*codebaseApi.CDStageDeploy, error) {
	h.log.Info("getting cd stage deploy", logNameKey, name)

	ctx := context.Background()
	i := &codebaseApi.CDStageDeploy{}
	nn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	if err := h.client.Get(ctx, nn, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to fetch CDStageDeploy resource %q: %w", name, err)
	}

	return i, nil
}

func getCreateCommand(envLabel, name, namespace, codebase string, tags []codebaseApi.Tag) (*cdStageDeployCommand, error) {
	env := strings.Split(envLabel, "/")

	lastTag, err := getLastTag(tags)
	if err != nil {
		return nil, fmt.Errorf("failed to create cdStageDeployCommand with name %v: %w", name, err)
	}

	return &cdStageDeployCommand{
		Name:      name,
		Namespace: namespace,
		Pipeline:  env[0],
		Stage:     env[1],
		Tag: jenkinsApi.Tag{
			Codebase: codebase,
			Tag:      lastTag.Name,
		},
		Tags: []jenkinsApi.Tag{
			{
				Codebase: codebase,
				Tag:      lastTag.Name,
			},
		},
	}, nil
}

func getLastTag(tags []codebaseApi.Tag) (codebaseApi.Tag, error) {
	var (
		latestTag     codebaseApi.Tag
		latestTagTime = time.Time{}
	)

	for i, s := range tags {
		if current, err := time.Parse(dateLayout, tags[i].Created); err == nil {
			if current.After(latestTagTime) {
				latestTagTime = current
				latestTag = s
			}
		}
	}

	if latestTag.Name == "" {
		return latestTag, errors.New("There are no valid tags")
	}

	return latestTag, nil
}

func (h PutCDStageDeploy) create(command *cdStageDeployCommand) error {
	log := h.log.WithValues(logNameKey, command.Name)
	log.Info("cd stage deploy is not present in cluster. start creating...")

	ctx := context.Background()
	stageDeploy := &codebaseApi.CDStageDeploy{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: util.V2APIVersion,
			Kind:       util.CDStageDeployKind,
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      command.Name,
			Namespace: command.Namespace,
		},
		Spec: codebaseApi.CDStageDeploySpec{
			Pipeline: command.Pipeline,
			Stage:    command.Stage,
			Tag:      command.Tag,
			Tags:     command.Tags,
		},
	}

	err := h.client.Create(ctx, stageDeploy)
	if err != nil {
		return fmt.Errorf("failed to create CDStageDeploy reasource %q: %w", command.Name, err)
	}

	log.Info("cd stage deploy has been created.")

	return nil
}
