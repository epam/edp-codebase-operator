<a name="unreleased"></a>
## [Unreleased]

### Features

- Switch to edp-common for BuildInfo [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Provide operator's build information [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Proper exclusion for golang tests in sonar [EPMDEDP-7535](https://jiraeu.epam.com/browse/EPMDEDP-7535)
- Exclude groovy files from sonar scan [EPMDEDP-7743](https://jiraeu.epam.com/browse/EPMDEDP-7743)
- Improve ingress creation [EPMDEDP-7483](https://jiraeu.epam.com/browse/EPMDEDP-7483)
- Improve default ingress creation [EPMDEDP-7483](https://jiraeu.epam.com/browse/EPMDEDP-7483)
- implement dynamic params for codebase branch trigger release [EPMDEDP-7456](https://jiraeu.epam.com/browse/EPMDEDP-7456)

### Bug Fixes

- Address vet, lint issues [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Show username value in logs [EPMDEDP-7880](https://jiraeu.epam.com/browse/EPMDEDP-7880)
- Address issue with git remote replacement [EPMDEDP-7880](https://jiraeu.epam.com/browse/EPMDEDP-7880)
- Do not fail codebase provisioining for public repos [EPMDEDP-7880](https://jiraeu.epam.com/browse/EPMDEDP-7880)
- update failure reconciliation timeout formula [EPMDEDP-7458](https://jiraeu.epam.com/browse/EPMDEDP-7458)
- Fix issue with finalizer removal for codebasebranch [EPMDEDP-7544](https://jiraeu.epam.com/browse/EPMDEDP-7544)
- fix default release name param for trigger release function [EPMDEDP-7456](https://jiraeu.epam.com/browse/EPMDEDP-7456)

### Code Refactoring

- Address golangci-lint issues [EPMDEDP-7945](https://jiraeu.epam.com/browse/EPMDEDP-7945)
- Remove Jenkins pipelines logic [EPMDEDP-7136](https://jiraeu.epam.com/browse/EPMDEDP-7136)
- Add tests for Controllers [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Codebase and Codebasebranch ctrl [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Disable jira integration by default [EPMDEDP-7812](https://jiraeu.epam.com/browse/EPMDEDP-7812)
- Address staticcheck issues [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Update VCS model [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add unit tests for project creation in Gerrit [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Fix issue with template rendering [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Increase code coverage for gerrit pkg [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Remove duplicate code [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add validation and checks in put_cd_stage_deploy [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Decrease Jenkins job status poll timeout [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Remove duplicate code [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Expand codebase-operator role [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Add namespace field in roleRef in OKD RB, align CRB name [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Push with default branch for GitLabCI flow [EPMDEDP-7336](https://jiraeu.epam.com/browse/EPMDEDP-7336)
- Replace cluster-wide role/rolebinding to namespaced [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)

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

- Update andygrunwald/go-jira to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Update cd-pipeline, jenkins, component to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- update Go version at codecov.yaml [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)
- Use custom go build step for operator [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Update go to version 1.17 [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Update docker image [EPMDEDP-7895](https://jiraeu.epam.com/browse/EPMDEDP-7895)
- Run CodeQL only on push to master branch [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)
- Add codecov integration [EPMDEDP-7505](https://jiraeu.epam.com/browse/EPMDEDP-7505)

### Documentation

- Update the links on GitHub [EPMDEDP-7781](https://jiraeu.epam.com/browse/EPMDEDP-7781)


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


[Unreleased]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.9.0...HEAD
[v2.9.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.8.1...v2.9.0
[v2.8.1]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.8.0...v2.8.1
[v2.8.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.7.2...v2.8.0
[v2.7.2]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.7.1...v2.7.2
[v2.7.1]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.7.0...v2.7.1
[v2.7.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.3.0-95...v2.7.0
