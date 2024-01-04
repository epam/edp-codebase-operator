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

	// SkipWebhookSSLVerification is a flag to skip webhook tls verification.
	// +optional
	SkipWebhookSSLVerification bool `json:"skipWebhookSSLVerification"`

	// WebhookUrl is a URL for webhook that will be created in the git provider.
	// If it is not set, a webhook will be created from Ingress with the name "event-listener".
	// +optional
	// +kubebuilder:example:=`https://webhook-url.com`
	WebhookUrl string `json:"webhookUrl,omitempty"`
}

// GitServerStatus defines the observed state of GitServer.
type GitServerStatus struct {
	// Error represents error message if something went wrong.
	// +optional
	Error string `json:"error,omitempty"`

	// Connected shows if operator is connected to git server.
	// +optional
	Connected bool `json:"connected"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=gs
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Connected",type="boolean",JSONPath=".status.connected",description="Is connected to git server"
// +kubebuilder:printcolumn:name="Host",type="string",JSONPath=".spec.gitHost",description="GitSever host"
// +kubebuilder:printcolumn:name="Git Provider",type="string",JSONPath=".spec.gitProvider",description="Git Provider type"

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
