# Integration Tests

This directory contains integration tests for the multi-region ledger application.

## Test Categories

### 1. Multi-Region Consistency Tests
Tests that verify data consistency across regions when both are operational:
- **Create and Verify**: Creates transactions in one region and verifies they appear in both
- **Update and Verify**: Updates transaction status and verifies consistency
- **List Consistency**: Verifies transaction lists match across regions

### 2. Failover Tests
Tests that verify system behavior when one region is down:
- **US Region Down**: Verifies EU region can handle all operations
- **EU Region Down**: Verifies US region can handle all operations
- **Data Persistence**: Verifies transactions created during outage persist when region recovers

### 3. Load Balancer Tests
Tests that verify automatic failover via the global load balancer:
- **Healthy Routing**: Verifies load balancer routes to healthy regions
- **Automatic Failover**: Verifies load balancer automatically fails over when region is down
- **Health Checks**: Verifies load balancer health endpoint

## Prerequisites

1. **Both regions running**:
   ```bash
   # US region
   kubectl --context k3d-dc-us get pods -l app=ledger-app
   
   # EU region
   kubectl --context k3d-dc-eu get pods -l app=ledger-app
   ```

2. **Port forwarding** (if not using LoadBalancer):
   ```bash
   # Terminal 1: US region
   kubectl --context k3d-dc-us port-forward svc/ledger-app 8080:80
   
   # Terminal 2: EU region
   kubectl --context k3d-dc-eu port-forward svc/ledger-app 8081:80
   
   # Terminal 3: Global LB
   kubectl --context k3d-dc-us port-forward svc/global-lb 8082:8080
   ```

3. **CockroachDB running** in both regions with replication configured

## Running Tests

### Run All Integration Tests
```bash
cd ledger-app
go test -v ./integration/... -timeout 5m
```

### Run Specific Test
```bash
go test -v ./integration/... -run TestMultiRegionConsistency
go test -v ./integration/... -run TestFailoverWithOneRegionDown
go test -v ./integration/... -run TestLoadBalancerFailover
```

### Run with Short Mode (Skip Integration Tests)
```bash
go test -short ./...
```

## Test Configuration

Update the constants in `multi_region_test.go` to match your setup:

```go
const (
    USEndpoint = "http://localhost:8080"      // US region
    EUEndpoint = "http://localhost:8081"      // EU region
    GlobalLBEndpoint = "http://localhost:8082" // Global LB
    TestTimeout = 30 * time.Second
)
```

## Test Scenarios

### Scenario 1: Both Regions Up
1. Create transaction via US region
2. Verify transaction appears in both US and EU regions
3. Update transaction status
4. Verify status update appears in both regions
5. List transactions from both regions
6. Verify lists match

### Scenario 2: US Region Down
1. Stop US region (using `blast_radius.sh us-east-1 pause`)
2. Create transaction via EU region
3. Verify transaction is created successfully
4. Restore US region
5. Verify transaction appears in US region after replication

### Scenario 3: EU Region Down
1. Stop EU region (using `blast_radius.sh eu-central-1 pause`)
2. Create transaction via US region
3. Verify transaction is created successfully
4. Restore EU region
5. Verify transaction appears in EU region after replication

### Scenario 4: Load Balancer Failover
1. Create transaction via global load balancer
2. Verify transaction is created in at least one region
3. Stop one region
4. Create another transaction via load balancer
5. Verify load balancer routes to healthy region
6. Verify transaction is created successfully

## Expected Results

### Consistency Tests
- ✅ All transactions created in one region appear in both regions
- ✅ Transaction updates are consistent across regions
- ✅ Transaction lists match across regions

### Failover Tests
- ✅ System continues operating when one region is down
- ✅ Transactions created during outage persist
- ✅ Data is replicated when region recovers

### Load Balancer Tests
- ✅ Load balancer routes to healthy regions
- ✅ Automatic failover works when region is down
- ✅ Health checks return correct status

## Troubleshooting

### Tests Fail: "Connection Refused"
- Ensure port forwarding is set up correctly
- Check that services are running: `kubectl get svc -A`

### Tests Fail: "Transaction Not Found"
- Wait longer for CockroachDB replication (increase sleep time)
- Check CockroachDB replication status
- Verify database connectivity

### Tests Fail: "Load Balancer Not Responding"
- Check global load balancer is deployed
- Verify load balancer service is running
- Check load balancer configuration

## Continuous Integration

These tests can be integrated into CI/CD pipelines:

```yaml
# .github/workflows/integration-tests.yml
name: Integration Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Setup clusters
        run: |
          # Setup k3d clusters
      - name: Run integration tests
        run: |
          go test -v ./integration/... -timeout 10m
```
