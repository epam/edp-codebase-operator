# Jira Fix Version Controller

**Jira Fix Version** is the representation of the Fix Version in Jira that automates a routine work by simplifying 
the tracking and updating of the Fix Version fields.

The main purpose of a Jira Fix Version controller is to watch changes in the respective Kubernetes Custom Resource 
(JiraFixVersion CR) and to ensure that the state in that resource is applied in the corresponding Jira Server.
 
Inspect the main steps performed in the reconcile loop on the diagram below:

![arch](http://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/epmd-edp/codebase-operator/release/2.5/documentation/puml/jira_fix_version_chain.puml&raw=true)

The diagram above displays the following steps:

- *Check Parent Jira Server CR*. Each Jira Fix Version has a parent Jira Server. Firstly, the parent codebase is retrieved
by the `*spec.codebaseName"` field. Secondly, the Jira Server CR is retrieved by the `codebave.spec.jiraServer` field. If one of the
aforementioned components is not available, the loop ends up with an error.
- *Ensure Fix Version*. The purpose of this step is to create a Fix Version in Jira Server by the `metadata.name` field. Jira 
Project Id is get by the first item in the `spec.tickets` field. The URL and required credentials are retrieved from the parent
Jira Server CR.
- *Ensure Jira Tickets With Fix Version*. Jira Fix Version is added to all Jira tickets specified in the `spec.tickets` field.
- *Delete Jira Fix Version CR*. When the previous steps are successfully completed, the controller removes the respective 
Jira Fix Version Custom Resource. By design, Jira Fix Version Custom Resource is disposable and it is not recommended to make any changes in it. 
To avoid redundant calls to Jira Server, this CR should be deleted.