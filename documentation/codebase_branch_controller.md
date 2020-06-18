# Codebase Branch Controller Overview

**Codebase Branch** is the representation of any git branch for particular codebase.

The main purpose of a codebase branch controller is to watch changes in the respective Kubernetes Custom Resource
 (Codebase Branch CR) and to ensure that the state in that resource is applied in EDP.
 
Explore the main steps performed in the reconcile loop on the diagram below:

![arch](http://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/epmd-edp/codebase-operator/master/documentation/puml/codebase_branch_chain.puml)

There are 2 versioning types presented in Codebase (field spec.versioning.type):
1. **EDP**. Versioning of the Codebase is managed in Admin Console by EDP Administrator. Version for particular builds
is assembled from the Kubernetes CR value 
2. **default**. This type fully depends on default version in respective build tools and is managed by developers.
Version is assembled using default flow and values from the build tool descriptors (pom.xml, package.json etc)

Reconcile loop for Codebase Branch CR includes the following steps:
- *Check parent codebase* Each codebase branch has parent codebase and it cannot be reconciled until the parent codebase finished its reconcilation.
Parent codebase is retrieving by field *spec.codebaseName" and is the status (field status.available) is checked with the
value true. Otherwise, loop ends with error
- *Check Jenkins Folder* Each of the Codebase has respective Jenkins Folder that is presented as CR.
 Jenkins Folder CR name is build by the following convention: _spec.codebaseName-**codebase**_. Status (field status.available)
 of the corresponding Jenkins Folder CR should have value true. Otherwise, loop ends with error
- *Check new version/ is spec.version in status.history* Codebase branch holds the history of the all versions that was applied for this branch in the
field status.history including current (field spec.version). The version is considered as new if it is not presented
in the history
- *Reset Jenkins build count* Codebase Branch CR holds the overall build count for the current version (field status.build).
When new version for this Codebase Branch is set, build count should be reset to 0.
- *Reset Jenkins last successful build* Codebase Branch CR holds the number of the last successful build for the current version (field status.build).
When new version for this Codebase Branch is set, last successful build should be reset to 0.
- *Trigger Jenkins Release Job* The whole process of provisioning Jenkins jobs for CI of the current Branch is made
 in the special Jenkins job named Create-release-CodebaseName that should be created during the Codebase CR provisioning.
 Codebase controller triggers the job with the parameters (branch name and commit id) and ensures that job has blue status
 (finished successfully)