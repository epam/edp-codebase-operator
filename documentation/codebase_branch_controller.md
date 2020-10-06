# Codebase Branch Controller

**Codebase Branch** is the representation of any Git branch for a particular codebase.

The main purpose of a codebase branch controller is to watch changes in the respective Kubernetes Custom Resource
 (Codebase Branch CR) and to ensure that the state in that resource is applied in EDP.
 
Explore the main steps performed in the reconcile loop on the diagram below:

![arch](http://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/epmd-edp/codebase-operator/release/2.5/documentation/puml/codebase_branch_chain.puml&raw=true)

There are two versioning types presented in the **spec.versioning.type** field of a codebase:

1. **EDP versioning**. Versioning of a codebase is managed in Admin Console by EDP Administrator. The version for particular builds
is assembled from the Kubernetes CR value. 
2. **Default versioning**. This versioning type fully depends on the default version in respective build tools, and it is managed by developers.
The version is assembled using the default flow and values from the build tool descriptors (pom.xml, package.json, etc.).

The reconcile loop for the *Codebase Branch CR* includes the following steps:

* *Check Parent Codebase*. Each codebase branch has a parent codebase and it cannot be reconciled until the parent codebase finishes its reconcilation.
The parent codebase is retrieved by the `*spec.codebaseName"` field so as the `status.available` field is checked with the
value "true". Otherwise, the loop ends up with an error.

* *Check CI Tool*. Can be selected either Jenkins or GitLab CI tool.

* Using **Jenkins Tool**:

   - *Check Jenkins Folder*. Each of a codebase has a respective Jenkins folder that is presented as a CR. The Jenkins Folder CR name is built following the convention: _spec.codebaseName-**codebase**_. The `status.available` field of the corresponding Jenkins folder CR should have the value "true:. Otherwise, the loop ends up with an error.
   - *Check new version/ is spec.version in status.history*. A codebase branch stores the history of all the versions that were applied for this branch in the `status.history` field including the current `spec.version` field. The version is considered as a new if it is not present in the history.
   - *EDP Versioning*. EDP versioning or the default versioning type, see above the difference.
   - *Reset Jenkins Build Count*. A codebase branch CR stores the overall build count for the current version (the `status.build` field). When a new version for this codebase branch is set, the build count should be reset to 0.
   - *Reset Jenkins Last Successful Build*. A codebase branch CR stores the number of the last successful build for the current version (the `status.build` field). When a new version for this codebase branch is set, the last successful build should be reset to 0.
   - *Trigger Jenkins Release Job*. For the current branch CI, the whole process of the Jenkins jobs provisioning is made in the special Jenkins job **Create-release-CodebaseName** that should be created during the codebase CR provisioning. The codebase controller triggers the job with the parameters (branch name and commit id) and ensures that the job has the blue status, i.e. has finished successfully.

* Using **GitLab CI Tool**:

    - *Reset GitLab CI Build Count*. A codebase branch CR stores the overall build count for the current version (the `status.build` field).  When a new version for this codebase branch is set, the build count should be reset to 0.                                       
    - *Create Branch in GIT Repository*. A branch in a Git provider is created.
    - *Create CodebaseImageStream CR in Cluster*. The appropriate CodebaseImageStream CR for a branch is created.
    - *Clean Temp Directory in Pod*. Temp files are removed from a pod.