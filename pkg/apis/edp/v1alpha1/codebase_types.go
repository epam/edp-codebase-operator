package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CodebaseSpec defines the desired state of Codebase
// +k8s:openapi-gen=true
const (
	Create  Strategy       = "create"
	Clone   Strategy       = "clone"
	Edp     VersioningType = "edp"
	Default VersioningType = "default"
)

type VersioningType string

type Strategy string

type Versioning struct {
	Type      VersioningType `json:"type"`
	StartFrom *string        `json:"startFrom, omitempty"`
}

type Repository struct {
	Url string `json:"url"`
}

type Route struct {
	Site string `json:"site"`
	Path string `json:"path"`
}

type Database struct {
	Kind     string `json:"kind"`
	Version  string `json:"version"`
	Capacity string `json:"capacity"`
	Storage  string `json:"storage"`
}

type CodebaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Lang                string      `json:"lang"`
	Description         *string     `json:"description,omitempty"`
	Framework           *string     `json:"framework,omitempty"`
	BuildTool           string      `json:"buildTool"`
	Strategy            Strategy    `json:"strategy"`
	Repository          *Repository `json:"repository,omitempty"`
	Route               *Route      `json:"route,omitempty"`
	Database            *Database   `json:"database,omitempty"`
	TestReportFramework *string     `json:"testReportFramework,omitempty"`
	Type                string      `json:"type"`
	GitServer           string      `json:"gitServer"`
	GitUrlPath          *string     `json:"gitUrlPath,omitempty"`
	JenkinsSlave        string      `json:"jenkinsSlave"`
	JobProvisioning     string      `json:"jobProvisioning"`
	DeploymentScript    string      `json:"deploymentScript"`
	Versioning          Versioning  `json:"versioning"`
}

// CodebaseStatus defines the observed state of Codebase
// +k8s:openapi-gen=true
type CodebaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Available       bool       `json:"available"`
	LastTimeUpdated time.Time  `json:"lastTimeUpdated"`
	Status          string     `json:"status"`
	Username        string     `json:"username"`
	Action          ActionType `json:"action"`
	Result          Result     `json:"result"`
	DetailedMessage string     `json:"detailedMessage"`
	Value           string     `json:"value"`
	FailureCount    int64      `json:"failureCount"`
}

type ActionType string
type Result string

const (
	CodebaseRegistration             ActionType = "codebase_registration"
	AcceptCodebaseRegistration       ActionType = "accept_codebase_registration"
	GerritRepositoryProvisioning     ActionType = "gerrit_repository_provisioning"
	JenkinsConfiguration             ActionType = "jenkins_configuration"
	PergRegistration                 ActionType = "perf_registration"
	SetupDeploymentTemplates         ActionType = "setup_deployment_templates"
	CodebaseBranchRegistration       ActionType = "codebase_branch_registration"
	AcceptCodebaseBranchRegistration ActionType = "accept_codebase_branch_registration"
	PutS2I                           ActionType = "put_s2i"
	PutJenkinsFolder                 ActionType = "put_jenkins_folder"
	CleanData                        ActionType = "clean_data"
	ImportProject                    ActionType = "import_project"

	Success Result = "success"
	Error   Result = "error"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Codebase is the Schema for the codebases API
// +k8s:openapi-gen=true
type Codebase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodebaseSpec   `json:"spec,omitempty"`
	Status CodebaseStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CodebaseList contains a list of Codebase
type CodebaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Codebase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Codebase{}, &CodebaseList{})
}
