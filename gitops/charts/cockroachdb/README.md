# CockroachDB Helm Chart

This Helm chart deploys a CockroachDB cluster optimized for multi-region deployments.

## Features

- **StatefulSet** for stable network identities and persistent storage
- **Multi-region support** with locality configuration
- **Unified cluster** - connect multiple regions into one CockroachDB cluster ⭐ NEW
- **Automatic initialization** with database schema
- **Health checks** (liveness and readiness probes)
- **Pod Disruption Budget** for high availability
- **LoadBalancer service** for external access

## Installation

### Basic Installation

```bash
helm install cockroachdb . \
  --namespace default \
  --create-namespace
```

### Multi-Region Installation (Unified Cluster)

For US-East region (deploy first):
```bash
helm install cockroachdb-us . \
  --namespace default \
  --set region.name=us-east-1 \
  --set region.locality="region=us-east-1"
```

For EU-Central region (deploy after US, with US endpoint):
```bash
# Get US LoadBalancer IP first
US_IP=$(kubectl get svc cockroachdb-public -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

helm install cockroachdb-eu . \
  --namespace default \
  --set region.name=eu-central-1 \
  --set region.locality="region=eu-central-1" \
  --set region.remoteRegion=us-east-1 \
  --set region.remoteJoinAddresses="${US_IP}:26257" \
  --set region.useLoadBalancerForJoin=true
```

Then update US cluster to include EU endpoint:
```bash
# Get EU LoadBalancer IP
EU_IP=$(kubectl get svc cockroachdb-public -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

helm upgrade cockroachdb-us . \
  --set region.remoteRegion=eu-central-1 \
  --set region.remoteJoinAddresses="${EU_IP}:26257" \
  --set region.useLoadBalancerForJoin=true
```

### Using ApplicationSet (Recommended)

The ApplicationSet automatically handles multi-region deployment:

```bash
# Deploy ApplicationSet
kubectl apply -f ../../appsets/cockroachdb-appset.yaml

# Use helper script to configure join addresses
./scripts/setup-cockroachdb-cluster.sh
```

## Configuration

### Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `statefulset.replicas` | Number of CockroachDB nodes | `3` |
| `image.tag` | CockroachDB version | `"23.1.0"` |
| `storage.persistentVolume.size` | Storage size per pod | `10Gi` |
| `service.type` | Service type | `LoadBalancer` |
| `region.name` | Region identifier | `"us-east-1"` |
| `region.locality` | CockroachDB locality | `"region=us-east-1"` |
| `region.remoteRegion` | Remote region name | `""` |
| `region.remoteJoinAddresses` | Remote region join addresses | `""` |
| `region.useLoadBalancerForJoin` | Use LoadBalancer for cross-cluster | `false` |
| `conf.cache` | Cache size | `"256MiB"` |
| `conf.max-sql-memory` | Max SQL memory | `"256MiB"` |

### Example values.yaml

```yaml
statefulset:
  replicas: 3

region:
  name: "us-east-1"
  locality: "region=us-east-1"
  remoteRegion: "eu-central-1"
  remoteJoinAddresses: "192.168.1.20:26257"  # EU LoadBalancer IP
  useLoadBalancerForJoin: true

storage:
  persistentVolume:
    size: 20Gi

resources:
  requests:
    cpu: "1000m"
    memory: "1Gi"
  limits:
    cpu: "4000m"
    memory: "2Gi"
```

## Unified Cluster Architecture

When configured for multi-region, the clusters form **one unified CockroachDB cluster**:

```
┌─────────────────────────────────────────┐
│      Unified CockroachDB Cluster        │
├─────────────────────────────────────────┤
│                                         │
│  US-East Region (3 nodes)              │
│  ├─ cockroachdb-0 (locality: us-east-1)│
│  ├─ cockroachdb-1 (locality: us-east-1)│
│  └─ cockroachdb-2 (locality: us-east-1)│
│                                         │
│  EU-Central Region (3 nodes)           │
│  ├─ cockroachdb-0 (locality: eu-central-1)│
│  ├─ cockroachdb-1 (locality: eu-central-1)│
│  └─ cockroachdb-2 (locality: eu-central-1)│
│                                         │
└─────────────────────────────────────────┘
```

