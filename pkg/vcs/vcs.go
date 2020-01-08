package vcs

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/codebase-operator/v2/pkg/vcs/impl/bitbucket"
	"github.com/epmd-edp/codebase-operator/v2/pkg/vcs/impl/gitlab"
	"github.com/pkg/errors"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"log"
	"net/url"
)

type VCS interface {
	CheckProjectExist(groupPath, projectName string) (*bool, error)
	CreateProject(groupPath, projectName string) (string, error)
	GetRepositorySshUrl(groupPath, projectName string) (string, error)
}

func CreateVCSClient(vcsToolName model.VCSTool, url string, username string, password string) (VCS, error) {
	switch vcsToolName {
	case model.GitLab:
		log.Print("Creating VCS for GitLab implementation...")
		vcsClient := gitlab.GitLab{}
		err := vcsClient.Init(url, username, password)
		if err != nil {
			return nil, err
		}
		return &vcsClient, nil
	case model.BitBucket:
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

func GetVcsConfig(c coreV1Client.CoreV1Client, us *model.UserSettings, codebaseName, namespace string) (*model.Vcs, error) {
	vcsGroupNameUrl, err := url.Parse(us.VcsGroupNameUrl)
	if err != nil {
		return nil, err
	}

	projectVcsHostnameUrl := fmt.Sprintf("%v://%v", vcsGroupNameUrl.Scheme, vcsGroupNameUrl.Host)
	VcsCredentialsSecretName := fmt.Sprintf("vcs-autouser-codebase-%v-temp", codebaseName)
	vcsAutoUserLogin, vcsAutoUserPassword, err := util.GetVcsBasicAuthConfig(c, namespace, VcsCredentialsSecretName)

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
		ProjectVcsHostnameUrl: projectVcsHostnameUrl,
		ProjectVcsGroupPath:   vcsGroupNameUrl.Path[1:len(vcsGroupNameUrl.Path)],
	}, nil
}

func СreateProjectInVcs(c coreV1Client.CoreV1Client, us *model.UserSettings, codebaseName, namespace string) error {
	vcsConf, err := GetVcsConfig(c, us, codebaseName, namespace)
	if err != nil {
		return err
	}

	vcscn := fmt.Sprintf("vcs-autouser-codebase-%v-temp", codebaseName)
	vcsAutoUserLogin, vcsAutoUserPassword, err := util.GetVcsBasicAuthConfig(c, namespace, vcscn)
	vcsTool, err := CreateVCSClient(model.VCSTool(vcsConf.VcsToolName),
		vcsConf.ProjectVcsHostnameUrl, vcsAutoUserLogin, vcsAutoUserPassword)
	if err != nil {
		return errors.Wrap(err, "unable to create VCS client")
	}

	e, err := vcsTool.CheckProjectExist(vcsConf.ProjectVcsGroupPath, codebaseName)
	if err != nil {
		return err
	}

	if *e {
		log.Printf("couldn't copy project to your VCS group. Repository %s is already exists in " +
			codebaseName + "" + vcsConf.ProjectVcsGroupPath)
		return nil
	}
	_, err = vcsTool.CreateProject(vcsConf.ProjectVcsGroupPath, codebaseName)
	if err != nil {
		return err
	}
	vcsConf.VcsSshUrl, err = vcsTool.GetRepositorySshUrl(vcsConf.ProjectVcsGroupPath, codebaseName)

	return nil
}
