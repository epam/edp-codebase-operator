# Codebase Controller Overview

**Codebase** is the representation of any repository that holds code and can be maintained with CI/CD pipelines.

The main purpose of a codebase controller is to watch changes in the respective Kubernetes Custom Resource (Codebase CR) and to ensure that the state in that resource is applied in EDP.
 
Explore the main steps performed in the reconcile loop on the diagram below:

![arch](http://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/epmd-edp/codebase-operator/master/documentation/puml/codebase_chain.puml)

There are three strategies: import, create and clone. Selecting the strategy depends on the project needs. 
The diagram above displays two branches (flows) that represents the difference between the strategies:

1. **Import** strategy uses the existing repository and integrates it to EDP.
2. **Clone** and **create** strategies require the creation of a new repository in VCS, which is used for a particular installation 
of EDP. The source code is pulling from the repository, that is specified in Codebase CR (clone strategy), or from one of 
the predefined (create strategy). 

The **import** strategy flow consists of the following steps:

- *Clone Git Repository*. The existence of the repository from Codebase CR is checked and the repository is pulled
in to the temporary workspace.
- *Ensure Deploy Config in Git*. Instructions on how to deploy this codebase in Kubernetes are added (represented as Helm charts).
- *Ensure S2i Image Stream*. The deprecated step that should be deleted in https://jiraeu.epam.com/browse/EPMDEDP-4116.
- *Ensure Jenkins Folder CR*. Custom resource for Jenkins folder is added to hold CI/CD pipelines related to this codebase.
- *Cleaner*. The technical step, it ensures that all workspaces are wiped out.

The **clone** and **create** strategy flow includes the following steps:

- *Ensure Project in Gerrit*. Ensures that the corresponding Gerrit project is created for this codebase. Cloning and pushing
of the source code from the specified repository are performed.
- *Ensure Gerrit Replication*. The replication configuration of a newly created Gerrit project is set up. The replication is
enabled if the vcs_integration_enabled field in the edp-config config map is set to true.
- *Ensure Perf Integration*. The deprecated step.
- *Ensure Deploy Config in Git*. Instructions on how to deploy this codebase in Kubernetes are added.
- *Ensure S2i Image Stream*. The deprecated step that should be deleted in https://jiraeu.epam.com/browse/EPMDEDP-4116.
- *Ensure Jenkins Folder CR*. Custom resource for Jenkins folder is added to hold CI/CD pipelines related to this codebase.
- *Cleaner*. The technical step, it ensures that all workspaces are wiped out.

### Related Articles

- [Codebase Branch Controller](documentation/codebase_branch_controller.md)