<a name="unreleased"></a>
## [Unreleased]


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

[Unreleased]: https://github.com/epam/edp-codebase-operator/compare/v2.22.0...HEAD
[v2.22.0]: https://github.com/epam/edp-codebase-operator/compare/v2.21.0...v2.22.0
[v2.21.0]: https://github.com/epam/edp-codebase-operator/compare/v2.20.0...v2.21.0
[v2.20.0]: https://github.com/epam/edp-codebase-operator/compare/v2.19.0...v2.20.0
[v2.19.0]: https://github.com/epam/edp-codebase-operator/compare/v2.18.1...v2.19.0
[v2.18.1]: https://github.com/epam/edp-codebase-operator/compare/v2.18.0...v2.18.1
[v2.18.0]: https://github.com/epam/edp-codebase-operator/compare/v2.17.1...v2.18.0
[v2.17.1]: https://github.com/epam/edp-codebase-operator/compare/v2.17.0...v2.17.1
[v2.17.0]: https://github.com/epam/edp-codebase-operator/compare/v2.16.0...v2.17.0
