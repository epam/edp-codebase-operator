<a name="unreleased"></a>
## [Unreleased]


<a name="v2.10.0"></a>
## [v2.10.0] - 2021-12-06
### Features

- Switch to edp-common for BuildInfo
- Provide operator's build information
- Proper exclusion for golang tests in sonar
- Exclude groovy files from sonar scan
- Improve ingress creation
- Improve default ingress creation
- implement dynamic params for codebase branch trigger release

### Bug Fixes

- Fix changelog links
- Address vet, lint issues
- Show username value in logs
- Address issue with git remote replacement
- Do not fail codebase provisioining for public repos
- update failure reconciliation timeout formula
- Fix issue with finalizer removal for codebasebranch
- fix default release name param for trigger release function

### Code Refactoring

- Address golangci-lint issues
- Remove Jenkins pipelines logic
- Add tests for Controllers
- Codebase and Codebasebranch ctrl
- Disable jira integration by default
- Address staticcheck issues
- Update VCS model
- Add unit tests for project creation in Gerrit
- Fix issue with template rendering
- Increase code coverage for gerrit pkg
- Remove duplicate code
- Add validation and checks in put_cd_stage_deploy
- Decrease Jenkins job status poll timeout
- Remove duplicate code
- Expand codebase-operator role
- Add namespace field in roleRef in OKD RB, align CRB name
- Push with default branch for GitLabCI flow
- Replace cluster-wide role/rolebinding to namespaced

### Testing

- Do not run SetupWithManager unit tests
- Add test for cdstagedeploy controller
- Add test for jira adapter part
- Exclude helm-chart templates from sonar scan
- Add tests for cluster package
- Add tests for jenkins package
- Increase coverage for jenkins client
- Add test for jenkins client

### Routine

- Add changelog generator
- Update andygrunwald/go-jira to the latest stable
- Update cd-pipeline, jenkins, component to the latest stable
- update Go version at codecov.yaml
- Use custom go build step for operator
- Update go to version 1.17
- Update docker image
- Run CodeQL only on push to master branch
- Add codecov integration

### Documentation

- Update the links on GitHub


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


[Unreleased]: https://github.com/epam/edp-codebase-operator/compare/v2.10.0...HEAD
[v2.10.0]: https://github.com/epam/edp-codebase-operator/compare/v2.9.0...v2.10.0
[v2.9.0]: https://github.com/epam/edp-codebase-operator/compare/v2.8.1...v2.9.0
[v2.8.1]: https://github.com/epam/edp-codebase-operator/compare/v2.8.0...v2.8.1
[v2.8.0]: https://github.com/epam/edp-codebase-operator/compare/v2.7.2...v2.8.0
[v2.7.2]: https://github.com/epam/edp-codebase-operator/compare/v2.7.1...v2.7.2
[v2.7.1]: https://github.com/epam/edp-codebase-operator/compare/v2.7.0...v2.7.1
[v2.7.0]: https://github.com/epam/edp-codebase-operator/compare/v2.3.0-95...v2.7.0
