# Git Server Controller

**Git Server** is the representation of Git Server that is used to communicate with Git using Rest and SSH connections
to work with repositories. Specified ssh credentials are stored in Kubernetes secret by the `spec.nameSshKeySecret` name.

The main purpose of a Git Server controller is to watch changes in the respective Kubernetes Custom Resource (Git Server CR) 
and to ensure that the state in that resource is applied in EPAM Delivery Platform.
 
Inspect the main steps performed in the reconcile loop on the diagram below:

![arch](http://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/epmd-edp/codebase-operator/release/2.5/documentation/puml/git_server_chain.puml&raw=true)

The diagram above displays the following steps:

- *Ensure Connection to Git Server*. The controller tries to establish ssh connection with the server by the `spec.gitHost` URL, 
the `spec.sshPort` port, the `spec.gitUser` user, and credentials that were retrieved from Kubernetes secret by the `spec.nameSshKeySecret` name.
If the connection is not successful, the loop ends up with an error. 
- *Update Status*. The status update in the respective Git Server CR.