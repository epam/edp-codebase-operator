# EDP Codebase Operator — AI Agent Instructions

This is a Kubernetes operator built with [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime), written in Go. It manages git-repository lifecycle resources (Codebase, CodebaseBranch, GitServer, etc.) on Kubernetes/OpenShift clusters as part of the EPAM Delivery Platform (EDP).

## Build & Test Commands

```bash
make build          # Build operator binary (CGO_ENABLED=0)
make test           # Run tests with coverage (runs fmt + vet + envtest first)
make lint           # Run golangci-lint v2.8.0
make lint-fix       # Auto-fix linting issues
make fmt            # go fmt
make vet            # go vet
make manifests      # Regenerate CRDs, RBAC, webhooks (controller-gen)
make generate       # Regenerate DeepCopy methods
make mocks          # Regenerate mocks via mockery v3.6.2
make api-docs       # Regenerate docs/api.md
make validate-docs  # Verify docs are current
```

After modifying types in `api/v1/`, always run `make manifests && make generate`.
After modifying interfaces in `pkg/`, run `make mocks` to regenerate mocks.

## Architecture

```
api/v1/           → CRD type definitions (Codebase, CodebaseBranch, GitServer, etc.)
api/v1alpha1/     → Alpha CRD types (Template)
controllers/      → One subdirectory per controller (reconciler + handler chain)
pkg/              → Shared business logic (git, gitlab, gerrit, util, webhook, …)
cmd/main.go       → Operator entrypoint: manager init + controller registration
config/           → Kustomize manifests (CRDs, RBAC, webhooks)
deploy-templates/ → Helm chart
mocks/            → Auto-generated mocks (do not edit manually)
```

**Module:** `github.com/epam/edp-codebase-operator/v2`

## Controller Pattern

Each controller in `controllers/<name>/` follows this structure:

- **Reconciler struct** holds `client.Client`, `scheme`, `log logr.Logger`, and a `chainGetter` for the handler chain.
- **`SetupWithManager(mgr)`** registers custom predicates (e.g., `codebasepredicate.NewPause`, `predicate.Funcs`) then calls `ctrl.NewControllerManagedBy(mgr).For(...).Complete(r)`.
- **`Reconcile(ctx, req)`** pattern:
  1. Fetch CR; ignore `IsNotFound`
  2. Apply object modifications (defaults via `objectmodifier`)
  3. Set/handle finalizers
  4. Execute handler chain
  5. Update `.status` via `r.client.Status().Update(ctx, cr)`
  6. Return `reconcile.Result{RequeueAfter: util.GetTimeout(failureCount, timeout)}` on error

**Finalizer name convention:** `"<resource>.operator.finalizer.name"` (e.g., `"codebase.operator.finalizer.name"`)

**Status constants:** `util.StatusFinished`, `util.StatusFailed` (from `pkg/util/`)

## Testing Conventions

- Test files: `*_test.go` alongside the file under test.
- Always call `t.Parallel()` at the top of each test function and sub-test.
- Use **table-driven tests** (`tests := []struct{ ... }{ ... }`).
- Assertions: `github.com/stretchr/testify/require` (fail-fast preferred over `assert`).
- Mocking: `github.com/stretchr/testify/mock` with `mock.On(...).Return(...)`. Generated mocks live in `mocks/` subdirectories (e.g., `controllers/codebase/mocks/`).
- Fake Kubernetes clients: `fake.NewClientBuilder().WithScheme(scheme).WithObjects(...).Build()`.
- Always add CRD types to the scheme in tests: `require.NoError(t, codebaseApi.AddToScheme(scheme))`.

## Code Generation Markers

Place these on type definitions in `api/v1/`:

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:rbac:groups=v2.edp.epam.com,resources=codebases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:validation:Enum=create;clone;import
// +kubebuilder:default:="tekton"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=.status.status
```

Long lines from markers are excluded from the `lll` linter in `api/`.

## Common Patterns

**Logging** — always use context-derived logger:
```go
log := ctrl.LoggerFrom(ctx)
log.Info("Reconciling Codebase", "name", req.Name)
log.Error(err, "Failed to fetch CR")
```

**Error wrapping:**
```go
return fmt.Errorf("failed to fetch Codebase: %w", err)
```

**Kubernetes error checks:**
```go
k8sErrors.IsNotFound(err)
k8sErrors.IsAlreadyExists(err)
```

**Utility helpers** (`pkg/util/`):
- `util.GetStringP(val)` — string to `*string`
- `util.GetWorkDir(name, namespace)` — respects `WORKING_DIR` env var
- `util.GetAssetsDir()` — respects `ASSETS_DIR` env var
- `util.GetTimeout(failureCount, base)` — exponential backoff duration

## Key CRD Types

| Type | Purpose |
|------|---------|
| `Codebase` | Git repository resource; strategies: `create`, `clone`, `import` |
| `CodebaseBranch` | Branch within a Codebase |
| `GitServer` | VCS connection config (GitHub, GitLab, Gerrit, Bitbucket) |
| `JiraServer` | JIRA integration settings |
| `CodebaseImageStream` | Container image stream tracking |
| `CDStageDeploy` | CD pipeline stage deployment record |

## Git Commit Format

```
[EPMDEDP-XXXXX: ]<type>: <subject>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
Example: `EPMDEDP-12345: feat: add reconciliation for QuickLink`

See [CONTRIBUTING.md](CONTRIBUTING.md) and [.chglog/config.yml](.chglog/config.yml).

## Linting

Config: `.golangci.yaml` (golangci-lint v2.8.0).
Key linters: `errcheck`, `govet`, `staticcheck`, `gocyclo`, `revive`, `ginkgolinter`, `wsl_v5`.
Formatters: `gofmt`, `goimports` (imports must be grouped).
Generated files (`zz_generated.*`, `mocks/`) are excluded from most rules.

## Further References

- [API docs](docs/api.md)
- [Helm chart](deploy-templates/README.md)
- [Local development guide](https://epam.github.io/edp-install/developer-guide/local-development/)
