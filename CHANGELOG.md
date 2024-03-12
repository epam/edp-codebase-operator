<a name="unreleased"></a>
## [Unreleased]


<a name="v2.22.1"></a>
## v2.22.1 - 2024-03-12
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
- Send telemetry metrics to the EDP backend ([#31](https://github.com/epam/edp-codebase-operator/issues/31))
- Add connection status to registry integration secret ([#29](https://github.com/epam/edp-codebase-operator/issues/29))
- Add connection error to integration secret ([#29](https://github.com/epam/edp-codebase-operator/issues/29))
- Add connection status to the integration secret annotation ([#29](https://github.com/epam/edp-codebase-operator/issues/29))
- Automate rekor uuid in release tag ([#27](https://github.com/epam/edp-codebase-operator/issues/27))
- Clean private repository secret ([#25](https://github.com/epam/edp-codebase-operator/issues/25))
- Migrate registry url from EDP component to edp-config configmap ([#20](https://github.com/epam/edp-codebase-operator/issues/20))
- Allow CodebaseImageStream to contain docker hub image ([#17](https://github.com/epam/edp-codebase-operator/issues/17))
- Add valid error in status field of GitServer ([#15](https://github.com/epam/edp-codebase-operator/issues/15))

### Bug Fixes

- Do not fail reconciliation when CDStageDeploy exists ([#47](https://github.com/epam/edp-codebase-operator/issues/47))
- Auto-deploy failed into a remote cluster ([#49](https://github.com/epam/edp-codebase-operator/issues/49))
- Connection status for docker repository is failed ([#48](https://github.com/epam/edp-codebase-operator/issues/48))
- Empty auto-deploy flow logs attributes ([#44](https://github.com/epam/edp-codebase-operator/issues/44))
- Set success status for JiraServer ([#42](https://github.com/epam/edp-codebase-operator/issues/42))
- Empty project failed status ([#43](https://github.com/epam/edp-codebase-operator/issues/43))
- Fix deployment template for telemetry ([#31](https://github.com/epam/edp-codebase-operator/issues/31))
- Use minutes for telemetry delay ([#31](https://github.com/epam/edp-codebase-operator/issues/31))
- Integration status for dockerhub registry failed ([#29](https://github.com/epam/edp-codebase-operator/issues/29))
- Use EDP approach for label naming ([#29](https://github.com/epam/edp-codebase-operator/issues/29))
- Reset version for edp versioning in case the new release branch created is not working ([#24](https://github.com/epam/edp-codebase-operator/issues/24))
- Auto deploy failed with GitOps approach ([#23](https://github.com/epam/edp-codebase-operator/issues/23))
- Build pipeline failed for go operator sdk on sonar step ([#12](https://github.com/epam/edp-codebase-operator/issues/12))

### Code Refactoring

- Migrate EDPComponent CRD to edp-codebase-operator ([#40](https://github.com/epam/edp-codebase-operator/issues/40))
- Replace edpName with helm Release Namespace value ([#10](https://github.com/epam/edp-codebase-operator/issues/10))

### Routine

- Bump alpine version ([#51](https://github.com/epam/edp-codebase-operator/issues/51))
- Remove Jira EDPComponent CR ([#168](https://github.com/epam/edp-codebase-operator/issues/168))
- Add jira QuickLink ([#168](https://github.com/epam/edp-codebase-operator/issues/168))
- Bump github.com/cloudflare/circl from 1.3.3 to 1.3.7 ([#38](https://github.com/epam/edp-codebase-operator/issues/38))
- Bump github.com/go-git/go-git/v5 to 5.11.0 ([#34](https://github.com/epam/edp-codebase-operator/issues/34))
- Bump golang.org/x/crypto from 0.14.0 to 0.17.0 ([#33](https://github.com/epam/edp-codebase-operator/issues/33))
- Update alpine image to the latest stable ([#35](https://github.com/epam/edp-codebase-operator/issues/35))
- Update current development version ([#32](https://github.com/epam/edp-codebase-operator/issues/32))
- Update current development version ([#26](https://github.com/epam/edp-codebase-operator/issues/26))
- Add commit rule for pull request template ([#21](https://github.com/epam/edp-codebase-operator/issues/21))
- Bump golang.org/x/net from 0.8.0 to 0.17.0 ([#22](https://github.com/epam/edp-codebase-operator/issues/22))
- Upgrade pull request template ([#21](https://github.com/epam/edp-codebase-operator/issues/21))
- Remove jenkins admin-console perf operator logic ([#18](https://github.com/epam/edp-codebase-operator/issues/18))
- Update current development version ([#16](https://github.com/epam/edp-codebase-operator/issues/16))
- Upgrade Go to 1.20 ([#14](https://github.com/epam/edp-codebase-operator/issues/14))
- Update current development version ([#11](https://github.com/epam/edp-codebase-operator/issues/11))
- Align secret name for new secret approach ([#65](https://github.com/epam/edp-codebase-operator/issues/65))
- Update current development version ([#9](https://github.com/epam/edp-codebase-operator/issues/9))

### Documentation

- Add more details regarding telemetry option ([#31](https://github.com/epam/edp-codebase-operator/issues/31))

### Reverts

- [MDTUDDM-673] Save git status in the intermediate operations
- [EPMDEDP-3421] - Fix job+provisioning bug
- [EPMDEDP-868] Add functionality generates correct ssh URL for replication config for Gerrit

### BREAKING CHANGE:


removed required fields from the GitServer status. CRD should be updated.


[Unreleased]: https://github.com/epam/edp-codebase-operator/compare/v2.22.1...HEAD
