package chain

import (
	"context"

	"github.com/go-resty/resty/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-codebase-operator/v2/controllers/codebase/service/chain/handler"
	"github.com/epam/edp-codebase-operator/v2/pkg/gerrit"
	"github.com/epam/edp-codebase-operator/v2/pkg/git"
	"github.com/epam/edp-codebase-operator/v2/pkg/gitprovider"
)

func MakeChain(ctx context.Context, c client.Client) handler.CodebaseHandler {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Default chain is selected")

	ch := &chain{}
	gp := &git.GitProvider{}

	ch.Use(
		NewPutGitWebRepoUrl(c),
		NewPutProject(c, gp, &gerrit.SSHGerritClient{}, gitprovider.NewGitProjectProvider),
		NewPutWebHook(c, resty.New()),
		NewPutDeployConfigs(c, gp),
		NewPutDefaultCodeBaseBranch(c),
		NewCleaner(c),
	)

	return ch
}

func MakeDeletionChain(ctx context.Context, c client.Client) handler.CodebaseHandler {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Deletion chain is selected")

	ch := &chain{}

	ch.Use(
		NewDeleteWebHook(c, resty.New(), log),
		NewCleaner(c),
	)

	return ch
}
