# codebase-operator

![Version: 2.26.0-SNAPSHOT](https://img.shields.io/badge/Version-2.26.0--SNAPSHOT-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.26.0-SNAPSHOT](https://img.shields.io/badge/AppVersion-2.26.0--SNAPSHOT-informational?style=flat-square)

A Helm chart for KubeRocketCI Codebase Operator

**Homepage:** <https://docs.kuberocketci.io/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| epmd-edp | <SupportEPMD-EDP@epam.com> | <https://solutionshub.epam.com/solution/kuberocketci> |
| sergk |  | <https://github.com/SergK> |

## Source Code

* <https://github.com/epam/edp-codebase-operator>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| annotations | object | `{}` |  |
| envs[0].name | string | `"RECONCILATION_PERIOD"` |  |
| envs[0].value | string | `"360"` |  |
| envs[1] | object | `{"name":"CODEBASE_BRANCH_MAX_CONCURRENT_RECONCILES","value":"3"}` | Maximum number of parallel reconciliation codebasebranches |
| global.platform | string | `"kubernetes"` | platform type that can be "kubernetes" or "openshift" |
| image.repository | string | `"epamedp/codebase-operator"` | EDP codebase-operator Docker image name. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/codebase-operator) |
| image.tag | string | `nil` | EDP codebase-operator Docker image tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/codebase-operator/tags) |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| jira.apiUrl | string | `"https://jiraeu-api.example.com"` | API URL for development |
| jira.credentialName | string | `"ci-jira"` | Name of secret with credentials to Jira server |
| jira.integration | bool | `false` | Flag to enable/disable Jira integration |
| jira.name | string | `"jira"` | JiraServer CR name |
| jira.quickLink | object | `{"enabled":true}` | Enable creation of QuickLink for Jira |
| jira.rootUrl | string | `"https://jiraeu.example.com"` | URL to Jira server |
| name | string | `"codebase-operator"` | component name |
| nodeSelector | object | `{}` |  |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| telemetryEnabled | bool | `true` | Flag to enable/disable telemetry |
| tolerations | list | `[]` |  |

