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
)

type PutCDStageDeploy struct {
	client client.Client
}

type cdStageDeployCommand struct {
	Name        string
	Namespace   string
	Pipeline    string
	Stage       string
	TriggerType string
	Tag         codebaseApi.CodebaseTag
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

	if errs := validateCbis(imageStream); len(errs) != 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	labelValueRegexp := regexp.MustCompile("^[-A-Za-z0-9_.]+/[-A-Za-z0-9_.]+$")

	for envLabel := range imageStream.ObjectMeta.Labels {
		if !labelValueRegexp.MatchString(envLabel) {
			l.Info("Label value does not match the pattern cd-pipeline-name/stage-name. Skip CDStageDeploy creating.")

			continue
		}

		if err := h.putCDStageDeploy(ctx, envLabel, imageStream.Namespace, imageStream.Spec); err != nil {
			return err
		}
	}

	return nil
}

func validateCbis(imageStream *codebaseApi.CodebaseImageStream) []string {
	var errs []string

	if imageStream.Spec.Codebase == "" {
		errs = append(errs, "codebase is not defined in spec ")
	}

	if len(imageStream.Spec.Tags) == 0 {
		errs = append(errs, "tags are not defined in spec ")
	}

	return errs
}

func (h PutCDStageDeploy) putCDStageDeploy(ctx context.Context, envLabel, namespace string, spec codebaseApi.CodebaseImageStreamSpec) error {
	l := ctrl.LoggerFrom(ctx)
	// use name for CDStageDeploy, it is converted from envLabel and cdpipeline/stage now is cdpipeline-stage
	name := strings.ReplaceAll(envLabel, "/", "-")
	env := strings.Split(envLabel, "/")
	pipeline := env[0]
	stage := env[1]
	stageCrName := fmt.Sprintf("%s-%s", pipeline, stage)

	stageCr := &pipelineApi.Stage{}
	if err := h.client.Get(
		ctx,
		types.NamespacedName{
			Name:      stageCrName,
			Namespace: namespace,
		},
		stageCr,
	); err != nil {
		return fmt.Errorf("failed to get CDStage %s: %w", stageCrName, err)
	}

	skip, err := h.skipCDStageDeployCreation(ctx, pipeline, namespace, stageCr)
	if err != nil {
		return fmt.Errorf("failed to check if CDStageDeploy exists: %w", err)
	}

	if skip {
		l.Info("Skip CDStageDeploy creation.")

		return nil
	}

	cdsd, err := getCreateCommand(ctx, pipeline, stage, name, namespace, spec.Codebase, stageCr.Spec.TriggerType, spec.Tags)
	if err != nil {
		return fmt.Errorf("failed to construct command to create %v cd stage deploy: %w", name, err)
	}

	if err = h.create(ctx, cdsd, stageCr); err != nil {
		return fmt.Errorf("failed to create %v cd stage deploy: %w", name, err)
	}

	return nil
}

func (h PutCDStageDeploy) skipCDStageDeployCreation(ctx context.Context, pipeline, namespace string, stage *pipelineApi.Stage) (bool, error) {
	l := ctrl.LoggerFrom(ctx)

	if !stage.IsAutoDeployTriggerType() {
		l.Info("CDStage trigger type is not Auto. Don't need to skip CDStageDeploy creation.")
		return false, nil
	}

	l.Info("Getting CDStageDeploys.")

	list := &codebaseApi.CDStageDeployList{}
	if err := h.client.List(
		ctx,
		list,
		client.InNamespace(namespace),
		client.MatchingLabels{
			codebaseApi.CdPipelineLabel: pipeline,
			codebaseApi.CdStageLabel:    stage.Name,
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

func getCreateCommand(
	ctx context.Context,
	pipeline, stage, name, namespace, codebase, triggerType string,
	tags []codebaseApi.Tag,
) (*cdStageDeployCommand, error) {
	lastTag, err := codebaseimagestream.GetLastTag(tags, ctrl.LoggerFrom(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get last tag: %w", err)
	}

	return &cdStageDeployCommand{
		Name:        name,
		Namespace:   namespace,
		Pipeline:    pipeline,
		Stage:       stage,
		TriggerType: triggerType,
		Tag: codebaseApi.CodebaseTag{
			Codebase: codebase,
			Tag:      lastTag.Name,
		},
	}, nil
}

func (h PutCDStageDeploy) create(ctx context.Context, command *cdStageDeployCommand, stage *pipelineApi.Stage) error {
	l := ctrl.LoggerFrom(ctx)
	l.Info("CDStageDeploy is not present in cluster. Start creating.")

	stageDeploy := &codebaseApi.CDStageDeploy{
		ObjectMeta: metaV1.ObjectMeta{
			GenerateName: command.Name,
			Namespace:    command.Namespace,
		},
		Spec: codebaseApi.CDStageDeploySpec{
			Pipeline:    command.Pipeline,
			Stage:       command.Stage,
			Tag:         command.Tag,
			Tags:        []codebaseApi.CodebaseTag{command.Tag},
			TriggerType: command.TriggerType,
		},
	}

	stageDeploy.SetLabels(map[string]string{
		codebaseApi.CdPipelineLabel: command.Pipeline,
		codebaseApi.CdStageLabel:    stageDeploy.GetStageCRName(),
	})

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
