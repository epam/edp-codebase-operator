# Jira Server Controller

**Jira Server** is the representation of Jira that is used to communicate with Jira API to manage different entities 
(tickets, fix version, etc.)

The main purpose of a Jira Server controller is to watch changes in the respective Kubernetes Custom Resource (Codebase CR)
 and to ensure that the state in that resource is applied in EPAM Delivery Platform.
 
Inspect the main steps performed in the reconcile loop on the diagram below:

![arch](http://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/epmd-edp/codebase-operator/master/documentation/puml/jira_server_chain.puml)

The diagram above displays the following steps:

- *Ensure Connection to Jira Server*. The controller tries to log in to the specified URL using spec.ApiUrl and spec.credentialName. 
If connection is not successful, the loop ends up with an error. 
- *Put EDP Component*. Registration of a new component in EDP.
- *Update Status*. The status update in the respective Jira Server CR.