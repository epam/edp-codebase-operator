<a name="unreleased"></a>
## [Unreleased]

### Routine

- Upgrade alpine image version to 3.16.4 [EPMDEDP-11764](https://jiraeu.epam.com/browse/EPMDEDP-11764)


<a name="v2.15.0"></a>
## [v2.15.0] - 2023-03-24
### Features

- Import strategy supports .git [EPMDEDP-10886](https://jiraeu.epam.com/browse/EPMDEDP-10886)
- Validating Codebase gitUrlpath for uniqueness [EPMDEDP-10945](https://jiraeu.epam.com/browse/EPMDEDP-10945)
- Updated operator sdk version to 1.25.3 [EPMDEDP-11022](https://jiraeu.epam.com/browse/EPMDEDP-11022)
- Add annotation for disabling reconciliation [EPMDEDP-11200](https://jiraeu.epam.com/browse/EPMDEDP-11200)
- Update EDP components [EPMDEDP-11206](https://jiraeu.epam.com/browse/EPMDEDP-11206)
- Duplicated api to their old folder [EPMDEDP-11206](https://jiraeu.epam.com/browse/EPMDEDP-11206)
- Set codebase.gitUrlPath equal codebase.name when empty [EPMDEDP-11300](https://jiraeu.epam.com/browse/EPMDEDP-11300)
- Add helm language support [EPMDEDP-11359](https://jiraeu.epam.com/browse/EPMDEDP-11359)
- Add suffix .git to the end of git url path if it does not exist with clone strategy [EPMDEDP-11406](https://jiraeu.epam.com/browse/EPMDEDP-11406)
- Use Route instead of Ingress on OKD [EPMDEDP-11588](https://jiraeu.epam.com/browse/EPMDEDP-11588)
- Add logs for resources with turned off reconciliation [EPMDEDP-11597](https://jiraeu.epam.com/browse/EPMDEDP-11597)
- Removed VERSION file creation for Go codebases [EPMDEDP-11598](https://jiraeu.epam.com/browse/EPMDEDP-11598)

### Bug Fixes

- Jira issue metadata wrong ticket blocker [EPMDEDP-10454](https://jiraeu.epam.com/browse/EPMDEDP-10454)
- Get jira project info error handler [EPMDEDP-10454](https://jiraeu.epam.com/browse/EPMDEDP-10454)
- Check if gitUrlPath is unique with the .git suffix [EPMDEDP-10945](https://jiraeu.epam.com/browse/EPMDEDP-10945)
- Align GitHub workflows to the new project structure [EPMDEDP-11022](https://jiraeu.epam.com/browse/EPMDEDP-11022)
- Check if Codebasebranch commit hash is valid before triggering the Jenkins job [EPMDEDP-11189](https://jiraeu.epam.com/browse/EPMDEDP-11189)
- Set unsuccessful status for Codebasebranch if commit hash is invalid [EPMDEDP-11189](https://jiraeu.epam.com/browse/EPMDEDP-11189)
- Set Codebasebranch failureCount for tekton flow [EPMDEDP-11189](https://jiraeu.epam.com/browse/EPMDEDP-11189)
- Set Codebasebranch failureCount for tekton flow [EPMDEDP-11189](https://jiraeu.epam.com/browse/EPMDEDP-11189)
- Operator updates the codebase branch status all the time [EPMDEDP-11357](https://jiraeu.epam.com/browse/EPMDEDP-11357)
- Add suffix .git to the end of git repository url if it does not exist with clone strategy [EPMDEDP-11406](https://jiraeu.epam.com/browse/EPMDEDP-11406)
- Removed error type fields from API [EPMDEDP-11463](https://jiraeu.epam.com/browse/EPMDEDP-11463)
- Reconcile resources after updating the pause annotation [EPMDEDP-11631](https://jiraeu.epam.com/browse/EPMDEDP-11631)
- Do not make empty commits [EPMDEDP-7985](https://jiraeu.epam.com/browse/EPMDEDP-7985)

### Code Refactoring

- Move documentation to docs [EPMDEDP-11195](https://jiraeu.epam.com/browse/EPMDEDP-11195)

### Testing

- Improved test coverage [EPMDEDP-11065](https://jiraeu.epam.com/browse/EPMDEDP-11065)

### Routine

- Update current development version [EPMDEDP-10610](https://jiraeu.epam.com/browse/EPMDEDP-10610)
- Add support for csharp language [EPMDEDP-11173](https://jiraeu.epam.com/browse/EPMDEDP-11173)
- Removed redundant pkg/errors direct dependency [EPMDEDP-11201](https://jiraeu.epam.com/browse/EPMDEDP-11201)
- Updated cd-pipeline-operator dependency [EPMDEDP-11275](https://jiraeu.epam.com/browse/EPMDEDP-11275)
- Update git-chglog for codebase-operator [EPMDEDP-11518](https://jiraeu.epam.com/browse/EPMDEDP-11518)
- Bump golang.org/x/net from 0.5.0 to 0.8.0 [EPMDEDP-11578](https://jiraeu.epam.com/browse/EPMDEDP-11578)


<a name="v2.14.1"></a>
## [v2.14.1] - 2023-01-23
### Bug Fixes

- Set unsuccessful status for Codebasebranch if commit hash is invalid [EPMDEDP-11189](https://jiraeu.epam.com/browse/EPMDEDP-11189)

### Routine

- Update git package version to 2.36.4-r0 [EPMDEDP-11260](https://jiraeu.epam.com/browse/EPMDEDP-11260)


<a name="v2.14.0"></a>
## [v2.14.0] - 2022-12-06
### Features

- Add support for Tekton Tool [EPMDEDP-10641](https://jiraeu.epam.com/browse/EPMDEDP-10641)
- Implement webhook GitLab integration as a part of codebase onboarding [EPMDEDP-10708](https://jiraeu.epam.com/browse/EPMDEDP-10708)
- Implement default branch provisioning by codebase controller [EPMDEDP-10733](https://jiraeu.epam.com/browse/EPMDEDP-10733)
- Implement GitHub Webhook integration for codebase [EPMDEDP-10743](https://jiraeu.epam.com/browse/EPMDEDP-10743)
- GitServer - make gitUser optional with the default value git [EPMDEDP-11026](https://jiraeu.epam.com/browse/EPMDEDP-11026)

### Bug Fixes

- Warnings from go-critic [EPMDEDP-10542](https://jiraeu.epam.com/browse/EPMDEDP-10542)
- Add golangci-lint config file [EPMDEDP-10542](https://jiraeu.epam.com/browse/EPMDEDP-10542)
- Golangci-lint errors [EPMDEDP-10542](https://jiraeu.epam.com/browse/EPMDEDP-10542)
- Remove slash from CodebaseSpec.GitUrlPath for the webhook creation [EPMDEDP-10708](https://jiraeu.epam.com/browse/EPMDEDP-10708)
- GitHub webhook events names [EPMDEDP-10743](https://jiraeu.epam.com/browse/EPMDEDP-10743)
- Fix permissions for configmap in leader election [EPMDEDP-10802](https://jiraeu.epam.com/browse/EPMDEDP-10802)
- Use an already created webhook [EPMDEDP-10844](https://jiraeu.epam.com/browse/EPMDEDP-10844)
- CodebaseBranch with empty version may produce panic [EPMDEDP-10883](https://jiraeu.epam.com/browse/EPMDEDP-10883)
- Concurrent write ssh key to the same file [EPMDEDP-10892](https://jiraeu.epam.com/browse/EPMDEDP-10892)
- Do not ignore git command errors [EPMDEDP-10892](https://jiraeu.epam.com/browse/EPMDEDP-10892)
- Set GitServer unavailable status if the ssh key is invalid [EPMDEDP-10914](https://jiraeu.epam.com/browse/EPMDEDP-10914)
- Add empty line to git server private key [EPMDEDP-10982](https://jiraeu.epam.com/browse/EPMDEDP-10982)

### Code Refactoring

- Use single chain [EPMDEDP-10498](https://jiraeu.epam.com/browse/EPMDEDP-10498)
- Resolve `revive` lint warnings [EPMDEDP-10542](https://jiraeu.epam.com/browse/EPMDEDP-10542)
- Partially resolve golangci-lint warnings [EPMDEDP-10542](https://jiraeu.epam.com/browse/EPMDEDP-10542)
- Enable two more golangci-lint rules [EPMDEDP-10542](https://jiraeu.epam.com/browse/EPMDEDP-10542)
- Set remote git HEAD to default branch in Gerrit [EPMDEDP-10658](https://jiraeu.epam.com/browse/EPMDEDP-10658)
- Create default codebasebranch for other flows [EPMDEDP-10733](https://jiraeu.epam.com/browse/EPMDEDP-10733)
- Remove SQL DB repository [EPMDEDP-10751](https://jiraeu.epam.com/browse/EPMDEDP-10751)

### Routine

- Update current development version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Upgrade git package to version 2.36.3-r0 [EPMDEDP-10743](https://jiraeu.epam.com/browse/EPMDEDP-10743)
- Update current development version [EPMDEDP-10805](https://jiraeu.epam.com/browse/EPMDEDP-10805)
- Update Changelog [EPMDEDP-11010](https://jiraeu.epam.com/browse/EPMDEDP-11010)
- Remove deprecated Dotnet 2.1 support [EPMDEDP-11024](https://jiraeu.epam.com/browse/EPMDEDP-11024)

### Documentation

- Update Readme [EPMDEDP-11008](https://jiraeu.epam.com/browse/EPMDEDP-11008)
- Update Readme [EPMDEDP-11008](https://jiraeu.epam.com/browse/EPMDEDP-11008)


<a name="v2.13.2"></a>
## [v2.13.2] - 2023-02-03
### Bug Fixes

- Get jira project info error handler [EPMDEDP-10454](https://jiraeu.epam.com/browse/EPMDEDP-10454)
- Jira issue metadata wrong ticket blocker [EPMDEDP-10454](https://jiraeu.epam.com/browse/EPMDEDP-10454)


<a name="v2.13.1"></a>
## [v2.13.1] - 2022-10-28
### Code Refactoring

- Set remote git HEAD to default branch in Gerrit [EPMDEDP-10810](https://jiraeu.epam.com/browse/EPMDEDP-10810)

### Routine

- Upgrade git package to version 2.36.3-r0 [EPMDEDP-10743](https://jiraeu.epam.com/browse/EPMDEDP-10743)


<a name="v2.13.0"></a>
## [v2.13.0] - 2022-08-26
### Features

- Switch to use V1 apis of EDP components [EPMDEDP-10086](https://jiraeu.epam.com/browse/EPMDEDP-10086)
- Download required tools for Makefile targets [EPMDEDP-10105](https://jiraeu.epam.com/browse/EPMDEDP-10105)
- Add label for a type of Codebase [EPMDEDP-10122](https://jiraeu.epam.com/browse/EPMDEDP-10122)
- Add Kubernetes and GitOps languages [EPMDEDP-8257](https://jiraeu.epam.com/browse/EPMDEDP-8257)
- Switch all CRDs to V1 schema [EPMDEDP-9215](https://jiraeu.epam.com/browse/EPMDEDP-9215)

### Bug Fixes

- Add a short names for CRs, plus fix two plurals [EPMDEDP-10043](https://jiraeu.epam.com/browse/EPMDEDP-10043)
- Make sure CRD which requires update of a status field have enabled status as subresource [EPMDEDP-10043](https://jiraeu.epam.com/browse/EPMDEDP-10043)
- Make sure we init "build_number" field for "edp" versioning [EPMDEDP-10122](https://jiraeu.epam.com/browse/EPMDEDP-10122)
- Allow to update known_hosts [EPMDEDP-10473](https://jiraeu.epam.com/browse/EPMDEDP-10473)

### Code Refactoring

- Refactor user creation procedure [EPMDEDP-10131](https://jiraeu.epam.com/browse/EPMDEDP-10131)
- Replace usermod with sed approach [EPMDEDP-10131](https://jiraeu.epam.com/browse/EPMDEDP-10131)
- Refactor user creation for OpenShift [EPMDEDP-10131](https://jiraeu.epam.com/browse/EPMDEDP-10131)
- Remove createCodeReviewPipeline in Codebase [EPMDEDP-10156](https://jiraeu.epam.com/browse/EPMDEDP-10156)
- Use repository and tag for image reference in chart [EPMDEDP-10389](https://jiraeu.epam.com/browse/EPMDEDP-10389)

### Routine

- Upgrade go version to 1.18 [EPMDEDP-10110](https://jiraeu.epam.com/browse/EPMDEDP-10110)
- Fix Jira Ticket pattern for changelog generator [EPMDEDP-10159](https://jiraeu.epam.com/browse/EPMDEDP-10159)
- Update alpine base image to 3.16.2 version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update alpine base image version [EPMDEDP-10280](https://jiraeu.epam.com/browse/EPMDEDP-10280)
- Change 'go get' to 'go install' for git-chglog [EPMDEDP-10337](https://jiraeu.epam.com/browse/EPMDEDP-10337)
- Remove VERSION file [EPMDEDP-10387](https://jiraeu.epam.com/browse/EPMDEDP-10387)
- Remove Kubernetes and GitOps languages [EPMDEDP-10397](https://jiraeu.epam.com/browse/EPMDEDP-10397)
- Add gcflags for go build artifact [EPMDEDP-10411](https://jiraeu.epam.com/browse/EPMDEDP-10411)
- Update current development version [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Update chart annotation [EPMDEDP-9515](https://jiraeu.epam.com/browse/EPMDEDP-9515)

### Documentation

- Align README.md [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Align README.md [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)


<a name="v2.12.0"></a>
## [v2.12.0] - 2022-05-25
### Features

- Add tags-field for the list of codebases' tags. [EPMDEDP-8313](https://jiraeu.epam.com/browse/EPMDEDP-8313)
- Add replace default branch option to spec [EPMDEDP-8804](https://jiraeu.epam.com/browse/EPMDEDP-8804)
- Postpone reconciliation for import strategy [EPMDEDP-8956](https://jiraeu.epam.com/browse/EPMDEDP-8956)
- Repeat reconciliation if git server is unavailable [EPMDEDP-8996](https://jiraeu.epam.com/browse/EPMDEDP-8996)
- Disable of putting deploy configs by flag [EPMDEDP-9003](https://jiraeu.epam.com/browse/EPMDEDP-9003)

### Bug Fixes

- Clone of large codebases [EPMDEDP-8505](https://jiraeu.epam.com/browse/EPMDEDP-8505)
- Enable RSA algorithm [EPMDEDP-8847](https://jiraeu.epam.com/browse/EPMDEDP-8847)

### Routine

- Update base docker image to alpine 3.15.4 [EPMDEDP-8853](https://jiraeu.epam.com/browse/EPMDEDP-8853)
- Update changelog [EPMDEDP-9185](https://jiraeu.epam.com/browse/EPMDEDP-9185)


<a name="v2.11.0"></a>
## [v2.11.0] - 2022-02-28
### Features

- Update Makefile changelog target [EPMDEDP-8218](https://jiraeu.epam.com/browse/EPMDEDP-8218)
- Parametrize ENV variables for codebase-operator deployment [EPMDEDP-8268](https://jiraeu.epam.com/browse/EPMDEDP-8268)
- Add container language and docker framework [EPMDEDP-8341](https://jiraeu.epam.com/browse/EPMDEDP-8341)
- Generate CRDs and helm docs automatically [EPMDEDP-8385](https://jiraeu.epam.com/browse/EPMDEDP-8385)
- add label for the codebaseBranch custom resource on reconciliation. [EPMDEDP-8414](https://jiraeu.epam.com/browse/EPMDEDP-8414)

### Bug Fixes

- Fix changelog breaking change section [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Do not mount db secrets when DB_ENABLED is false [EPMDEDP-8203](https://jiraeu.epam.com/browse/EPMDEDP-8203)
- Change ca-certificates in dockerfile [EPMDEDP-8238](https://jiraeu.epam.com/browse/EPMDEDP-8238)
- Fix typo in env parameter value [EPMDEDP-8268](https://jiraeu.epam.com/browse/EPMDEDP-8268)
- Fix CRD doc generation [EPMDEDP-8385](https://jiraeu.epam.com/browse/EPMDEDP-8385)
- Fix changelog generation in GH Release Action [EPMDEDP-8386](https://jiraeu.epam.com/browse/EPMDEDP-8386)
- Fix import strategy for arbitrary branch [EPMDEDP-8423](https://jiraeu.epam.com/browse/EPMDEDP-8423)
- Correct image version [EPMDEDP-8471](https://jiraeu.epam.com/browse/EPMDEDP-8471)

### Routine

- Add automatic GitHub Release Action [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Bump main branch version [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Update release CI pipelines [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Fix git-chglog installation (GH Actions) [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Switch changelog from JiraIssueID to Scope [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Fix release pipeline in GH Actions [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Populate chart with Artifacthub annotations [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)


<a name="v2.10.1"></a>
## [v2.10.1] - 2022-02-21
### Features

- Generate CRDs and helm docs automatically [EPMDEDP-8385](https://jiraeu.epam.com/browse/EPMDEDP-8385)

### Bug Fixes

- Change ca-certificates in dockerfile [EPMDEDP-8238](https://jiraeu.epam.com/browse/EPMDEDP-8238)
- Fix changelog generation in GH Release Action [EPMDEDP-8386](https://jiraeu.epam.com/browse/EPMDEDP-8386)
- Fix import strategy for arbitrary branch [EPMDEDP-8423](https://jiraeu.epam.com/browse/EPMDEDP-8423)

### Routine

- Switch changelog from JiraIssueID to Scope [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Fix git-chglog installation (GH Actions) [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Fix release pipeline in GH Actions [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Add automatic GitHub Release Action [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)


<a name="v2.10.0"></a>
## [v2.10.0] - 2021-12-06
### Features

- implement dynamic params for codebase branch trigger release [EPMDEDP-7456](https://jiraeu.epam.com/browse/EPMDEDP-7456)
- Improve ingress creation [EPMDEDP-7483](https://jiraeu.epam.com/browse/EPMDEDP-7483)
- Improve default ingress creation [EPMDEDP-7483](https://jiraeu.epam.com/browse/EPMDEDP-7483)
- Proper exclusion for golang tests in sonar [EPMDEDP-7535](https://jiraeu.epam.com/browse/EPMDEDP-7535)
- Exclude groovy files from sonar scan [EPMDEDP-7743](https://jiraeu.epam.com/browse/EPMDEDP-7743)
- Provide operator's build information [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Switch to edp-common for BuildInfo [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)

### Bug Fixes

- fix default release name param for trigger release function [EPMDEDP-7456](https://jiraeu.epam.com/browse/EPMDEDP-7456)
- update failure reconciliation timeout formula [EPMDEDP-7458](https://jiraeu.epam.com/browse/EPMDEDP-7458)
- Fix issue with finalizer removal for codebasebranch [EPMDEDP-7544](https://jiraeu.epam.com/browse/EPMDEDP-7544)
- Fix changelog links [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Address vet, lint issues [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Show username value in logs [EPMDEDP-7880](https://jiraeu.epam.com/browse/EPMDEDP-7880)
- Address issue with git remote replacement [EPMDEDP-7880](https://jiraeu.epam.com/browse/EPMDEDP-7880)
- Do not fail codebase provisioining for public repos [EPMDEDP-7880](https://jiraeu.epam.com/browse/EPMDEDP-7880)

### Code Refactoring

- Remove Jenkins pipelines logic [EPMDEDP-7136](https://jiraeu.epam.com/browse/EPMDEDP-7136)
- Replace cluster-wide role/rolebinding to namespaced [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Add namespace field in roleRef in OKD RB, align CRB name [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Expand codebase-operator role [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Push with default branch for GitLabCI flow [EPMDEDP-7336](https://jiraeu.epam.com/browse/EPMDEDP-7336)
- Increase code coverage for gerrit pkg [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Update VCS model [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add unit tests for project creation in Gerrit [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Fix issue with template rendering [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Address staticcheck issues [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Remove duplicate code [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add validation and checks in put_cd_stage_deploy [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Decrease Jenkins job status poll timeout [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Remove duplicate code [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Codebase and Codebasebranch ctrl [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add tests for Controllers [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Disable jira integration by default [EPMDEDP-7812](https://jiraeu.epam.com/browse/EPMDEDP-7812)
- Address golangci-lint issues [EPMDEDP-7945](https://jiraeu.epam.com/browse/EPMDEDP-7945)

### Testing

- Do not run SetupWithManager unit tests [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add test for cdstagedeploy controller [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add test for jira adapter part [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Exclude helm-chart templates from sonar scan [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add tests for cluster package [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add tests for jenkins package [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Increase coverage for jenkins client [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add test for jenkins client [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)

### Routine

- Run CodeQL only on push to master branch [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add codecov integration [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add changelog generator [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- update Go version at codecov.yaml [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)
- Update docker image [EPMDEDP-7895](https://jiraeu.epam.com/browse/EPMDEDP-7895)
- Update andygrunwald/go-jira to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Update cd-pipeline, jenkins, component to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Use custom go build step for operator [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Update go to version 1.17 [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)

### Documentation

- Update the links on GitHub [EPMDEDP-7781](https://jiraeu.epam.com/browse/EPMDEDP-7781)

### BREAKING CHANGE:


Now, codebase operator doesn't provision
Jenkins pipelines for code-review, build, create-release flow.


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


[Unreleased]: https://github.com/epam/edp-codebase-operator/compare/v2.15.0...HEAD
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
