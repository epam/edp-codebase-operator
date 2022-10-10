package chain

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/repository"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/gitserver"
)

// TODO: remove global variable log.
var log = ctrl.Log.WithName("codebase_handler")

func MakeGerritDefChain(c client.Client, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info("chain is selected", "type", "gerrit")

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
	log.Info("chain is selected", "type", "gerrit")

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
	log.Info("chain is selected", "type", "third party VCS provider")

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

func MakeDeletionChain(c client.Client) handler.CodebaseHandler {
	ch := &chain{}

	ch.Use(NewDropJenkinsFolders(c))

	return ch
}

func MakeGitlabCiDefChain(c client.Client, cr repository.CodebaseRepository) handler.CodebaseHandler {
	log.Info("chain is selected", "type", "gitlab ci")

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
	log.Info("chain is selected", "type", "tekton")

	ch := &chain{}
	gp := &gitserver.GitProvider{}

	ch.Use(
		NewCloneGitProject(c, gp),
		NewPutPerfDataSources(c),
		NewPutDeployConfigsToGitProvider(c, cr, gp),
		NewPutVersionFile(c, cr, gp),
		NewPutDefaultCodeBaseBranch(c),
		NewCleaner(c),
	)

	return ch
}
