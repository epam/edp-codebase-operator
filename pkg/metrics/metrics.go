package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelineAPi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

type Collector struct {
	namespace  string
	metricsUrl string
	k8sClient  client.Client
}

func NewCollector(namespace, metricsUrl string, k8sClient client.Client) *Collector {
	return &Collector{namespace: namespace, metricsUrl: metricsUrl, k8sClient: k8sClient}
}

func (c *Collector) Start(ctx context.Context, delay, sendEvery time.Duration) {
	log := ctrl.Log.WithName("metrics-collector")

	go func() {
		timeToSend := time.Now().Add(delay)
		ticker := time.NewTicker(time.Second)

		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("Stop metrics collector")
				return
			case now := <-ticker.C:
				if timeToSend.Before(now) {
					if err := c.sendMetrics(ctx); err != nil {
						log.Error(err, "Failed to send metrics")
						return
					}

					log.Info("Metrics were sent")

					return
				}
			}
		}
	}()

	ticker := time.NewTicker(sendEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Stop metrics collector")
			return
		case <-ticker.C:
			if err := c.sendMetrics(ctx); err != nil {
				log.Error(err, "Failed to send metrics")
				break
			}

			log.Info("Metrics were sent")
		}
	}
}

func (c *Collector) sendMetrics(ctx context.Context) error {
	edpConfig := &corev1.ConfigMap{}
	if err := c.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: c.namespace,
		Name:      util.EdpConfigMap,
	}, edpConfig); err != nil {
		return fmt.Errorf("failed to get edp config: %w", err)
	}

	metrics := PlatformMetrics{}
	metrics.RegistryType = edpConfig.Data["container_registry_type"]
	metrics.Version = edpConfig.Data["edp_version"]

	codebases := &codebaseApi.CodebaseList{}
	if err := c.k8sClient.List(ctx, codebases, client.InNamespace(c.namespace)); err != nil {
		return fmt.Errorf("failed to get codebases: %w", err)
	}

	for i := 0; i < len(codebases.Items); i++ {
		metrics.CodebaseMetrics = append(metrics.CodebaseMetrics, CodebaseMetrics{
			Lang:       codebases.Items[i].Spec.Lang,
			Framework:  codebases.Items[i].Spec.Framework,
			BuildTool:  codebases.Items[i].Spec.BuildTool,
			Strategy:   string(codebases.Items[i].Spec.Strategy),
			Type:       codebases.Items[i].Spec.Type,
			Versioning: string(codebases.Items[i].Spec.Versioning.Type),
		})
	}

	gitProviders := &codebaseApi.GitServerList{}
	if err := c.k8sClient.List(ctx, gitProviders, client.InNamespace(c.namespace)); err != nil {
		return fmt.Errorf("failed to get git providers: %w", err)
	}

	if len(gitProviders.Items) > 0 {
		metrics.GitProviders = append(metrics.GitProviders, gitProviders.Items[0].Spec.GitProvider)
	}

	stages := &pipelineAPi.StageList{}
	if err := c.k8sClient.List(ctx, stages, client.InNamespace(c.namespace)); err != nil {
		return fmt.Errorf("failed to get stages: %w", err)
	}

	deploymentType := map[string]string{}
	stagesCount := map[string]int{}

	for i := 0; i < len(stages.Items); i++ {
		stagesCount[stages.Items[i].Spec.CdPipeline]++

		if stages.Items[i].Spec.TriggerType == "Auto" {
			deploymentType[stages.Items[i].Spec.CdPipeline] = "Auto"
		}
	}

	cdPipelines := &pipelineAPi.CDPipelineList{}
	if err := c.k8sClient.List(ctx, cdPipelines, client.InNamespace(c.namespace)); err != nil {
		return fmt.Errorf("failed to get cd pipelines: %w", err)
	}

	for i := 0; i < len(cdPipelines.Items); i++ {
		pipeDeployment := "Manual"
		if val, ok := deploymentType[cdPipelines.Items[i].Name]; ok {
			pipeDeployment = val
		}

		metrics.CdPipelineMetrics = append(metrics.CdPipelineMetrics, CdPipelineMetrics{
			DeploymentType: pipeDeployment,
			NumberOfStages: stagesCount[cdPipelines.Items[i].Name],
		})
	}

	jiraServers := &codebaseApi.JiraServerList{}
	if err := c.k8sClient.List(ctx, jiraServers, client.InNamespace(c.namespace)); err != nil {
		return fmt.Errorf("failed to get jira servers: %w", err)
	}

	metrics.JiraEnabled = len(jiraServers.Items) > 0

	resp, err := resty.New().
		SetHostURL(c.metricsUrl).
		R().
		SetContext(ctx).
		SetBody(map[string]PlatformMetrics{"platformMetrics": metrics}).
		Post("/v1/submit")
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to send metrics: http status code: %s, body: %s", resp.Status(), resp.String())
	}

	return nil
}
