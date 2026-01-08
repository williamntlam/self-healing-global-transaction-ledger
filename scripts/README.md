# Chaos Engineering Scripts

This directory contains scripts for testing the resilience and self-healing capabilities of the multi-region ledger system.

## Scripts

### `blast_radius.sh`

Simulates regional outages and validates failover behavior.

**Usage:**
```bash
# Simulate outage in US-East region
./scripts/blast_radius.sh us-east-1

# Simulate outage in EU-Central region
./scripts/blast_radius.sh eu-central-1

# Default (US-East)
./scripts/blast_radius.sh
```

**What it does:**

1. **Establishes Baseline**
   - Creates test transactions in both regions
   - Verifies both regions are operational

2. **Simulates Outage**
   - Stops the target K3d cluster
   - Simulates a complete regional failure

3. **Validates Failover**
   - Checks surviving region is healthy
   - Verifies CockroachDB continues serving
   - Tests transaction creation in surviving region
   - Validates data persistence
   - Checks ArgoCD self-healing behavior

4. **Restores Cluster**
   - Starts the stopped cluster
   - Waits for pods to be ready

5. **Validates Recovery**
   - Verifies restored region is operational
   - Tests transaction creation
   - Validates data consistency

**Prerequisites:**
- `kubectl` installed and configured
- `k3d` installed
- `curl` installed
- Both clusters (`k3d-dc-us` and `k3d-dc-eu`) running
- Ledger app deployed to both clusters

**Example Output:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Starting Chaos Engineering Test
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ Prerequisites check passed
✅ Baseline established
⚠️  About to simulate outage in us-east-1
...
✅ Failover validation passed
✅ Recovery validation passed
```

## Additional Chaos Scenarios

### Pod Deletion Test
```bash
# Delete random pods to test Kubernetes self-healing
kubectl --context k3d-dc-us delete pod -l app=ledger-app --field-selector=status.phase=Running
```

### Database Node Failure
```bash
# Delete a CockroachDB pod
kubectl --context k3d-dc-us delete pod cockroachdb-0
```

### Network Partition (Advanced)
```bash
# Block network traffic between clusters
docker network disconnect k3d-dc-us k3d-dc-eu-server-0
```

## What to Validate

When running chaos tests, verify:

1. **Zero Data Loss**: All transactions persist across outages
2. **Service Continuity**: Surviving region handles all traffic
3. **Self-Healing**: ArgoCD redeploys when cluster restores
4. **Database Consistency**: CockroachDB maintains consistency
5. **Recovery Time**: How long until full recovery

## Troubleshooting

### Service Not Found
If you see "ledger-app service not found", ensure:
- The application is deployed via ArgoCD
- The service exists: `kubectl get svc ledger-app`

### Cannot Connect to Service
If health checks fail:
- Set up port-forwarding: `kubectl port-forward svc/ledger-app 8080:80`
- Update the script to use the forwarded port

### Cluster Not Found
If cluster doesn't exist:
- Check available clusters: `k3d cluster list`
- Create missing clusters using Ansible playbook

## Integration with CI/CD

You can integrate this into CI/CD pipelines:

```yaml
# .github/workflows/chaos-test.yml
name: Chaos Engineering
on:
  schedule:
    - cron: '0 2 * * *'  # Run daily at 2 AM

jobs:
  chaos-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Setup clusters
        run: |
          # Setup clusters here
      - name: Run chaos test
        run: ./scripts/blast_radius.sh us-east-1
```

## Safety Notes

⚠️ **Warning**: This script will stop Kubernetes clusters. Ensure:
- You're running in a development/test environment
- You have backups of important data
- You understand the impact of stopping clusters
- You can restore clusters if needed

## Best Practices

1. **Run in Test Environment First**: Always test in non-production
2. **Monitor During Tests**: Watch logs and metrics during chaos tests
3. **Document Results**: Record what works and what doesn't
4. **Iterate**: Improve based on test results
5. **Automate**: Run regularly to catch regressions

