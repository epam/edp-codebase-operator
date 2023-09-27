<a name="unreleased"></a>
## [Unreleased]


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


<a name="v2.17.0"></a>
## [v2.17.0] - 2023-08-17

[Unreleased]: https://github.com/epam/edp-codebase-operator/compare/v2.19.0...HEAD
[v2.19.0]: https://github.com/epam/edp-codebase-operator/compare/v2.18.1...v2.19.0
[v2.18.1]: https://github.com/epam/edp-codebase-operator/compare/v2.18.0...v2.18.1
[v2.18.0]: https://github.com/epam/edp-codebase-operator/compare/v2.17.0...v2.18.0
[v2.17.0]: https://github.com/epam/edp-codebase-operator/compare/v2.16.0...v2.17.0
