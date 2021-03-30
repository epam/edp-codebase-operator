# Jira Issue Metadata Controller

**Jira Issue Metadata** is the representation of metadata for Jira tickets that automates a routine work by simplifying 
the tracking and updating of metadata fields.

The main purpose of a Jira Issue Metadata controller is to watch changes in the respective Kubernetes Custom Resource 
(JiraIssueMetadata CR) and to ensure that the state in that resource is applied in the corresponding Jira ticket.
 
Inspect the main steps performed in the reconcile loop on the diagram below:

![arch](http://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/epam/edp-codebase-operator/master/documentation/puml/jira_issue_metadata.puml&raw=true)

The diagram above displays the following steps:

- *Create tag in Jira if doesn't exist*. Creates tag in Jira if current doesn't exist.
- *Apply tags to issues*. Applies created tags to issues that specified in JiraIssueMetadata CR.
- *Create issue web links in tickets*. Creates links to Jenkins builds.
- *Delete JiraIssueMetadata CR*. When the previous steps are successfully completed, the controller removes the respective.
Jira Issue Metadata Custom Resource. By design, Jira Issue Metadata Custom Resource is disposable and it is not recommended to make any changes in it. 
To avoid redundant calls to Jira Server, this CR should be deleted.