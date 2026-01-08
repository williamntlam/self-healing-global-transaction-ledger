# ArgoCD ApplicationSets

This directory contains ArgoCD ApplicationSets that deploy applications to multiple clusters automatically.

## ApplicationSets

### 1. `cockroachdb-appset.yaml`
Deploys CockroachDB to both `dc-us` and `dc-eu` clusters with region-specific configuration.

### 2. `ledger-app-appset.yaml`
Deploys the Ledger App to both `dc-us` and `dc-eu` clusters with region-specific AWS endpoints and configurations.

### 3. `global-lb-appset.yaml` ⭐ NEW
Deploys a Global Load Balancer that provides **automatic failover** between regions.

**Key Feature**: If one region goes down, requests automatically route to the healthy region.

## Global Load Balancer Setup

The global load balancer provides automatic HTTP-level failover. Here's how to set it up:

### Step 1: Discover Service Endpoints

```bash
# Get EU region LoadBalancer IP
EU_IP=$(kubectl --context k3d-dc-eu get svc ledger-app -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "EU IP: $EU_IP"

# Or use the helper script
./scripts/setup-global-lb.sh
```

### Step 2: Update ApplicationSet

Edit `global-lb-appset.yaml` and replace `CHANGE_ME` with the EU IP:

```yaml
euHost: "192.168.1.20"  # Your EU LoadBalancer IP
```

### Step 3: Deploy

```bash
kubectl --context k3d-dc-us apply -f gitops/appsets/global-lb-appset.yaml
```

### Step 4: Use Global Endpoint

```bash
# Get Global LB IP
GLOBAL_IP=$(kubectl --context k3d-dc-us get svc global-lb -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Use this endpoint for all requests (automatic failover)
curl -X POST http://$GLOBAL_IP/transactions ...
```

## How Automatic Failover Works (Bidirectional)

```
Client Request
    ↓
Global Load Balancer (NGINX)
    ↓
    ├─→ US-East (Active) ✅
    │   └─→ If fails → EU-Central (Automatic) ✅
    │
    └─→ EU-Central (Active) ✅
        └─→ If fails → US-East (Automatic) ✅
```

**Behavior:**
1. **Normal**: Requests load balance between US-East and EU-Central
2. **US fails**: NGINX detects failure → routes to EU-Central automatically
3. **EU fails**: NGINX detects failure → routes to US-East automatically
4. **Recovery**: When failed region recovers, traffic automatically returns to load balancing

## Prerequisites

1. **ArgoCD installed** on both clusters (via `argocd-install` Ansible role)
2. **Helm charts ready** in `gitops/charts/`:
   - `cockroachdb/` chart
   - `ledger-app/` chart
   - `global-lb/` chart ⭐ NEW
3. **Git repository** (or local filesystem access for ArgoCD)
4. **Both regions deployed** (ledger-app must be running in both clusters)

## Usage

### Option 1: Apply directly to ArgoCD (Local Development)

If you're running locally and ArgoCD can access the filesystem:

```bash
# Apply to US-East cluster's ArgoCD
kubectl --context k3d-dc-us apply -f cockroachdb-appset.yaml
kubectl --context k3d-dc-us apply -f ledger-app-appset.yaml
kubectl --context k3d-dc-us apply -f global-lb-appset.yaml  # ⭐ NEW

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
kubectl --context k3d-dc-us apply -f global-lb-appset.yaml
```

## What Gets Created

Each ApplicationSet automatically creates:

### CockroachDB ApplicationSet:
- `k3d-dc-us-cockroachdb` Application → Deploys CockroachDB to US cluster
- `k3d-dc-eu-cockroachdb` Application → Deploys CockroachDB to EU cluster

### Ledger App ApplicationSet:
- `k3d-dc-us-ledger-app` Application → Deploys Ledger App to US cluster
- `k3d-dc-eu-ledger-app` Application → Deploys Ledger App to EU cluster

### Global Load Balancer ApplicationSet: ⭐ NEW
- `k3d-dc-us-global-lb` Application → Deploys Global LB to US cluster
- Provides automatic failover between regions

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

### Global Load Balancer: ⭐ NEW
- **Deployed to**: US cluster (can reach both regions)
- **Primary**: US-East region
- **Backup**: EU-Central region (used if US fails)
- **Failover**: Automatic (detects failures and switches)

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
kubectl --context k3d-dc-us get application k3d-dc-us-global-lb -n argocd  # ⭐ NEW
```

### Test Automatic Failover: ⭐ NEW
```bash
# Get Global LB IP
GLOBAL_IP=$(kubectl --context k3d-dc-us get svc global-lb -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Test normal operation
curl http://$GLOBAL_IP/health

# Simulate US region failure
k3d cluster stop k3d-dc-us

# Wait for failover (10 seconds)
sleep 10

# Test - should automatically use EU region
curl http://$GLOBAL_IP/health
# ✅ Should still work - automatic failover!
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

### Issue: Global LB can't reach EU region
**Solution:** 
1. Ensure EU LoadBalancer IP is correct in ApplicationSet
2. Check network connectivity between clusters
3. Verify EU region service is accessible:
   ```bash
   kubectl --context k3d-dc-eu get svc ledger-app
   ```

### Issue: Repository not found
**Solution:** For local development, you may need to:
1. Use a Git repository instead of `file:///gitops`
2. Or configure ArgoCD to access local filesystem
3. Or mount the gitops directory into ArgoCD repo-server

## Notes

- **Local Development**: The `file:///gitops` repoURL works if ArgoCD repo-server has access to the filesystem
- **Production**: Use a Git repository URL (GitHub, GitLab, etc.)
- **Auto-sync**: All ApplicationSets have `automated: true` for automatic syncing
- **Self-healing**: `selfHeal: true` ensures manual changes are reverted
- **Automatic Failover**: Global LB provides HTTP-level failover between regions ⭐ NEW
