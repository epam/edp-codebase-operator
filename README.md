[![codecov](https://codecov.io/gh/epam/edp-codebase-operator/branch/master/graph/badge.svg?token=9C0CBQS08X)](https://codecov.io/gh/epam/edp-codebase-operator)

# Codebase Operator

| :heavy_exclamation_mark: Please refer to the [KubeRocketCI documentation](https://docs.kuberocketci.io/) to get the notion of the main concepts and guidelines. |
| --- |

Get acquainted with the Codebase Operator and the installation process as well as the local development, and architecture scheme.

## Overview

Codebase Operator is an KubeRocketCI operator that is responsible for provisioning codebase entities. Operator installation can be applied on two container orchestration platforms: OpenShift and Kubernetes.

_**NOTE:** Operator is platform-independent, that is why there is a unified instruction for deploying._

## Prerequisites

1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. KubeRocketCI project/namespace is deployed by following the [Install KubeRocketCI](https://docs.kuberocketci.io/) instruction.

## Installation

In order to install the Codebase Operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for local client, run "helm repo add":

     ```bash
     helm repo add epamedp https://epam.github.io/edp-helm-charts/stable
     ```

2. Choose available Helm chart version:

     ```bash
     helm search repo epamedp/codebase-operator -l
     NAME                           CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/codebase-operator      2.23.0          2.23.0          A Helm chart for KubeRocketCI Codebase Operator
     ```

    _**NOTE:** It is highly recommended to use the latest released version._

3. Full chart parameters available in [deploy-templates/README.md](deploy-templates/README.md).

4. Install operator in the <edp-project> namespace with the helm command; find below the installation command example:

    ```bash
    helm install codebase-operator epamedp/codebase-operator --version <chart_version> --namespace <edp-project> --set name=codebase-operator --set global.edpName=<edp-project> --set global.platform=<platform_type> --set jira.integration=false
    ```

5. Check the <edp-project> namespace that should contain operator deployment with your operator in a running status.

## Local Development

In order to develop the operator, first set up a local environment. For details, please refer to the [Developer Guide](https://docs.kuberocketci.io/docs/next/developer-guide) page.

Development versions are also available, please refer to the [snapshot helm chart repository](https://epam.github.io/edp-helm-charts/snapshot/) page.

### Related Articles

- [Architecture Scheme of Codebase Operator](docs/arch.md)
- [Install KubeRocketCI](https://docs.kuberocketci.io/docs/next/operator-guide/installation-overview)
