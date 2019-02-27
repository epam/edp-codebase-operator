package vcs

import (
	"business-app-handler-controller/models"
	"business-app-handler-controller/pkg/vcs/impl/gitlab"
	"fmt"
	"log"
)

type VCS interface {
	CreateProject(vcsProjectName, vcsGroupId string) (string, error)
	GetGroupIdByName(groupName string) (string, error)
	GetRepositorySshUrl(projectPath string) (string, error)
	CheckProjectExist(projectPath string) (*bool, error)
	Init(url string, username string, password string) error
	DeleteProject(projectId string) error
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
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid VCS tool. Currently we do not support %v", vcsToolName)
	}
}
