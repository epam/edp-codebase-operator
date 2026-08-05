# codebase-operator

![Version: 2.35.0-SNAPSHOT](https://img.shields.io/badge/Version-2.35.0--SNAPSHOT-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.35.0-SNAPSHOT](https://img.shields.io/badge/AppVersion-2.35.0--SNAPSHOT-informational?style=flat-square)

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
| branchStaleCheckInterval | string | `"24h"` | How often the operator verifies that codebase branches still exist in git, marking missing ones with the Stale condition and the app.edp.epam.com/stale label. Accepts Go duration strings (e.g. 24h, 30m); "0" disables the check. |
| caCerts.enabled | bool | `false` | Mount additional CA certificates from an existing secret, e.g. for integrations behind a self-signed or private CA. |
| caCerts.secret | string | `"custom-ca-certificates"` | Name of an existing secret with CA certificates. Each key must hold a PEM-encoded certificate (a key may also hold a bundle of concatenated certificates); key names are arbitrary. Example: `kubectl create secret generic custom-ca-certificates --from-file=ca.crt=my-root-ca.pem` |
| enableWebhooks | bool | `true` | Enable webhook and cert-manager certificate resources. Webhooks require cert-manager to be installed in the cluster. |
| envs[0].name | string | `"RECONCILATION_PERIOD"` |  |
| envs[0].value | string | `"360"` |  |
| envs[1] | object | `{"name":"CODEBASE_BRANCH_MAX_CONCURRENT_RECONCILES","value":"3"}` | Maximum number of parallel reconciliation codebasebranches |
| global.gatewayApi | object | `{"gatewayName":"main-gateway","gatewayNamespace":"envoy-gateway-system"}` | Gateway API parent Gateway that EventListener HTTPRoutes attach to (used when ingressController=envoy) |
| global.gatewayApi.gatewayName | string | `"main-gateway"` | Name of the parent Gateway resource |
| global.gatewayApi.gatewayNamespace | string | `"envoy-gateway-system"` | Namespace of the parent Gateway resource |
| global.platform | string | `"kubernetes"` | platform type that can be "kubernetes" or "openshift" |
| image.registry | string | `""` | KubeRocketCI codebase-operator Docker image registry. |
| image.repository | string | `"epamedp/codebase-operator"` | KubeRocketCI codebase-operator Docker image name. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/codebase-operator) |
| image.tag | string | `nil` | KubeRocketCI codebase-operator Docker image tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/codebase-operator/tags) |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| imagePullSecrets | list | `[]` | Optional array of imagePullSecrets containing private registry credentials # Ref: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry |
| ingressController | string | `"nginx"` | Ingress controller for the GitServer EventListener webhook: "nginx" (Ingress) or "envoy" (Gateway API HTTPRoute) |
| jira.apiUrl | string | `"https://jiraeu-api.example.com"` | API URL for development |
| jira.credentialName | string | `"ci-jira"` | Name of secret with credentials to Jira server |
| jira.integration | bool | `false` | Flag to enable/disable Jira integration |
| jira.name | string | `"jira"` | JiraServer CR name |
| jira.quickLink | object | `{"enabled":true}` | Enable creation of QuickLink for Jira |
| jira.rootUrl | string | `"https://jiraeu.example.com"` | URL to Jira server |
| knownHosts.entries | string | `""` (no self-hosted servers pinned) | Host keys for self-hosted git servers, in known_hosts format, one per line. Obtain them with `ssh-keyscan -t rsa,ecdsa,ed25519 -p <port> <host>` and verify the fingerprints out-of-band before trusting them. Servers on a port other than 22 must use the bracket form, e.g. `[git.example.com]:2222 ssh-ed25519 AAAA...`. |
| knownHosts.includeDefaultProviders | bool | `true` | Include the shipped host keys for github.com, gitlab.com and bitbucket.org. Disable only if you pin these hosts yourself through `entries`. |
| name | string | `"codebase-operator"` | component name |
| nodeSelector | object | `{}` |  |
| podLabels | object | `{}` | Labels to be added to the pod |
| podSecurityContext | object | `{"runAsNonRoot":true}` | Pod Security Context Ref: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/ |
| resources.limits.memory | string | `"1Gi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"256Mi"` |  |
| securityContext | object | `{"allowPrivilegeEscalation":false}` | Container Security Context Ref: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/ |
| telemetryEnabled | bool | `true` | Flag to enable/disable telemetry |
| tolerations | list | `[]` |  |

