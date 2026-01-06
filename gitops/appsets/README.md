# ArgoCD ApplicationSets

This directory contains ArgoCD ApplicationSets that deploy applications to multiple clusters automatically.

## ApplicationSets

### 1. `cockroachdb-appset.yaml`
Deploys CockroachDB to both `dc-us` and `dc-eu` clusters with region-specific configuration.

### 2. `ledger-app-appset.yaml`
Deploys the Ledger App to both `dc-us` and `dc-eu` clusters with region-specific AWS endpoints and configurations.

## Prerequisites

1. **ArgoCD installed** on both clusters (via `argocd-install` Ansible role)
2. **Helm charts ready** in `gitops/charts/`:
   - `cockroachdb/` chart
   - `ledger-app/` chart
3. **Git repository** (or local filesystem access for ArgoCD)

## Usage

### Option 1: Apply directly to ArgoCD (Local Development)

If you're running locally and ArgoCD can access the filesystem:

```bash
# Apply to US-East cluster's ArgoCD
kubectl --context k3d-dc-us apply -f cockroachdb-appset.yaml
kubectl --context k3d-dc-us apply -f ledger-app-appset.yaml

# Apply to EU-Central cluster's ArgoCD
kubectl --context k3d-dc-eu apply -f cockroachdb-appset.yaml
kubectl --context k3d-dc-eu apply -f ledger-app-appset.yaml
```

### Option 2: Use Git Repository (Recommended for Production)

1. **Push your GitOps repo to Git:**
```bash
git add .
git commit -m "Add ApplicationSets"
git push
```

2. **Update ApplicationSets to use Git repo:**
```yaml
# Change repoURL in ApplicationSets
source:
  repoURL: https://github.com/your-org/your-gitops-repo  # ← Your Git repo
  targetRevision: main
  path: charts/cockroachdb
```

3. **Apply ApplicationSets:**
```bash
kubectl --context k3d-dc-us apply -f cockroachdb-appset.yaml
kubectl --context k3d-dc-us apply -f ledger-app-appset.yaml
```

## What Gets Created

Each ApplicationSet automatically creates:

### CockroachDB ApplicationSet:
- `k3d-dc-us-cockroachdb` Application → Deploys CockroachDB to US cluster
- `k3d-dc-eu-cockroachdb` Application → Deploys CockroachDB to EU cluster

### Ledger App ApplicationSet:
- `k3d-dc-us-ledger-app` Application → Deploys Ledger App to US cluster
- `k3d-dc-eu-ledger-app` Application → Deploys Ledger App to EU cluster

## Region-Specific Configuration

### CockroachDB:
- **US-East**: `region=us-east-1`, `locality="region=us-east-1"`
- **EU-Central**: `region=eu-central-1`, `locality="region=eu-central-1"`

### Ledger App:
- **US-East**: 
  - Endpoint: `http://localhost:4566`
  - S3 Bucket: `us-east-1-audit-logs`
  - SQS Queue: `us-east-1-transaction-queue`
- **EU-Central**:
  - Endpoint: `http://localhost:4567`
  - S3 Bucket: `eu-central-1-audit-logs`
  - SQS Queue: `eu-central-1-transaction-queue`

## Verifying Deployment

### Check ApplicationSet status:
```bash
kubectl --context k3d-dc-us get applicationset -n argocd
```

### Check Applications created:
```bash
kubectl --context k3d-dc-us get applications -n argocd
```

### Check Application sync status:
```bash
kubectl --context k3d-dc-us get application k3d-dc-us-cockroachdb -n argocd
kubectl --context k3d-dc-us get application k3d-dc-us-ledger-app -n argocd
```

### View in ArgoCD UI:
```bash
# Get ArgoCD external IP
kubectl --context k3d-dc-us get svc argocd-server -n argocd

# Open in browser
open http://<EXTERNAL_IP>:80
```

## Troubleshooting

### Issue: ApplicationSet not creating Applications
**Solution:** Check if ArgoCD can access the repository:
```bash
kubectl --context k3d-dc-us logs -n argocd -l app.kubernetes.io/name=argocd-applicationset-controller
```

### Issue: Applications stuck in "Unknown" or "Syncing"
**Solution:** Check application logs:
```bash
kubectl --context k3d-dc-us describe application k3d-dc-us-cockroachdb -n argocd
```

### Issue: Repository not found
**Solution:** For local development, you may need to:
1. Use a Git repository instead of `file:///gitops`
2. Or configure ArgoCD to access local filesystem
3. Or mount the gitops directory into ArgoCD repo-server

## Notes

- **Local Development**: The `file:///gitops` repoURL works if ArgoCD repo-server has access to the filesystem
- **Production**: Use a Git repository URL (GitHub, GitLab, etc.)
- **Auto-sync**: Both ApplicationSets have `automated: true` for automatic syncing
- **Self-healing**: `selfHeal: true` ensures manual changes are reverted

