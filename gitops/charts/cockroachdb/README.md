# CockroachDB Helm Chart

This Helm chart deploys a CockroachDB cluster optimized for multi-region deployments.

## Features

- **StatefulSet** for stable network identities and persistent storage
- **Multi-region support** with locality configuration
- **Automatic initialization** with database schema
- **Health checks** (liveness and readiness probes)
- **Pod Disruption Budget** for high availability
- **LoadBalancer service** for external access
- **Resource limits** and security contexts

## Installation

### Basic Installation

```bash
helm install cockroachdb . \
  --namespace default \
  --create-namespace
```

### Multi-Region Installation

For US-East region:
```bash
helm install cockroachdb-us . \
  --namespace default \
  --set region.name=us-east-1 \
  --set region.locality="region=us-east-1"
```

For EU-Central region:
```bash
helm install cockroachdb-eu . \
  --namespace default \
  --set region.name=eu-central-1 \
  --set region.locality="region=eu-central-1"
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
| `conf.cache` | Cache size | `"256MiB"` |
| `conf.max-sql-memory` | Max SQL memory | `"256MiB"` |

### Example values.yaml

```yaml
statefulset:
  replicas: 3

region:
  name: "us-east-1"
  locality: "region=us-east-1"

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

For multi-region deployments:

1. **Deploy to US-East:**
```bash
helm install cockroachdb-us . \
  --set region.name=us-east-1 \
  --set region.locality="region=us-east-1"
```

2. **Deploy to EU-Central:**
```bash
helm install cockroachdb-eu . \
  --set region.name=eu-central-1 \
  --set region.locality="region=eu-central-1"
```

3. **Join clusters** (if needed):
```sql
-- Run on one cluster to join the other
ALTER DATABASE ledger ADD REGION "eu-central-1";
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

## Resources Created

- **StatefulSet**: `cockroachdb` (or release name)
- **Service (Headless)**: `cockroachdb` - for internal communication
- **Service (LoadBalancer)**: `cockroachdb-public` - for external access
- **ServiceAccount**: `cockroachdb` (if enabled)
- **ConfigMap**: `cockroachdb-init` - initialization SQL script
- **Job**: `cockroachdb-init` - runs initialization (if enabled)
- **PodDisruptionBudget**: `cockroachdb` (if enabled)
- **PersistentVolumeClaims**: One per replica (auto-created)

## Security Notes

- Currently uses `--insecure` flag for simplicity
- In production, enable TLS and authentication
- Use Kubernetes secrets for credentials
- Implement network policies for pod-to-pod communication

## References

- [CockroachDB Documentation](https://www.cockroachlabs.com/docs/)
- [Multi-Region Overview](https://www.cockroachlabs.com/docs/stable/multiregion-overview.html)
- [Kubernetes Deployment Guide](https://www.cockroachlabs.com/docs/stable/deploy-cockroachdb-with-kubernetes.html)

