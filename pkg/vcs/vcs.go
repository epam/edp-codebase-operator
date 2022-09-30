package vcs

import (
	"fmt"
	"log"
	"net/url"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epam/edp-codebase-operator/v2/pkg/vcs/impl/bitbucket"
	"github.com/epam/edp-codebase-operator/v2/pkg/vcs/impl/gitlab"
)

type VCS interface {
	CheckProjectExist(groupPath, projectName string) (*bool, error)
	CreateProject(groupPath, projectName string) (string, error)
	GetRepositorySshUrl(groupPath, projectName string) (string, error)
}

func CreateVCSClient(vcsToolName model.VCSTool, u, username, password string) (VCS, error) {
	switch vcsToolName {
	case model.GitLab:
		log.Print("Creating VCS for GitLab implementation...")
		vcsClient := gitlab.GitLab{}
		err := vcsClient.Init(u, username, password)
		if err != nil {
			return nil, err
		}
		return &vcsClient, nil
	case model.BitBucket:
		log.Print("Creating VCS for BitBucket implementation...")
		vcsClient := bitbucket.BitBucket{}
		err := vcsClient.Init(u, username, password)
		if err != nil {
			return nil, err
		}
		return &vcsClient, nil
	default:
		return nil, fmt.Errorf("invalid VCS tool. Currently we do not support %v", vcsToolName)
	}
}

func GetVcsConfig(c client.Client, us *model.UserSettings, codebaseName, namespace string) (*model.Vcs, error) {
	vcsAutoUserLogin, vcsAutoUserPassword, err := util.GetVcsBasicAuthConfig(c, namespace,
		fmt.Sprintf("vcs-autouser-codebase-%v-temp", codebaseName))
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get secret")
	}

	vcsGroupNameUrl, err := url.Parse(us.VcsGroupNameUrl)
	if err != nil {
		return nil, err
	}

	projectVcsHostnameUrl := fmt.Sprintf("%v://%v", vcsGroupNameUrl.Scheme, vcsGroupNameUrl.Host)
	vcsTool, err := CreateVCSClient(us.VcsToolName, projectVcsHostnameUrl, vcsAutoUserLogin, vcsAutoUserPassword)
	if err != nil {
		return nil, err
	}

	vcsSshUrl, err := vcsTool.GetRepositorySshUrl(vcsGroupNameUrl.Path[1:len(vcsGroupNameUrl.Path)], codebaseName)
	if err != nil {
		return nil, err
	}

	return &model.Vcs{
		VcsSshUrl:             vcsSshUrl,
		VcsIntegrationEnabled: us.VcsIntegrationEnabled,
		VcsToolName:           us.VcsToolName,
		VcsUsername:           vcsAutoUserLogin,
		VcsPassword:           vcsAutoUserPassword,
		ProjectVcsHostnameUrl: projectVcsHostnameUrl,
		ProjectVcsGroupPath:   vcsGroupNameUrl.Path[1:len(vcsGroupNameUrl.Path)],
	}, nil
}

func CreateProjectInVcs(c client.Client, us *model.UserSettings, codebaseName, namespace string) error {
	vcsConf, err := GetVcsConfig(c, us, codebaseName, namespace)
	if err != nil {
		return err
	}

	vcsTool, err := CreateVCSClient(vcsConf.VcsToolName,
		vcsConf.ProjectVcsHostnameUrl, vcsConf.VcsUsername, vcsConf.VcsPassword)
	if err != nil {
		return errors.Wrap(err, "unable to create VCS client")
	}

	e, err := vcsTool.CheckProjectExist(vcsConf.ProjectVcsGroupPath, codebaseName)
	if err != nil {
		return err
	}

	if *e {
		log.Printf("couldn't copy project to your VCS group. Repository %v is already exists in %v", codebaseName, vcsConf.ProjectVcsGroupPath)
		return nil
	}
	_, err = vcsTool.CreateProject(vcsConf.ProjectVcsGroupPath, codebaseName)
	if err != nil {
		return err
	}
	vcsConf.VcsSshUrl, err = vcsTool.GetRepositorySshUrl(vcsConf.ProjectVcsGroupPath, codebaseName)
	if err != nil {
		return errors.Wrap(err, "Unable to get repository ssh url")
	}

	return nil
}
