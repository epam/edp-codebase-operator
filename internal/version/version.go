package version

import (
	"fmt"
	"runtime"
)

var (
	version        = "XXXX"
	buildDate      = "1970-01-01T00:00:00Z"
	gitCommit      = ""
	gitTag         = ""
	kubectlVersion = ""
)

type Version struct {
	Version        string `json:"version"`
	BuildDate      string `json:"build_data"`
	GitCommit      string `json:"git_commit"`
	GitTag         string `json:"git_tag"`
	Go             string `json:"go-version"`
	KubectlVersion string `json:"kubectl_version"`
	Platform       string `json:"platform"`
}

func (v Version) String() string {
	return v.Version
}

func Get() Version {
	return Version{
		Version:        version,
		BuildDate:      buildDate,
		GitCommit:      gitCommit,
		GitTag:         gitTag,
		Go:             runtime.Version(),
		Platform:       fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		KubectlVersion: kubectlVersion,
	}
}
