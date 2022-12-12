package v1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	GitProviderGerrit = "gerrit"
	GitProviderGithub = "github"
	GitProviderGitlab = "gitlab"
)

// GitServerSpec defines the desired state of GitServer.
type GitServerSpec struct {
	GitHost string `json:"gitHost"`

	// GitUser is a user name for git server.
	// +kubebuilder:default:=git
	// +optional
	GitUser string `json:"gitUser"`

	HttpsPort int32 `json:"httpsPort"`

	SshPort int32 `json:"sshPort"`

	NameSshKeySecret string `json:"nameSshKeySecret"`

	// GitProvider is a git provider type. It can be gerrit, github or gitlab. Default value is gerrit.
	// +kubebuilder:validation:Enum=gerrit;gitlab;github
	// +kubebuilder:default:=gerrit
	// +optional
	GitProvider string `json:"gitProvider,omitempty"`
}

// GitServerStatus defines the observed state of GitServer.
type GitServerStatus struct {
	// This flag indicates neither JiraServer are initialized and ready to work. Defaults to false.
	Available bool `json:"available"`

	// Information when the last time the action were performed.
	LastTimeUpdated metaV1.Time `json:"last_time_updated"`

	// Specifies a current status of GitServer.
	Status string `json:"status"`

	// Name of user who made a last change.
	Username string `json:"username"`

	// The last Action was performed.
	Action string `json:"action"`

	// A result of an action which were performed.
	// - "success": action where performed successfully;
	// - "error": error has occurred;
	Result string `json:"result"`

	// Detailed information regarding action result
	// which were performed
	// +optional
	DetailedMessage string `json:"detailed_message,omitempty"`

	// Specifies a current state of GitServer.
	Value string `json:"value"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=gs
// +kubebuilder:storageversion

// GitServer is the Schema for the gitservers API.
type GitServer struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitServerSpec   `json:"spec,omitempty"`
	Status GitServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GitServerList contains a list of GitServer.
type GitServerList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`
	Items           []GitServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitServer{}, &GitServerList{})
}