**Key Features:**
- **6 total nodes** (3 per region)
- **One logical cluster** - all nodes participate in Raft consensus
- **Data partitioning** - `REGIONAL BY ROW` stores data in appropriate region
- **Cross-region access** - can read/write from any region
- **Survive region failure** - `SURVIVE REGION FAILURE` ensures availability

## Accessing CockroachDB

### Via LoadBalancer Service

After installation, get the external IP:
```bash
kubectl get svc cockroachdb-public
```

Connect using the CockroachDB SQL client:
```bash
cockroach sql --insecure --host=<EXTERNAL_IP>:26257
```

### Via Port Forward

```bash
kubectl port-forward svc/cockroachdb-public 26257:26257
cockroach sql --insecure --host=localhost:26257
```

### Access Admin UI

```bash
# Get external IP
EXTERNAL_IP=$(kubectl get svc cockroachdb-public -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Open in browser
open http://$EXTERNAL_IP:8080
```

## Database Initialization

The chart includes an initialization job that:
1. Waits for CockroachDB to be ready
2. Creates the `ledger` database
3. Creates the `transactions` table with multi-region configuration
4. Sets survival goals for region failure

View the initialization script in `values.yaml` under `init.script`.

## Multi-Region Setup

### Step 1: Deploy US Cluster

```bash
helm install cockroachdb-us . \
  --set region.name=us-east-1 \
  --set region.locality="region=us-east-1"
```

### Step 2: Get US Endpoint

```bash
US_IP=$(kubectl get svc cockroachdb-public -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "US IP: $US_IP"
```

### Step 3: Deploy EU Cluster (with US endpoint)

```bash
helm install cockroachdb-eu . \
  --set region.name=eu-central-1 \
  --set region.locality="region=eu-central-1" \
  --set region.remoteRegion=us-east-1 \
  --set region.remoteJoinAddresses="${US_IP}:26257" \
  --set region.useLoadBalancerForJoin=true
```

### Step 4: Update US Cluster (with EU endpoint)

```bash
EU_IP=$(kubectl get svc cockroachdb-public -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

helm upgrade cockroachdb-us . \
  --set region.remoteRegion=eu-central-1 \
  --set region.remoteJoinAddresses="${EU_IP}:26257" \
  --set region.useLoadBalancerForJoin=true
```

### Step 5: Verify Unified Cluster

```bash
# Connect to any node
kubectl exec -it cockroachdb-0 -- cockroach sql --insecure

# Should show 6 nodes
SHOW NODES;

# Should show both regions
SHOW REGIONS;
```

## Troubleshooting

### Check Pod Status
```bash
kubectl get pods -l app.kubernetes.io/name=cockroachdb
```

### View Logs
```bash
kubectl logs cockroachdb-0
```

### Check StatefulSet
```bash
kubectl get statefulset cockroachdb
```

### Verify Storage
```bash
kubectl get pvc -l app.kubernetes.io/name=cockroachdb
```

### Test Connection
```bash
kubectl run -it --rm debug --image=cockroachdb/cockroach:23.1.0 --restart=Never -- \
  sql --insecure --host=cockroachdb-0.cockroachdb:26257 -e "SELECT 1"
```

### Verify Unified Cluster

```bash
# Check nodes (should show 6: 3 US + 3 EU)
kubectl exec -it cockroachdb-0 -- cockroach node ls --insecure

# Check regions
kubectl exec -it cockroachdb-0 -- cockroach sql --insecure -e "SHOW REGIONS"

# Check database survival goals
kubectl exec -it cockroachdb-0 -- cockroach sql --insecure -e "SHOW DATABASE ledger"
```

### Common Issues

**Issue**: Nodes can't join remote region
- **Check**: LoadBalancer service has external IP
- **Check**: `remoteJoinAddresses` is set correctly
- **Check**: Network connectivity between clusters

**Issue**: Only local nodes visible
- **Check**: `remoteJoinAddresses` includes correct IP:port
- **Check**: Both clusters have `useLoadBalancerForJoin: true`
- **Check**: CockroachDB logs for connection errors

## Resources Created

- **StatefulSet**: `cockroachdb` (or release name)
- **Service (Headless)**: `cockroachdb` - for internal communication
- **Service (LoadBalancer)**: `cockroachdb-public` - for external access
- **ServiceAccount**: `cockroachdb` (if enabled)
- **ConfigMap**: Database initialization script
- **Job**: Database initialization (if enabled)
