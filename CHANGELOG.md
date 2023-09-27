<a name="unreleased"></a>
## [Unreleased]


<a name="v2.19.0"></a>
## v2.19.0 - 2023-09-26
### Features

- Add valid error in status field of GitServer ([#15](https://github.com/epam/edp-codebase-operator/issues/15))

### Bug Fixes

- Build pipeline failed for go operator sdk on sonar step ([#12](https://github.com/epam/edp-codebase-operator/issues/12))

### Code Refactoring

- Replace edpName with helm Release Namespace value ([#10](https://github.com/epam/edp-codebase-operator/issues/10))

### Routine

- Upgrade Go to 1.20 ([#14](https://github.com/epam/edp-codebase-operator/issues/14))
- Update current development version ([#11](https://github.com/epam/edp-codebase-operator/issues/11))
- Align secret name for new secret approach ([#65](https://github.com/epam/edp-codebase-operator/issues/65))
- Update current development version ([#9](https://github.com/epam/edp-codebase-operator/issues/9))

### Reverts

- [MDTUDDM-673] Save git status in the intermediate operations
- [EPMDEDP-3421] - Fix job+provisioning bug
- [EPMDEDP-868] Add functionality generates correct ssh URL for replication config for Gerrit

### BREAKING CHANGE:


removed required fields from the GitServer status. CRD should be updated.


[Unreleased]: https://github.com/epam/edp-codebase-operator/compare/v2.19.0...HEAD
