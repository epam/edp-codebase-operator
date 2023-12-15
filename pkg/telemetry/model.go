package telemetry

type CodebaseMetrics struct {
	Lang       string `json:"lang"`
	Framework  string `json:"framework"`
	BuildTool  string `json:"buildTool"`
	Strategy   string `json:"strategy"`
	Type       string `json:"type"`
	Versioning string `json:"versioning"`
}

type CdPipelineMetrics struct {
	DeploymentType string `json:"deploymentType"`
	NumberOfStages int    `json:"numberOfStages"`
}

type PlatformMetrics struct {
	CodebaseMetrics   []CodebaseMetrics   `json:"codebaseMetrics"`
	CdPipelineMetrics []CdPipelineMetrics `json:"cdPipelineMetrics"`
	GitProviders      []string            `json:"gitProviders"`
	JiraEnabled       bool                `json:"jiraEnabled"`
	RegistryType      string              `json:"registryType"`
	Version           string              `json:"version"`
}
