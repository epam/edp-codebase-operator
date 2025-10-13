# GitLab CI ConfigMap Templates

This directory contains example ConfigMaps for GitLab CI template injection. These ConfigMaps replace the hard-coded component URLs in the GitLab CI manager, allowing for runtime updates without operator rebuilds.

## ConfigMap Naming Convention

ConfigMaps follow a predictable naming pattern:

- Specific: `gitlab-ci-{language}-{buildtool}` (e.g., `gitlab-ci-java-maven`)
- Language fallback: `gitlab-ci-{language}` (e.g., `gitlab-ci-java`)
- Default fallback: `gitlab-ci-default`

## Template Structure

Each ConfigMap contains a `.gitlab-ci.yml` template with:

- Workflow rules for merge requests, protected branches, and semantic version tags
- Global variables customizable per project type
- Component inclusion with conditional pipeline execution
- Template substitution using `{{.CodebaseName}}` placeholder

## Deployment

Apply the ConfigMaps to your cluster:

```bash
kubectl apply -f gitlab-ci-java-maven.yaml
kubectl apply -f gitlab-ci-go.yaml
kubectl apply -f gitlab-ci-default.yaml
```

## Version Updates

To update CI component versions, simply patch the ConfigMap:

```bash
# Update Java Maven CI component from 0.1.1 to 0.1.2
kubectl patch configmap gitlab-ci-java-maven -n krci --type='merge' -p='
{
  "data": {
    ".gitlab-ci.yml": "... updated template with @0.1.2 ..."
  }
}'
```

## Fallback Hierarchy

The GitLab CI manager uses this fallback hierarchy:

1. Specific ConfigMap: `gitlab-ci-java-maven`
2. Language ConfigMap: `gitlab-ci-java`
3. Default ConfigMap: `gitlab-ci-default`

This ensures graceful handling of unsupported language/build tool combinations.

## Benefits

- No operator rebuilds for CI component updates
- Runtime template updates via ConfigMap changes
- Kubernetes-native configuration management
- Version control of CI templates outside operator
- Easy customization per environment/organization
