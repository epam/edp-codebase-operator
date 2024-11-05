<a name="unreleased"></a>
## [Unreleased]

### Bug Fixes

- Processing default branch for bitbucket ([#155](https://github.com/epam/edp-codebase-operator/issues/155))
- Ignore extra labels in CodebaseImageStream for auto-deploy ([#153](https://github.com/epam/edp-codebase-operator/issues/153))

### Routine

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

[Unreleased]: https://github.com/epam/edp-codebase-operator/compare/v2.24.1...HEAD
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
