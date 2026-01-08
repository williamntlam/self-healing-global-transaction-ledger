# Global Load Balancer Helm Chart

This Helm chart deploys an NGINX-based global load balancer that provides automatic failover between regions.

## Features

- **Automatic Failover**: Routes to healthy region automatically
- **Health Checks**: Monitors upstream health and fails over on errors
- **Zero Downtime**: Seamless failover when region goes down
- **Load Balancing**: Distributes traffic across regions (when both healthy)

## How It Works

```
Client Request
    ↓
Global Load Balancer (NGINX)
    ↓
    ├─→ US-East Region (Active)
    │   └─→ If fails → EU-Central (Automatic)
    │
    └─→ EU-Central Region (Active)
        └─→ If fails → US-East (Automatic)
```

### Failover Behavior (Bidirectional)

**Active-Active Configuration**: Both regions are active. NGINX load balances between them and automatically fails over if one is down.

1. **Normal Operation**: 
   - Requests are load balanced between US-East and EU-Central
   - Both regions handle traffic simultaneously

2. **US-East Fails**: 
   - NGINX detects failure (max_fails)
   - Automatically routes all traffic to EU-Central
   - No manual intervention needed

3. **EU-Central Fails**: 
   - NGINX detects failure (max_fails)
   - Automatically routes all traffic to US-East
   - No manual intervention needed

4. **Recovery**: 
   - After fail_timeout, NGINX retries the failed region
   - If healthy, traffic automatically returns to load balancing
   - Seamless recovery without downtime

## Installation

### Step 1: Discover Service Endpoints

First, get the external IPs of both regions:

```bash
# Get US-East LoadBalancer IP
kubectl --context k3d-dc-us get svc ledger-app -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

# Get EU-Central LoadBalancer IP
kubectl --context k3d-dc-eu get svc ledger-app -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

Or use the helper script:

```bash
./scripts/setup-global-lb.sh
```

### Step 2: Update ApplicationSet

Update `gitops/appsets/global-lb-appset.yaml` with the discovered IPs:

```yaml
euHost: "192.168.1.20"  # EU LoadBalancer IP
```

### Step 3: Deploy

```bash
# Apply ApplicationSet
kubectl --context k3d-dc-us apply -f gitops/appsets/global-lb-appset.yaml
```

### Step 4: Get Global LB Endpoint

```bash
# Get Global Load Balancer IP
kubectl --context k3d-dc-us get svc global-lb -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

## Usage

Once deployed, clients use the global load balancer endpoint:

```bash
# Use global LB instead of region-specific endpoints
GLOBAL_LB_IP=$(kubectl --context k3d-dc-us get svc global-lb -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Create transaction (automatically routes to healthy region)
curl -X POST http://$GLOBAL_LB_IP/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "from_account": "acc1",
    "to_account": "acc2",
    "amount": "100.50"
  }'
```

## Configuration

### Upstream Configuration

```yaml
upstreams:
  us-east-1:
    host: "ledger-app.default.svc.cluster.local"  # Same cluster
    port: 80
    maxFails: 3          # Fail after 3 errors
    failTimeout: 10s     # Retry after 10 seconds
  eu-central-1:
    host: "192.168.1.20"  # External IP (cross-cluster)
    port: 80
    maxFails: 3
    failTimeout: 10s
```

### Failover Behavior

- **max_fails**: Number of failed requests before marking upstream as down (default: 3)
- **fail_timeout**: Time before retrying a failed upstream (default: 10s)
- **Active-Active**: Both regions are active - load balances between them
- **Bidirectional Failover**: 
  - If US fails → automatically uses EU
  - If EU fails → automatically uses US
  - If both healthy → load balances between them

## Testing Failover

### Test 1: Normal Operation

```bash
# Create transaction via global LB
curl -X POST http://$GLOBAL_LB_IP/transactions ...

# Check which region handled it (via X-Region header or logs)
```

### Test 2: Simulate US Region Failure

```bash
# Stop US cluster
k3d cluster stop k3d-dc-us

# Wait a few seconds for failover
sleep 10

# Create transaction (should route to EU automatically)
curl -X POST http://$GLOBAL_LB_IP/transactions ...
# ✅ Should work - automatically using EU region
```

### Test 3: Restore US Region

```bash
# Start US cluster
k3d cluster start k3d-dc-us

# Wait for recovery
sleep 30

# Create transaction (should return to US)
curl -X POST http://$GLOBAL_LB_IP/transactions ...
# ✅ Should work - back to US region
```

## Architecture

```
┌─────────────────────────────────────────┐
│         Global Load Balancer            │
│         (NGINX in k3d-dc-us)           │
│      Active-Active Configuration        │
└──────────────┬──────────────────────────┘
               │
       ┌───────┴────────┐
       │                │
       ▼                ▼
┌─────────────┐  ┌─────────────┐
│  US-East    │  │ EU-Central  │
│  (Active)   │  │  (Active)   │
│             │  │             │
│ ledger-app  │  │ ledger-app  │
│             │  │             │
│ If fails →  │  │ If fails →  │
│   EU        │  │   US        │
└──────┬──────┘  └──────┬──────┘
       │                │
       └────────┬───────┘
                ▼
         ┌──────────────┐
         │  CockroachDB │
         │ (Multi-Region)│
         └──────────────┘
```

**Key Points:**
- Both regions are **active** (not primary/backup)
- Load balances between both when healthy
- **Bidirectional failover**: US→EU or EU→US automatically
- Seamless recovery when failed region comes back

## Troubleshooting

### Global LB can't reach EU region

**Problem**: EU region uses external IP that's not accessible

**Solution**: 
1. Ensure both clusters are on same Docker network
2. Use `host.docker.internal` if needed
3. Or use port-forwarding for testing

### Failover not working

**Check NGINX logs**:
```bash
kubectl --context k3d-dc-us logs -l app=global-lb
```

**Check upstream status**:
```bash
# Check if upstreams are reachable
kubectl --context k3d-dc-us exec -it deployment/global-lb -- \
  curl -v http://ledger-app.default.svc.cluster.local/health
```

### Service discovery

For production, consider:
- Using DNS names instead of IPs
- Service mesh (Istio/Linkerd) for automatic discovery
- External load balancer (AWS ALB, GCP LB)

## Production Considerations

1. **DNS**: Use DNS names instead of IPs
2. **Health Checks**: More sophisticated health checking
3. **Metrics**: Add Prometheus metrics
4. **SSL/TLS**: Add HTTPS termination
5. **Rate Limiting**: Add rate limiting per region
6. **Geographic Routing**: Route based on client location

