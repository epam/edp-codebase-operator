# Git Server Server Controller

**Git Server** is the representation of Git Server that is used to communicate with Git using Rest and SSH connections
to work with repositories. Specified ssh credentials is store in Kubernetes secret by name `spec.nameSshKeySecret`

The main purpose of a Git Server controller is to watch changes in the respective Kubernetes Custom Resource (Git Server CR)
 and to ensure that the state in that resource is applied in EPAM Delivery Platform.
 
Inspect the main steps performed in the reconcile loop on the diagram below:

![arch](http://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/epmd-edp/codebase-operator/master/documentation/puml/git_server_chain.puml)

The diagram above displays the following steps:

- *Ensure Connection to Git Server*. The controller tries to establish ssh connection with server by url `spec.gitHost`,
port - `spec.sshPort`, user - `spec.gitUser` and credentials that were retrieved from Kubernetes secret by name `spec.nameSshKeySecret`
If connection is not successful, the loop ends up with an error. 
- *Update Status*. The status update in the respective Git Server CR.