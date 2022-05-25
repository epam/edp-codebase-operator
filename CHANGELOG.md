<a name="unreleased"></a>
## [Unreleased]


<a name="v2.12.0"></a>
## [v2.12.0] - 2022-04-28
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


[Unreleased]: https://github.com/epam/edp-codebase-operator/compare/v2.12.0...HEAD
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
