package chain

import (
	"github.com/go-resty/resty/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/repository"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/controllers/gitserver"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const (
	logTypeKey              = "type"
	logChainSelectedMessage = "chain is selected"
)

// TODO: remove global variable log.
var log = ctrl.Log.WithName("codebase_handler")

func MakeGerritDefChain(c client.Client, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info(logChainSelectedMessage, logTypeKey, "gerrit")

	ch := &chain{}
	gp := &gitserver.GitProvider{}

	ch.Use(
		NewPutProjectGerrit(c, cr, gp),
		NewPutGerritReplication(c),
		NewPutPerfDataSources(c),
		NewPutDeployConfigs(c, cr, gp),
		NewPutVersionFile(c, cr, gp),
		NewPutJenkinsFolder(c),
		NewPutDefaultCodeBaseBranch(c),
		NewCleaner(c),
	)

	return ch
}

func MakeGerritTektonChain(c client.Client, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info(logChainSelectedMessage, logTypeKey, "gerrit")

	ch := &chain{}
	gp := &gitserver.GitProvider{}

	ch.Use(
		NewPutProjectGerrit(c, cr, gp),
		NewPutGerritReplication(c),
		NewPutPerfDataSources(c),
		NewPutDeployConfigs(c, cr, gp),
		NewPutVersionFile(c, cr, gp),
		NewPutDefaultCodeBaseBranch(c),
		NewCleaner(c),
	)

	return ch
}

func MakeThirdPartyVcsProviderDefChain(c client.Client, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info(logChainSelectedMessage, logTypeKey, "third party VCS provider")

	ch := &chain{}
	gp := &gitserver.GitProvider{}

	ch.Use(
		NewCloneGitProject(c, gp),
		NewPutPerfDataSources(c),
		NewPutDeployConfigsToGitProvider(c, cr, gp),
		NewPutVersionFile(c, cr, gp),
		NewPutJenkinsFolder(c),
		NewPutDefaultCodeBaseBranch(c),
		NewCleaner(c),
	)

	return ch
}

func MakeDeletionChain(c client.Client, codebase *codebaseApi.Codebase) handler.CodebaseHandler {
	ch := &chain{}

	if codebase.Spec.CiTool == util.Tekton {
		ch.Use(NewDeleteWebHook(c, resty.New()))
	}

	if codebase.Spec.CiTool != util.Tekton {
		ch.Use(NewDropJenkinsFolders(c))
	}

	return ch
}

func MakeGitlabCiDefChain(c client.Client, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info(logChainSelectedMessage, logTypeKey, "gitlab ci")

	ch := &chain{}
	gp := &gitserver.GitProvider{}

	ch.Use(
		NewCloneGitProject(c, gp),
		NewPutPerfDataSources(c),
		NewPutGitlabCiDeployConfigs(c, cr, gp),
		NewPutGitlabCiFile(c, cr, gp),
		NewPutVersionFile(c, cr, gp),
		NewPutDefaultCodeBaseBranch(c),
		NewCleaner(c),
	)

	return ch
}

func MakeTektonCiDefChain(c client.Client, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info(logChainSelectedMessage, logTypeKey, "tekton")

	ch := &chain{}
	gp := &gitserver.GitProvider{}

	ch.Use(
		NewPutWebHook(c, resty.New()),
		NewCloneGitProject(c, gp),
		NewPutPerfDataSources(c),
		NewPutDeployConfigsToGitProvider(c, cr, gp),
		NewPutVersionFile(c, cr, gp),
		NewPutDefaultCodeBaseBranch(c),
		NewCleaner(c),
	)

	return ch
}
