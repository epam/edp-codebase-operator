package chain

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/codebaseimagestream"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type PutCDStageDeploy struct {
	client client.Client
}

type cdStageDeployCommand struct {
	Name      string
	Namespace string
	Pipeline  string
	Stage     string
	Tag       codebaseApi.CodebaseTag
	Tags      []codebaseApi.CodebaseTag
}

func (h PutCDStageDeploy) ServeRequest(ctx context.Context, imageStream *codebaseApi.CodebaseImageStream) error {
	l := ctrl.LoggerFrom(ctx)

	l.Info("Creating CDStageDeploy.")

	if err := h.handleCodebaseImageStreamEnvLabels(ctx, imageStream); err != nil {
		return fmt.Errorf("failed to handle %v codebase image stream: %w", imageStream.Name, err)
	}

	l.Info("Creating CDStageDeploy has been finished.")

	return nil
}

func (h PutCDStageDeploy) handleCodebaseImageStreamEnvLabels(ctx context.Context, imageStream *codebaseApi.CodebaseImageStream) error {
	l := ctrl.LoggerFrom(ctx)

	if imageStream.ObjectMeta.Labels == nil || len(imageStream.ObjectMeta.Labels) == 0 {
		l.Info("CodebaseImageStream does not contain env labels. Skip CDStageDeploy creating.")
		return nil
	}

	labelValueRegexp := regexp.MustCompile("^[-A-Za-z0-9_.]+/[-A-Za-z0-9_.]+$")

	for envLabel := range imageStream.ObjectMeta.Labels {
		if errs := validateCbis(imageStream, envLabel, labelValueRegexp); len(errs) != 0 {
			return errors.New(strings.Join(errs, "; "))
		}

		if err := h.putCDStageDeploy(ctx, envLabel, imageStream.Namespace, imageStream.Spec); err != nil {
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
		errs = append(errs, "label must be in format cd-pipeline-name/stage-name")
	}

	return errs
}

func (h PutCDStageDeploy) putCDStageDeploy(ctx context.Context, envLabel, namespace string, spec codebaseApi.CodebaseImageStreamSpec) error {
	l := ctrl.LoggerFrom(ctx)
	// use name for CDStageDeploy, it is converted from envLabel and cdpipeline/stage now is cdpipeline-stage
	name := strings.ReplaceAll(envLabel, "/", "-")

	skip, err := h.skipCDStageDeployCreation(ctx, envLabel, namespace)
	if err != nil {
		return fmt.Errorf("failed to check if CDStageDeploy exists: %w", err)
	}

	if skip {
		l.Info("Skip CDStageDeploy creation.")

		return nil
	}

	cdsd, err := getCreateCommand(ctx, envLabel, name, namespace, spec.Codebase, spec.Tags)
	if err != nil {
		return fmt.Errorf("failed to construct command to create %v cd stage deploy: %w", name, err)
	}

	if err = h.create(ctx, cdsd); err != nil {
		return fmt.Errorf("failed to create %v cd stage deploy: %w", name, err)
	}

	return nil
}

func (h PutCDStageDeploy) skipCDStageDeployCreation(ctx context.Context, envLabel, namespace string) (bool, error) {
	l := ctrl.LoggerFrom(ctx)
	l.Info("Getting CDStageDeploys.")

	env := strings.Split(envLabel, "/")

	list := &codebaseApi.CDStageDeployList{}
	if err := h.client.List(
		ctx,
		list,
		client.InNamespace(namespace),
		client.MatchingLabels{
			codebaseApi.CdPipelineLabel: env[0],
			codebaseApi.CdStageLabel:    fmt.Sprintf("%s-%s", env[0], env[1]),
		},
	); err != nil {
		return false, fmt.Errorf("failed to get CDStageDeploys: %w", err)
	}

	switch len(list.Items) {
	case 0:
		l.Info("CDStageDeploy is not present in cluster.")
		return false, nil
	case 1:
		l.Info("One CDStageDeploy is present in cluster.")
		return false, nil
	default:
		l.Info("More than one CDStageDeploy is present in cluster.")
		return true, nil
	}
}

func getCreateCommand(ctx context.Context, envLabel, name, namespace, codebase string, tags []codebaseApi.Tag) (*cdStageDeployCommand, error) {
	env := strings.Split(envLabel, "/")

	lastTag, err := codebaseimagestream.GetLastTag(tags, ctrl.LoggerFrom(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get last tag: %w", err)
	}

	return &cdStageDeployCommand{
		Name:      name,
		Namespace: namespace,
		Pipeline:  env[0],
		Stage:     env[1],
		Tag: codebaseApi.CodebaseTag{
			Codebase: codebase,
			Tag:      lastTag.Name,
		},
		Tags: []codebaseApi.CodebaseTag{
			{
				Codebase: codebase,
				Tag:      lastTag.Name,
			},
		},
	}, nil
}

func (h PutCDStageDeploy) create(ctx context.Context, command *cdStageDeployCommand) error {
	l := ctrl.LoggerFrom(ctx)
	l.Info("CDStageDeploy is not present in cluster. Start creating.")

	stageDeploy := &codebaseApi.CDStageDeploy{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: util.V2APIVersion,
			Kind:       util.CDStageDeployKind,
		},
		ObjectMeta: metaV1.ObjectMeta{
			GenerateName: command.Name,
			Namespace:    command.Namespace,
		},
		Spec: codebaseApi.CDStageDeploySpec{
			Pipeline: command.Pipeline,
			Stage:    command.Stage,
			Tag:      command.Tag,
			Tags:     command.Tags,
		},
	}

	stageDeploy.SetLabels(map[string]string{
		codebaseApi.CdPipelineLabel: command.Pipeline,
		codebaseApi.CdStageLabel:    stageDeploy.GetStageCRName(),
	})

	stage := &pipelineApi.Stage{}
	if err := h.client.Get(
		ctx,
		types.NamespacedName{
			Name:      stageDeploy.GetStageCRName(),
			Namespace: command.Namespace,
		},
		stage,
	); err != nil {
		return fmt.Errorf("failed to get CDStage %s: %w", command.Stage, err)
	}

	if err := controllerutil.SetControllerReference(stage, stageDeploy, h.client.Scheme()); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	err := h.client.Create(ctx, stageDeploy)
	if err != nil {
		return fmt.Errorf("failed to create CDStageDeploy resource %q: %w", command.Name, err)
	}

	l.Info("CDStageDeploy has been created.")

	return nil
}
