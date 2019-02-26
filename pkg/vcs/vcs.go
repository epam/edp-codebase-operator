package vcs

import (
	"business-app-handler-controller/models"
	"business-app-handler-controller/pkg/vcs/impl/bitbucket"
	"business-app-handler-controller/pkg/vcs/impl/gitlab"
	"fmt"
	"log"
)

type VCS interface {
	CheckProjectExist(groupPath, projectName string) (*bool, error)
	CreateProject(groupPath, projectName string) (string, error)
	GetRepositorySshUrl(groupPath, projectName string) (string, error)
}

func CreateVCSClient(vcsToolName models.VCSTool, url string, username string, password string) (VCS, error) {
	switch vcsToolName {
	case models.GitLab:
		log.Print("Creating VCS for GitLab implementation...")
		vcsClient := gitlab.GitLab{}
		err := vcsClient.Init(url, username, password)
		if err != nil {
			return nil, err
		}
		return &vcsClient, nil
	case models.BitBucket:
		log.Print("Creating VCS for BitBucket implementation...")
		vcsClient := bitbucket.BitBucket{}
		err := vcsClient.Init(url, username, password)
		if err != nil {
			return nil, err
		}
		return &vcsClient, nil
	default:
		return nil, fmt.Errorf("invalid VCS tool. Currently we do not support %v", vcsToolName)
	}
}
