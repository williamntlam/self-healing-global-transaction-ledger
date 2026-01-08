# ArgoCD ApplicationSets

This directory contains ArgoCD ApplicationSets that deploy applications to multiple clusters automatically.

## ApplicationSets

### 1. `cockroachdb-appset.yaml`
Deploys CockroachDB to both `dc-us` and `dc-eu` clusters with region-specific configuration.
**Now configured for unified cluster** - both regions form one CockroachDB cluster.

### 2. `ledger-app-appset.yaml`
Deploys the Ledger App to both `dc-us` and `dc-eu` clusters with region-specific AWS endpoints and configurations.

### 3. `global-lb-appset.yaml` ⭐ NEW
Deploys a Global Load Balancer that provides **automatic failover** between regions.

**Key Feature**: If one region goes down, requests automatically route to the healthy region.

## CockroachDB Unified Cluster Setup

The CockroachDB clusters in US and EU are now configured to form **one unified database cluster**.

### How It Works

1. **US Cluster**: 3 nodes with locality `region=us-east-1`
2. **EU Cluster**: 3 nodes with locality `region=eu-central-1`
3. **Unified Cluster**: All 6 nodes join together to form one CockroachDB cluster
4. **Data Partitioning**: Data is partitioned by region (`REGIONAL BY ROW`)
5. **Cross-Region Access**: Both regions can read/write all data

### Setup Steps

#### Step 1: Deploy Initial Cluster (US)

```bash
# Deploy US cluster first (without remote join addresses)
kubectl --context k3d-dc-us apply -f gitops/appsets/cockroachdb-appset.yaml
```

#### Step 2: Get US Cluster Endpoint

```bash
# Get US LoadBalancer IP
US_IP=$(kubectl --context k3d-dc-us get svc cockroachdb-public -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "US IP: $US_IP"
```

#### Step 3: Deploy EU Cluster with US Endpoint

```bash
# Update ApplicationSet with US endpoint, then deploy EU
# Or use the helper script:
./scripts/setup-cockroachdb-cluster.sh
```

#### Step 4: Update US Cluster with EU Endpoint

After EU is deployed, update US cluster to include EU endpoints:

```bash
# Get EU LoadBalancer IP
EU_IP=$(kubectl --context k3d-dc-eu get svc cockroachdb-public -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Update ApplicationSet manually or use the helper script
./scripts/setup-cockroachdb-cluster.sh
```

### Automated Setup Script

Use the helper script to automatically discover and configure endpoints:

```bash
./scripts/setup-cockroachdb-cluster.sh
```

This script:
1. Discovers LoadBalancer IPs for both clusters
2. Updates the ApplicationSet with correct join addresses
3. Provides instructions for applying changes

### Verify Unified Cluster

```bash
# Connect to US cluster
kubectl --context k3d-dc-us exec -it cockroachdb-0 -- cockroach sql --insecure

# Check cluster nodes (should show 6 nodes: 3 US + 3 EU)
SHOW CLUSTER SETTING cluster.organization;

# Check regions
SHOW REGIONS;

# Check nodes
SHOW NODES;
```

### Troubleshooting

**Issue**: Nodes can't connect to each other
- **Solution**: Ensure LoadBalancer services are created and have external IPs
- **Check**: `kubectl get svc cockroachdb-public -A`

**Issue**: Only local nodes visible
- **Solution**: Verify `remoteJoinAddresses` is set correctly in ApplicationSet
- **Check**: `kubectl get application k3d-dc-us-cockroachdb -n argocd -o yaml`

**Issue**: Cross-cluster network connectivity
- **Solution**: Ensure k3d clusters are on the same Docker network
- **Check**: `docker network ls` and verify clusters can ping each other

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
4. **Both up**: Load balances between both regions

## Applications Created

Each ApplicationSet creates ArgoCD Applications for each cluster:

### CockroachDB ApplicationSet:
- `k3d-dc-us-cockroachdb` Application → Deploys CockroachDB to US cluster
- `k3d-dc-eu-cockroachdb` Application → Deploys CockroachDB to EU cluster
- **Both clusters form one unified CockroachDB cluster** ⭐

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
- **Unified Cluster**: Both regions join to form one cluster ⭐

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
kubectl --context k3d-dc-us get application k3d-dc-eu-cockroachdb -n argocd
kubectl --context k3d-dc-us get application k3d-dc-us-ledger-app -n argocd
kubectl --context k3d-dc-us get application k3d-dc-us-global-lb -n argocd  # ⭐ NEW
```

### Verify Unified CockroachDB Cluster:
```bash
# Should show 6 nodes (3 US + 3 EU)
kubectl --context k3d-dc-us exec -it cockroachdb-0 -- cockroach node ls --insecure
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
```
