<a name="unreleased"></a>
## [Unreleased]

### Routine

- Update current development version ([#217](https://github.com/epam/edp-codebase-operator/issues/217))


<a name="v2.28.0"></a>
## [v2.28.0] - 2025-07-21
### Features

- Support creation git branch from another branch ([#214](https://github.com/epam/edp-codebase-operator/issues/214))
- CodebaseBranch spec improvements ([#208](https://github.com/epam/edp-codebase-operator/issues/208))
- Allow usage of any branch name for CodebaseBranch ([#198](https://github.com/epam/edp-codebase-operator/issues/198))

### Bug Fixes

- CodebaseImageStream labels break auto-deploy ([#198](https://github.com/epam/edp-codebase-operator/issues/198))
- Prevent cleaning build number for CodebaseBranch ([#194](https://github.com/epam/edp-codebase-operator/issues/194))

### Code Refactoring

- Codebase label for CodebaseBranch ([#198](https://github.com/epam/edp-codebase-operator/issues/198))
- Update codebase and codebasebranch label to improve consistency ([#198](https://github.com/epam/edp-codebase-operator/issues/198))

### Routine

- Allow overriding securityContext fields ([#212](https://github.com/epam/edp-codebase-operator/issues/212))
- Add multi-architecture build support ([#210](https://github.com/epam/edp-codebase-operator/issues/210))
- Update current development version ([#196](https://github.com/epam/edp-codebase-operator/issues/196))
- Update current development version ([#192](https://github.com/epam/edp-codebase-operator/issues/192))
- Add ImagePullSecrets field support ([#190](https://github.com/epam/edp-codebase-operator/issues/190))
- Rename ConfigMap from 'edp-config' to 'krci-config' ([#183](https://github.com/epam/edp-codebase-operator/issues/183))
- Update current development version ([#187](https://github.com/epam/edp-codebase-operator/issues/187))


<a name="v2.27.2"></a>
## [v2.27.2] - 2025-04-11
### Bug Fixes

- Prevent cleaning build number for CodebaseBranch ([#194](https://github.com/epam/edp-codebase-operator/issues/194))


<a name="v2.27.1"></a>
## [v2.27.1] - 2025-03-31
### Routine

- Add ImagePullSecrets field support ([#190](https://github.com/epam/edp-codebase-operator/issues/190))
- Rename ConfigMap from 'edp-config' to 'krci-config' ([#183](https://github.com/epam/edp-codebase-operator/issues/183))


<a name="v2.27.0"></a>
## [v2.27.0] - 2025-03-22
### Features

- Add support for private repository creation ([#181](https://github.com/epam/edp-codebase-operator/issues/181))
- Rename platform versioning from edp to semver ([#176](https://github.com/epam/edp-codebase-operator/issues/176))
- Add validation webhook for protected resources ([#176](https://github.com/epam/edp-codebase-operator/issues/176))

### Bug Fixes

- Missing namespace selector for codebaseimagestream webhook ([#176](https://github.com/epam/edp-codebase-operator/issues/176))

### Routine

- Update Telemetry endpoint ([#174](https://github.com/epam/edp-codebase-operator/issues/174))
- Update current development version ([#170](https://github.com/epam/edp-codebase-operator/issues/170))


<a name="v2.26.0"></a>
## [v2.26.0] - 2025-01-24
### Routine

- Make QuickLink installation optional ([#168](https://github.com/epam/edp-codebase-operator/issues/168))
- Update current development version ([#166](https://github.com/epam/edp-codebase-operator/issues/166))


<a name="v2.25.0"></a>
## [v2.25.0] - 2024-12-12
### Bug Fixes

- Processing default branch for bitbucket ([#155](https://github.com/epam/edp-codebase-operator/issues/155))
- Ignore extra labels in CodebaseImageStream for auto-deploy ([#153](https://github.com/epam/edp-codebase-operator/issues/153))

### Routine

- Update current development version ([#158](https://github.com/epam/edp-codebase-operator/issues/158))
- Update Pull Request Template ([#21](https://github.com/epam/edp-codebase-operator/issues/21))
- Update current development version ([#150](https://github.com/epam/edp-codebase-operator/issues/150))


<a name="v2.24.1"></a>
## [v2.24.1] - 2024-11-05
### Bug Fixes

- Processing default branch for bitbucket ([#155](https://github.com/epam/edp-codebase-operator/issues/155))
- Ignore extra labels in CodebaseImageStream for auto-deploy ([#153](https://github.com/epam/edp-codebase-operator/issues/153))


<a name="v2.24.0"></a>
## [v2.24.0] - 2024-10-18
### Features

- Update Auto-stable deploy strategy logic ([#147](https://github.com/epam/edp-codebase-operator/issues/147))
- Add support for clone/create strategy for Bitbucket ([#143](https://github.com/epam/edp-codebase-operator/issues/143))
- Add auto-deploy type Auto-stable ([#139](https://github.com/epam/edp-codebase-operator/issues/139))
- Add support for Bitbucket webhooks ([#134](https://github.com/epam/edp-codebase-operator/issues/134))
- Add baseline support for bitbucket cloud ([#134](https://github.com/epam/edp-codebase-operator/issues/134))
- Implement fallback strategy for RPM packaging ([#123](https://github.com/epam/edp-codebase-operator/issues/123))
- Add support for RPM packages ([#116](https://github.com/epam/edp-codebase-operator/issues/116))
- Add dynamic pipelines names for CodebaseBranch ([#112](https://github.com/epam/edp-codebase-operator/issues/112))
- Add support for C/C++ languages ([#108](https://github.com/epam/edp-codebase-operator/issues/108))
- Remove deprecated v1alpha1 versions from the operator ([#101](https://github.com/epam/edp-codebase-operator/issues/101))
- Add cleanup of CodebaseImageStreams ([#97](https://github.com/epam/edp-codebase-operator/issues/97))
- Add GitServer reconciliation status ([#97](https://github.com/epam/edp-codebase-operator/issues/97))
- Skip Ingress creation if GitServer.spec.WebhookUrl is set ([#95](https://github.com/epam/edp-codebase-operator/issues/95))
- Set EventListener with edp prefix ([#83](https://github.com/epam/edp-codebase-operator/issues/83))
- Add label with GitServer name to EventListener ([#83](https://github.com/epam/edp-codebase-operator/issues/83))
- Add status for Codemie secret ([#86](https://github.com/epam/edp-codebase-operator/issues/86))
- Support GitServers with a custom name ([#83](https://github.com/epam/edp-codebase-operator/issues/83))

### Bug Fixes

- CDStageDeploy stuck in the running status ([#131](https://github.com/epam/edp-codebase-operator/issues/131))
- Align label for gen-ai chat secret ([#86](https://github.com/epam/edp-codebase-operator/issues/86))
- Integration secret shows connected status for invalid credentials ([#105](https://github.com/epam/edp-codebase-operator/issues/105))
- Integration secret shows connected status for invalid credentials ([#105](https://github.com/epam/edp-codebase-operator/issues/105))
- 401 error for correct username and password in JiraServer ([#103](https://github.com/epam/edp-codebase-operator/issues/103))
- Fix service name for the ingress ([#83](https://github.com/epam/edp-codebase-operator/issues/83))
- Update EventListener name to include edp prefix ([#83](https://github.com/epam/edp-codebase-operator/issues/83))
- Ensure ingress has proper labeling ([#83](https://github.com/epam/edp-codebase-operator/issues/83))

### Code Refactoring

- Fix hard-coded EventListener name ([#83](https://github.com/epam/edp-codebase-operator/issues/83))

### Testing

- Add check for EventListener creation ([#83](https://github.com/epam/edp-codebase-operator/issues/83))

### Routine

- Add set version block ([#145](https://github.com/epam/edp-codebase-operator/issues/145))
- Update alpine base image to v3.18.9 ([#141](https://github.com/epam/edp-codebase-operator/issues/141))
- Align argo-cd diff for OKD resources ([#76](https://github.com/epam/edp-codebase-operator/issues/76))
- Change apiVersion for OKD RBAC resources ([#76](https://github.com/epam/edp-codebase-operator/issues/76))
- Update default rpm spec and service templates ([#123](https://github.com/epam/edp-codebase-operator/issues/123))
- Add printcolumn for CDStageDeploy object ([#116](https://github.com/epam/edp-codebase-operator/issues/116))
- Do not rely on rpm changelog during build process ([#116](https://github.com/epam/edp-codebase-operator/issues/116))
- Hide rpm publish env variables ([#116](https://github.com/epam/edp-codebase-operator/issues/116))
- Allow to define binary path for RPM build ([#116](https://github.com/epam/edp-codebase-operator/issues/116))
- Deprecate codebasebranch properties ([#114](https://github.com/epam/edp-codebase-operator/issues/114))
- Set jira web-link icon to KubeRocketCI ([#108](https://github.com/epam/edp-codebase-operator/issues/108))
- Update documentation ([#108](https://github.com/epam/edp-codebase-operator/issues/108))
- Update codeql and codecov scan gh actions ([#83](https://github.com/epam/edp-codebase-operator/issues/83))
- Update current development version ([#81](https://github.com/epam/edp-codebase-operator/issues/81))

### Documentation

- Update changelog file for release notes ([#136](https://github.com/epam/edp-codebase-operator/issues/136))
- Update CHANGELOG md ([#129](https://github.com/epam/edp-codebase-operator/issues/129))


<a name="v2.23.0"></a>
## [v2.23.0] - 2024-06-12
### Features

- Implement queue for auto-deploy in pipelines ([#71](https://github.com/epam/edp-codebase-operator/issues/71))
- Update helm-template produced by operator ([#68](https://github.com/epam/edp-codebase-operator/issues/68))
- Process integration secret status for ghrc ([#63](https://github.com/epam/edp-codebase-operator/issues/63))

### Bug Fixes

- Deploy pipeline is failed if deploy with ingress ([#80](https://github.com/epam/edp-codebase-operator/issues/80))
- Incorrect handling of parallel CDStageDeploy ([#71](https://github.com/epam/edp-codebase-operator/issues/71))
- Set default pending status for CDStageDeploy ([#71](https://github.com/epam/edp-codebase-operator/issues/71))
- Align helm-docs to the latest changes ([#68](https://github.com/epam/edp-codebase-operator/issues/68))
- Fix helm-docs for new helm-chart ([#68](https://github.com/epam/edp-codebase-operator/issues/68))
- Align helm chart to more generalized configuration ([#68](https://github.com/epam/edp-codebase-operator/issues/68))
- Set CodebaseBranch failed status ([#65](https://github.com/epam/edp-codebase-operator/issues/65))
- Enable failed status for CodebaseBranch ([#65](https://github.com/epam/edp-codebase-operator/issues/65))
- Failed auto-deploy for the release branch ([#52](https://github.com/epam/edp-codebase-operator/issues/52))
- Fails to reprocess Codebase in case error ([#46](https://github.com/epam/edp-codebase-operator/issues/46))

### Routine

- Update argocd diff ([#76](https://github.com/epam/edp-codebase-operator/issues/76))
- Set Go 1.22 for CodeQL ([#58](https://github.com/epam/edp-codebase-operator/issues/58))
- Bump to Go 1.22 ([#58](https://github.com/epam/edp-codebase-operator/issues/58))
- Add codeowners file to the repo ([#56](https://github.com/epam/edp-codebase-operator/issues/56))
- Migrate from gerrit to github pipelines ([#54](https://github.com/epam/edp-codebase-operator/issues/54))
- Remove deprecated EDPComponents CRD ([#42](https://github.com/epam/edp-codebase-operator/issues/42))
- Update current development version ([#50](https://github.com/epam/edp-codebase-operator/issues/50))
- Bump alpine version ([#51](https://github.com/epam/edp-codebase-operator/issues/51))
- Update current development version ([#50](https://github.com/epam/edp-codebase-operator/issues/50))


<a name="v2.22.1"></a>
## [v2.22.1] - 2024-03-12
### Routine

- Bump alpine version ([#51](https://github.com/epam/edp-codebase-operator/issues/51))


<a name="v2.22.0"></a>
## [v2.22.0] - 2024-03-12
### Features

- Add delay for reconciling CDStageDeploy ([#47](https://github.com/epam/edp-codebase-operator/issues/47))
- Change naming approach for CDStageDeploy CR ([#47](https://github.com/epam/edp-codebase-operator/issues/47))
- Get EventListener Ingress by GitServer name ([#45](https://github.com/epam/edp-codebase-operator/issues/45))
- Change auto-deploy flow to a new approach ([#44](https://github.com/epam/edp-codebase-operator/issues/44))
- Migrate from EDPComponent to QuickLink ([#42](https://github.com/epam/edp-codebase-operator/issues/42))
- Migrate from EDPComponent to QuickLink ([#42](https://github.com/epam/edp-codebase-operator/issues/42))
- Change EDP icon for Jira weblinks ([#41](https://github.com/epam/edp-codebase-operator/issues/41))
- Implement a new QuickLink CRD ([#40](https://github.com/epam/edp-codebase-operator/issues/40))
- Enable secret status in annotation for argocd ([#39](https://github.com/epam/edp-codebase-operator/issues/39))
- Add webhookUrl optional property to the GitServer spec ([#37](https://github.com/epam/edp-codebase-operator/issues/37))
- Use event-listener name for Ingress and Route ([#36](https://github.com/epam/edp-codebase-operator/issues/36))

### Bug Fixes

- Do not fail reconciliation when CDStageDeploy exists ([#47](https://github.com/epam/edp-codebase-operator/issues/47))
- Auto-deploy failed into a remote cluster ([#49](https://github.com/epam/edp-codebase-operator/issues/49))
- Connection status for docker repository is failed ([#48](https://github.com/epam/edp-codebase-operator/issues/48))
- Empty auto-deploy flow logs attributes ([#44](https://github.com/epam/edp-codebase-operator/issues/44))
- Set success status for JiraServer ([#42](https://github.com/epam/edp-codebase-operator/issues/42))
- Empty project failed status ([#43](https://github.com/epam/edp-codebase-operator/issues/43))

### Code Refactoring

- Migrate EDPComponent CRD to edp-codebase-operator ([#40](https://github.com/epam/edp-codebase-operator/issues/40))

### Routine

- Remove Jira EDPComponent CR ([#168](https://github.com/epam/edp-codebase-operator/issues/168))
- Add jira QuickLink ([#168](https://github.com/epam/edp-codebase-operator/issues/168))
- Bump github.com/cloudflare/circl from 1.3.3 to 1.3.7 ([#38](https://github.com/epam/edp-codebase-operator/issues/38))
- Bump github.com/go-git/go-git/v5 to 5.11.0 ([#34](https://github.com/epam/edp-codebase-operator/issues/34))
- Bump golang.org/x/crypto from 0.14.0 to 0.17.0 ([#33](https://github.com/epam/edp-codebase-operator/issues/33))
- Update alpine image to the latest stable ([#35](https://github.com/epam/edp-codebase-operator/issues/35))
- Update current development version ([#32](https://github.com/epam/edp-codebase-operator/issues/32))

### Documentation

- Add more details regarding telemetry option ([#31](https://github.com/epam/edp-codebase-operator/issues/31))


<a name="v2.21.0"></a>
## [v2.21.0] - 2023-12-18
### Features

- Send telemetry metrics to the EDP backend ([#31](https://github.com/epam/edp-codebase-operator/issues/31))
- Add connection status to registry integration secret ([#29](https://github.com/epam/edp-codebase-operator/issues/29))
- Add connection error to integration secret ([#29](https://github.com/epam/edp-codebase-operator/issues/29))
- Add connection status to the integration secret annotation ([#29](https://github.com/epam/edp-codebase-operator/issues/29))
- Automate rekor uuid in release tag ([#27](https://github.com/epam/edp-codebase-operator/issues/27))

### Bug Fixes

- Fix deployment template for telemetry ([#31](https://github.com/epam/edp-codebase-operator/issues/31))
- Use minutes for telemetry delay ([#31](https://github.com/epam/edp-codebase-operator/issues/31))
- Integration status for dockerhub registry failed ([#29](https://github.com/epam/edp-codebase-operator/issues/29))
- Use EDP approach for label naming ([#29](https://github.com/epam/edp-codebase-operator/issues/29))

### Routine

- Update current development version ([#26](https://github.com/epam/edp-codebase-operator/issues/26))


<a name="v2.20.0"></a>
## [v2.20.0] - 2023-11-03
### Features

- Clean private repository secret ([#25](https://github.com/epam/edp-codebase-operator/issues/25))
- Migrate registry url from EDP component to edp-config configmap ([#20](https://github.com/epam/edp-codebase-operator/issues/20))
- Allow CodebaseImageStream to contain docker hub image ([#17](https://github.com/epam/edp-codebase-operator/issues/17))

### Bug Fixes

- Reset version for edp versioning in case the new release branch created is not working ([#24](https://github.com/epam/edp-codebase-operator/issues/24))
- Auto deploy failed with GitOps approach ([#23](https://github.com/epam/edp-codebase-operator/issues/23))

### Routine

- Add commit rule for pull request template ([#21](https://github.com/epam/edp-codebase-operator/issues/21))
- Bump golang.org/x/net from 0.8.0 to 0.17.0 ([#22](https://github.com/epam/edp-codebase-operator/issues/22))
- Upgrade pull request template ([#21](https://github.com/epam/edp-codebase-operator/issues/21))
- Remove jenkins admin-console perf operator logic ([#18](https://github.com/epam/edp-codebase-operator/issues/18))
- Update current development version ([#16](https://github.com/epam/edp-codebase-operator/issues/16))


<a name="v2.19.0"></a>
## [v2.19.0] - 2023-09-27
### Features

- Add valid error in status field of GitServer ([#15](https://github.com/epam/edp-codebase-operator/issues/15))

### Bug Fixes

- Build pipeline failed for go operator sdk on sonar step ([#12](https://github.com/epam/edp-codebase-operator/issues/12))

### Routine

- Upgrade Go to 1.20 ([#14](https://github.com/epam/edp-codebase-operator/issues/14))
- Update current development version ([#11](https://github.com/epam/edp-codebase-operator/issues/11))

### BREAKING CHANGE:


removed required fields from the GitServer status. CRD should be updated.


<a name="v2.18.1"></a>
## [v2.18.1] - 2023-09-25
### Bug Fixes

- Build pipeline failed for go operator sdk on sonar step ([#12](https://github.com/epam/edp-codebase-operator/issues/12))

### Routine

- Upgrade Go to 1.20 ([#14](https://github.com/epam/edp-codebase-operator/issues/14))
- Update CHANGELOG.md ([#85](https://github.com/epam/edp-codebase-operator/issues/85))


<a name="v2.18.0"></a>
## [v2.18.0] - 2023-09-20
### Code Refactoring

- Replace edpName with helm Release Namespace value ([#10](https://github.com/epam/edp-codebase-operator/issues/10))

### Routine

- Align secret name for new secret approach ([#65](https://github.com/epam/edp-codebase-operator/issues/65))
- Update current development version ([#9](https://github.com/epam/edp-codebase-operator/issues/9))


<a name="v2.17.1"></a>
## [v2.17.1] - 2023-12-11
### Features

- Make CI request timeouts configurable ([#30](https://github.com/epam/edp-codebase-operator/issues/30))


<a name="v2.17.0"></a>
## [v2.17.0] - 2023-08-17

<a name="v2.16.0"></a>
## [v2.16.0] - 2023-05-25

<a name="v2.15.0"></a>
## [v2.15.0] - 2023-03-24

<a name="v2.14.1"></a>
## [v2.14.1] - 2023-01-23

<a name="v2.14.0"></a>
## [v2.14.0] - 2022-12-06

<a name="v2.13.2"></a>
## [v2.13.2] - 2023-02-03

<a name="v2.13.1"></a>
## [v2.13.1] - 2022-10-28

<a name="v2.13.0"></a>
## [v2.13.0] - 2022-08-26

<a name="v2.12.0"></a>
## [v2.12.0] - 2022-05-25

<a name="v2.11.0"></a>
## [v2.11.0] - 2022-02-28

<a name="v2.10.1"></a>
## [v2.10.1] - 2022-02-21

<a name="v2.10.0"></a>
## [v2.10.0] - 2021-12-06

<a name="v2.9.0"></a>
## [v2.9.0] - 2021-12-03

<a name="v2.8.1"></a>
## [v2.8.1] - 2021-12-03

<a name="v2.8.0"></a>
## [v2.8.0] - 2021-12-03

<a name="v2.7.2"></a>
## [v2.7.2] - 2021-12-03

<a name="v2.7.1"></a>
## [v2.7.1] - 2021-12-03

<a name="v2.7.0"></a>
## [v2.7.0] - 2021-12-03
### Reverts

- [MDTUDDM-673] Save git status in the intermediate operations


<a name="v2.3.0-95"></a>
## [v2.3.0-95] - 2020-01-29

<a name="v2.3.0-94"></a>
## [v2.3.0-94] - 2020-01-24

<a name="v2.3.0-93"></a>
## [v2.3.0-93] - 2020-01-23

<a name="v2.3.0-92"></a>
## [v2.3.0-92] - 2020-01-22

<a name="v2.3.0-91"></a>
## [v2.3.0-91] - 2020-01-21

<a name="v2.3.0-89"></a>
## [v2.3.0-89] - 2020-01-20

<a name="v2.3.0-88"></a>
## [v2.3.0-88] - 2020-01-16

<a name="v2.3.0-87"></a>
## [v2.3.0-87] - 2020-01-16

<a name="v2.3.0-86"></a>
## [v2.3.0-86] - 2020-01-13

<a name="v2.3.0-85"></a>
## [v2.3.0-85] - 2020-01-13

<a name="v2.3.0-84"></a>
## [v2.3.0-84] - 2019-12-25

<a name="v2.2.0-90"></a>
## [v2.2.0-90] - 2020-01-21

<a name="v2.2.0-83"></a>
## [v2.2.0-83] - 2019-12-05

<a name="v2.2.0-82"></a>
## [v2.2.0-82] - 2019-12-04

<a name="v2.2.0-81"></a>
## [v2.2.0-81] - 2019-12-03

<a name="v2.2.0-80"></a>
## [v2.2.0-80] - 2019-11-27

<a name="v2.2.0-79"></a>
## [v2.2.0-79] - 2019-11-21

<a name="v2.2.0-78"></a>
## [v2.2.0-78] - 2019-11-15

<a name="v2.2.0-77"></a>
## [v2.2.0-77] - 2019-11-14

<a name="v2.2.0-76"></a>
## [v2.2.0-76] - 2019-11-13

<a name="v2.2.0-75"></a>
## [v2.2.0-75] - 2019-11-12

<a name="v2.2.0-74"></a>
## [v2.2.0-74] - 2019-11-05

<a name="v2.2.0-73"></a>
## [v2.2.0-73] - 2019-10-31

<a name="v2.2.0-72"></a>
## [v2.2.0-72] - 2019-10-28

<a name="v2.2.0-71"></a>
## [v2.2.0-71] - 2019-10-28

<a name="v2.2.0-70"></a>
## [v2.2.0-70] - 2019-10-25

<a name="v2.2.0-69"></a>
## [v2.2.0-69] - 2019-10-23

<a name="v2.2.0-68"></a>
## [v2.2.0-68] - 2019-10-15

<a name="v2.2.0-67"></a>
## [v2.2.0-67] - 2019-10-15

<a name="v2.2.0-66"></a>
## [v2.2.0-66] - 2019-10-15

<a name="v2.2.0-65"></a>
## [v2.2.0-65] - 2019-10-11

<a name="v2.2.0-64"></a>
## [v2.2.0-64] - 2019-10-04

<a name="v2.2.0-63"></a>
## [v2.2.0-63] - 2019-10-03

<a name="v2.2.0-62"></a>
## [v2.2.0-62] - 2019-10-02

<a name="v2.2.0-61"></a>
## [v2.2.0-61] - 2019-10-01

<a name="v2.2.0-60"></a>
## [v2.2.0-60] - 2019-10-01

<a name="v2.2.0-59"></a>
## [v2.2.0-59] - 2019-10-01

<a name="v2.2.0-58"></a>
## [v2.2.0-58] - 2019-09-30

<a name="v2.1.0-57"></a>
## [v2.1.0-57] - 2019-09-27

<a name="v2.1.0-56"></a>
## [v2.1.0-56] - 2019-09-27

<a name="v2.1.0-55"></a>
## [v2.1.0-55] - 2019-09-26

<a name="v2.1.0-54"></a>
## [v2.1.0-54] - 2019-09-26

<a name="v2.1.0-53"></a>
## [v2.1.0-53] - 2019-09-25

<a name="v2.1.0-52"></a>
## [v2.1.0-52] - 2019-09-19

<a name="v2.1.0-51"></a>
## [v2.1.0-51] - 2019-09-19

<a name="v2.1.0-50"></a>
## [v2.1.0-50] - 2019-09-17

<a name="v2.1.0-49"></a>
## [v2.1.0-49] - 2019-09-12

<a name="v2.1.0-48"></a>
## [v2.1.0-48] - 2019-09-10

<a name="v2.1.0-47"></a>
## v2.1.0-47 - 2019-09-10
### Reverts

- [EPMDEDP-3421] - Fix job+provisioning bug
- [EPMDEDP-868] Add functionality generates correct ssh URL for replication config for Gerrit


[Unreleased]: https://github.com/epam/edp-codebase-operator/compare/v2.28.0...HEAD
[v2.28.0]: https://github.com/epam/edp-codebase-operator/compare/v2.27.2...v2.28.0
[v2.27.2]: https://github.com/epam/edp-codebase-operator/compare/v2.27.1...v2.27.2
[v2.27.1]: https://github.com/epam/edp-codebase-operator/compare/v2.27.0...v2.27.1
[v2.27.0]: https://github.com/epam/edp-codebase-operator/compare/v2.26.0...v2.27.0
[v2.26.0]: https://github.com/epam/edp-codebase-operator/compare/v2.25.0...v2.26.0
[v2.25.0]: https://github.com/epam/edp-codebase-operator/compare/v2.24.1...v2.25.0
[v2.24.1]: https://github.com/epam/edp-codebase-operator/compare/v2.24.0...v2.24.1
[v2.24.0]: https://github.com/epam/edp-codebase-operator/compare/v2.23.0...v2.24.0
[v2.23.0]: https://github.com/epam/edp-codebase-operator/compare/v2.22.1...v2.23.0
[v2.22.1]: https://github.com/epam/edp-codebase-operator/compare/v2.22.0...v2.22.1
[v2.22.0]: https://github.com/epam/edp-codebase-operator/compare/v2.21.0...v2.22.0
[v2.21.0]: https://github.com/epam/edp-codebase-operator/compare/v2.20.0...v2.21.0
[v2.20.0]: https://github.com/epam/edp-codebase-operator/compare/v2.19.0...v2.20.0
[v2.19.0]: https://github.com/epam/edp-codebase-operator/compare/v2.18.1...v2.19.0
[v2.18.1]: https://github.com/epam/edp-codebase-operator/compare/v2.18.0...v2.18.1
[v2.18.0]: https://github.com/epam/edp-codebase-operator/compare/v2.17.1...v2.18.0
[v2.17.1]: https://github.com/epam/edp-codebase-operator/compare/v2.17.0...v2.17.1
[v2.17.0]: https://github.com/epam/edp-codebase-operator/compare/v2.16.0...v2.17.0
[v2.16.0]: https://github.com/epam/edp-codebase-operator/compare/v2.15.0...v2.16.0
[v2.15.0]: https://github.com/epam/edp-codebase-operator/compare/v2.14.1...v2.15.0
[v2.14.1]: https://github.com/epam/edp-codebase-operator/compare/v2.14.0...v2.14.1
[v2.14.0]: https://github.com/epam/edp-codebase-operator/compare/v2.13.2...v2.14.0
[v2.13.2]: https://github.com/epam/edp-codebase-operator/compare/v2.13.1...v2.13.2
[v2.13.1]: https://github.com/epam/edp-codebase-operator/compare/v2.13.0...v2.13.1
[v2.13.0]: https://github.com/epam/edp-codebase-operator/compare/v2.12.0...v2.13.0
[v2.12.0]: https://github.com/epam/edp-codebase-operator/compare/v2.11.0...v2.12.0
[v2.11.0]: https://github.com/epam/edp-codebase-operator/compare/v2.10.1...v2.11.0
[v2.10.1]: https://github.com/epam/edp-codebase-operator/compare/v2.10.0...v2.10.1
[v2.10.0]: https://github.com/epam/edp-codebase-operator/compare/v2.9.0...v2.10.0
[v2.9.0]: https://github.com/epam/edp-codebase-operator/compare/v2.8.1...v2.9.0
[v2.8.1]: https://github.com/epam/edp-codebase-operator/compare/v2.8.0...v2.8.1
[v2.8.0]: https://github.com/epam/edp-codebase-operator/compare/v2.7.2...v2.8.0
[v2.7.2]: https://github.com/epam/edp-codebase-operator/compare/v2.7.1...v2.7.2
[v2.7.1]: https://github.com/epam/edp-codebase-operator/compare/v2.7.0...v2.7.1
[v2.7.0]: https://github.com/epam/edp-codebase-operator/compare/v2.3.0-95...v2.7.0
[v2.3.0-95]: https://github.com/epam/edp-codebase-operator/compare/v2.3.0-94...v2.3.0-95
[v2.3.0-94]: https://github.com/epam/edp-codebase-operator/compare/v2.3.0-93...v2.3.0-94
[v2.3.0-93]: https://github.com/epam/edp-codebase-operator/compare/v2.3.0-92...v2.3.0-93
[v2.3.0-92]: https://github.com/epam/edp-codebase-operator/compare/v2.3.0-91...v2.3.0-92
[v2.3.0-91]: https://github.com/epam/edp-codebase-operator/compare/v2.3.0-89...v2.3.0-91
[v2.3.0-89]: https://github.com/epam/edp-codebase-operator/compare/v2.3.0-88...v2.3.0-89
[v2.3.0-88]: https://github.com/epam/edp-codebase-operator/compare/v2.3.0-87...v2.3.0-88
[v2.3.0-87]: https://github.com/epam/edp-codebase-operator/compare/v2.3.0-86...v2.3.0-87
[v2.3.0-86]: https://github.com/epam/edp-codebase-operator/compare/v2.3.0-85...v2.3.0-86
[v2.3.0-85]: https://github.com/epam/edp-codebase-operator/compare/v2.3.0-84...v2.3.0-85
[v2.3.0-84]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-90...v2.3.0-84
[v2.2.0-90]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-83...v2.2.0-90
[v2.2.0-83]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-82...v2.2.0-83
[v2.2.0-82]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-81...v2.2.0-82
[v2.2.0-81]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-80...v2.2.0-81
[v2.2.0-80]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-79...v2.2.0-80
[v2.2.0-79]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-78...v2.2.0-79
[v2.2.0-78]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-77...v2.2.0-78
[v2.2.0-77]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-76...v2.2.0-77
[v2.2.0-76]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-75...v2.2.0-76
[v2.2.0-75]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-74...v2.2.0-75
[v2.2.0-74]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-73...v2.2.0-74
[v2.2.0-73]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-72...v2.2.0-73
[v2.2.0-72]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-71...v2.2.0-72
[v2.2.0-71]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-70...v2.2.0-71
[v2.2.0-70]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-69...v2.2.0-70
[v2.2.0-69]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-68...v2.2.0-69
[v2.2.0-68]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-67...v2.2.0-68
[v2.2.0-67]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-66...v2.2.0-67
[v2.2.0-66]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-65...v2.2.0-66
[v2.2.0-65]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-64...v2.2.0-65
[v2.2.0-64]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-63...v2.2.0-64
[v2.2.0-63]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-62...v2.2.0-63
[v2.2.0-62]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-61...v2.2.0-62
[v2.2.0-61]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-60...v2.2.0-61
[v2.2.0-60]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-59...v2.2.0-60
[v2.2.0-59]: https://github.com/epam/edp-codebase-operator/compare/v2.2.0-58...v2.2.0-59
[v2.2.0-58]: https://github.com/epam/edp-codebase-operator/compare/v2.1.0-57...v2.2.0-58
[v2.1.0-57]: https://github.com/epam/edp-codebase-operator/compare/v2.1.0-56...v2.1.0-57
[v2.1.0-56]: https://github.com/epam/edp-codebase-operator/compare/v2.1.0-55...v2.1.0-56
[v2.1.0-55]: https://github.com/epam/edp-codebase-operator/compare/v2.1.0-54...v2.1.0-55
[v2.1.0-54]: https://github.com/epam/edp-codebase-operator/compare/v2.1.0-53...v2.1.0-54
[v2.1.0-53]: https://github.com/epam/edp-codebase-operator/compare/v2.1.0-52...v2.1.0-53
[v2.1.0-52]: https://github.com/epam/edp-codebase-operator/compare/v2.1.0-51...v2.1.0-52
[v2.1.0-51]: https://github.com/epam/edp-codebase-operator/compare/v2.1.0-50...v2.1.0-51
[v2.1.0-50]: https://github.com/epam/edp-codebase-operator/compare/v2.1.0-49...v2.1.0-50
[v2.1.0-49]: https://github.com/epam/edp-codebase-operator/compare/v2.1.0-48...v2.1.0-49
[v2.1.0-48]: https://github.com/epam/edp-codebase-operator/compare/v2.1.0-47...v2.1.0-48
