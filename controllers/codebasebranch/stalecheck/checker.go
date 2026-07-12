package stalecheck

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	gitproviderv2 "github.com/epam/edp-codebase-operator/v2/pkg/git"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

// GitClientFactory is the injection seam for tests;
// production wiring uses gitproviderv2.DefaultGitProviderFactory.
type GitClientFactory func(gitServer *codebaseApi.GitServer, secret *corev1.Secret) gitproviderv2.Git

// Checker periodically verifies that every CodebaseBranch still has a corresponding
// branch in the real git repository and applies the configured cleanup strategy.
//
// It runs as a manager Runnable (leader-only) rather than a watch-driven controller
// because staleness is external state that no Kubernetes event reports.
type Checker struct {
	client           client.Client
	namespace        string
	interval         time.Duration
	gitClientFactory GitClientFactory
	markAction       StaleBranchAction
	cleanupAction    StaleBranchAction
}

func NewChecker(
	k8sClient client.Client,
	namespace string,
	interval time.Duration,
	gitClientFactory GitClientFactory,
	markAction StaleBranchAction,
	cleanupAction StaleBranchAction,
) *Checker {
	return &Checker{
		client:           k8sClient,
		namespace:        namespace,
		interval:         interval,
		gitClientFactory: gitClientFactory,
		markAction:       markAction,
		cleanupAction:    cleanupAction,
	}
}

// Start implements manager.Runnable. It sweeps once on startup and then on every tick.
func (c *Checker) Start(ctx context.Context) error {
	log := ctrl.Log.WithName("stale-branch-checker")
	ctx = ctrl.LoggerInto(ctx, log)

	log.Info("Starting stale branch checker", "interval", c.interval)

	c.sweep(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping stale branch checker")
			return nil
		case <-ticker.C:
			c.sweep(ctx)
		}
	}
}

// NeedLeaderElection ensures only the elected leader talks to git servers.
func (c *Checker) NeedLeaderElection() bool {
	return true
}

func (c *Checker) sweep(ctx context.Context) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Checking codebase branches staleness")

	branches := &codebaseApi.CodebaseBranchList{}
	if err := c.client.List(ctx, branches, client.InNamespace(c.namespace)); err != nil {
		log.Error(err, "Failed to list codebase branches")
		return
	}

	branchesByCodebase := make(map[string][]*codebaseApi.CodebaseBranch)

	for i := range branches.Items {
		branch := &branches.Items[i]
		branchesByCodebase[branch.Spec.CodebaseName] = append(branchesByCodebase[branch.Spec.CodebaseName], branch)
	}

	for codebaseName, codebaseBranches := range branchesByCodebase {
		if err := c.checkCodebaseBranches(ctx, codebaseName, codebaseBranches); err != nil {
			log.Error(err, "Failed to check branches staleness", "codebase", codebaseName)
		}
	}

	log.Info("Codebase branches staleness check finished", "codebases", len(branchesByCodebase))
}

func (c *Checker) checkCodebaseBranches(
	ctx context.Context,
	codebaseName string,
	branches []*codebaseApi.CodebaseBranch,
) error {
	log := ctrl.LoggerFrom(ctx).WithValues("codebase", codebaseName)

	codebase := &codebaseApi.Codebase{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: c.namespace, Name: codebaseName}, codebase); err != nil {
		return fmt.Errorf("failed to get codebase %s: %w", codebaseName, err)
	}

	gitServer := &codebaseApi.GitServer{}
	if err := c.client.Get(
		ctx, client.ObjectKey{Namespace: c.namespace, Name: codebase.Spec.GitServer}, gitServer,
	); err != nil {
		return fmt.Errorf("failed to get git server %s: %w", codebase.Spec.GitServer, err)
	}

	secret := &corev1.Secret{}
	if err := c.client.Get(
		ctx, client.ObjectKey{Namespace: c.namespace, Name: gitServer.Spec.NameSshKeySecret}, secret,
	); err != nil {
		return fmt.Errorf("failed to get secret %s: %w", gitServer.Spec.NameSshKeySecret, err)
	}

	action := c.markAction
	if codebase.Annotations[codebaseApi.BranchCleanupStrategyAnnotation] == codebaseApi.BranchCleanupStrategyAuto {
		action = c.cleanupAction
	}

	repoURL := util.GetProjectGitUrl(gitServer, secret, codebase.Spec.GetProjectID())

	// A failed listing means the repository state is unknown; branches are marked stale
	// only on a successful listing that lacks them, never on connectivity/auth errors.
	remoteBranches, err := c.gitClientFactory(gitServer, secret).ListRemoteBranches(ctx, repoURL)
	if err != nil {
		return fmt.Errorf("failed to list remote branches for %s, skipping staleness check: %w", repoURL, err)
	}

	existsInGit := make(map[string]struct{}, len(remoteBranches))
	for _, name := range remoteBranches {
		existsInGit[name] = struct{}{}
	}

	for _, branch := range branches {
		if !c.eligibleForCheck(branch, codebase) {
			continue
		}

		_, exists := existsInGit[branch.Spec.BranchName]

		if err := action.Apply(ctx, branch, Verdict{ExistsInGit: exists}); err != nil {
			log.Error(err, "Failed to apply staleness verdict", "branch", branch.Name)
		}
	}

	return nil
}

// eligibleForCheck filters out branches whose absence in git is expected or must not be acted on:
// branches not yet pushed by the operator, branches being deleted, and the codebase default
// branch (its disappearance almost always means repository migration/rename, not branch cleanup).
func (c *Checker) eligibleForCheck(branch *codebaseApi.CodebaseBranch, codebase *codebaseApi.Codebase) bool {
	if branch.DeletionTimestamp != nil {
		return false
	}

	if branch.Status.Git != codebaseApi.CodebaseBranchGitStatusBranchCreated {
		return false
	}

	return branch.Spec.BranchName != codebase.Spec.DefaultBranch
}
