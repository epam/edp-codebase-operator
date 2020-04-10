package chain

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/perf"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/pkg/errors"
	"net/url"
	"strconv"
)

type SetupPerf struct {
	next      handler.CodebaseHandler
	clientSet openshift.ClientSet
}

func (h SetupPerf) ServeRequest(c *v1alpha1.Codebase) error {
	rLog := log.WithValues("codebase name", c.Name)
	rLog.Info("Start setting Perf...")

	if err := h.tryToSetupPerf(c.Name, c.Namespace); err != nil {
		setFailedFields(c, edpv1alpha1.GerritRepositoryProvisioning, err.Error())
		return errors.Wrapf(err, "setup Gerrit Perf for codebase %v has been failed", c.Name)
	}
	rLog.Info("end setting up Perf")
	return nextServeOrNil(h.next, c)
}

func (h SetupPerf) tryToSetupPerf(codebaseName, namespace string) error {
	log.Info("start setting up perf")
	_, us, err := util.GetConfigSettings(h.clientSet.CoreClient, namespace)
	if err != nil {
		return errors.Wrap(err, "unable get config settings")
	}

	if us.PerfIntegrationEnabled {
		return h.setupPerf(us, codebaseName, namespace)
	}
	log.Info("Skipped perf configuration. Perf integration isn't enabled")
	return nil
}

func (h SetupPerf) setupPerf(us *model.UserSettings, codebaseName, namespace string) error {
	perfSetting := perf.GetPerfSettings(h.clientSet, namespace)
	secret := perf.GetPerfCredentials(h.clientSet, namespace)

	perfUrl := perfSetting[perf.UrlSettingsKey]
	user := string(secret["username"])
	log.Info("Username for perf integration has been retrieved", "name", user)

	pass := string(secret["password"])
	perfClient, err := perf.NewRestClient(perfUrl, user, pass)
	if err != nil {
		return errors.Wrap(err, "an error has occurred during perf client initialization")
	}

	if err := setupJenkinsPerf(perfClient, codebaseName, perfSetting[perf.JenkinsDsSettingsKey]); err != nil {
		return errors.Wrap(err, "an error has occurred during setup Jenkins Perf")
	}

	if err := setupSonarPerf(perfClient, codebaseName, perfSetting[perf.SonarDsSettingsKey]); err != nil {
		return errors.Wrap(err, "an error has occurred during setup Sonar Perf")
	}

	if err := setupGerritPerf(perfClient, codebaseName, perfSetting[perf.GerritDsSettingsKey]); err != nil {
		return errors.Wrap(err, "Error has occurred during setup Gerrit Perf")
	}

	if err := trySetupGitlabPerf(perfClient, perfSetting[perf.GitlabDsSettingsKey], us, codebaseName); err != nil {
		return errors.Wrap(err, "Error has occurred during setup Gitlab Perf")
	}
	log.Info("Perf integration has been successfully finished")
	return nil
}

func setupJenkinsPerf(client *perf.Client, codebaseName string, dsId string) error {
	log.Info("start setting up Jenkins Perf", "codebase name", codebaseName)
	jenkinsDsID, err := strconv.Atoi(dsId)
	if err != nil {
		return err
	}
	jenkinsJobs := []string{fmt.Sprintf("/Code-review-%s", codebaseName), fmt.Sprintf("/Build-%s", codebaseName)}
	return client.AddJobsToJenkinsDS(jenkinsDsID, jenkinsJobs)
}

func setupSonarPerf(client *perf.Client, codebaseName string, dsId string) error {
	log.Info("start setting up Sonar Perf", "codebase name", codebaseName)
	sonarDsID, err := strconv.Atoi(dsId)
	if err != nil {
		return err
	}
	sonarProjects := []string{fmt.Sprintf("%s:master", codebaseName)}
	return client.AddProjectsToSonarDS(sonarDsID, sonarProjects)
}

func setupGerritPerf(client *perf.Client, codebaseName string, dsId string) error {
	log.Info("start setting up Gerrit Perf", "codebase name", codebaseName)
	gerritDsID, err := strconv.Atoi(dsId)
	if err != nil {
		return err
	}

	gerritProjects := []perf.GerritPerfConfig{{ProjectName: codebaseName, Branches: []string{"master"}}}
	return client.AddProjectsToGerritDS(gerritDsID, gerritProjects)
}

func trySetupGitlabPerf(client *perf.Client, dsId string, us *model.UserSettings, codebaseName string) error {
	if isGitLab(us.VcsIntegrationEnabled, us.VcsToolName) {
		vcsGroupNameUrl, err := url.Parse(us.VcsGroupNameUrl)
		if err != nil {
			return err
		}
		vcspp := fmt.Sprintf("%v/%v", vcsGroupNameUrl.Path[1:len(vcsGroupNameUrl.Path)], codebaseName)
		return setupGitlabPerf(client, vcspp, dsId)
	}
	return nil
}

func isGitLab(vcsIntegrationEnabled bool, vcsToolName model.VCSTool) bool {
	return vcsIntegrationEnabled && vcsToolName == model.GitLab
}

func setupGitlabPerf(client *perf.Client, codebaseName string, dsId string) error {
	log.Info("start setting up GitLab Perf", "codebase name", codebaseName)
	gitDsID, err := strconv.Atoi(dsId)
	if err != nil {
		return err
	}

	gitProjects := map[string]string{codebaseName: "master"}
	return client.AddRepositoriesToGitlabDS(gitDsID, gitProjects)
}
