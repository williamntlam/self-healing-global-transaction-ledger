# Testing Guide

This document describes the testing strategy for the ledger application.

## Test Types

### 1. Unit Tests (`*_test.go` files)
Fast, isolated tests for individual components:
- **Models**: `internal/models/transaction_test.go` - Tests for data models, parsing, serialization
- **Config**: `internal/config/config_test.go` - Tests for configuration loading

**Run unit tests:**
```bash
go test ./internal/... -v
```

### 2. Integration Tests (`integration/` directory)
Tests that verify multi-region behavior and failover:
- **Multi-Region Consistency**: Tests data consistency across regions
- **Failover Scenarios**: Tests behavior when one region is down
- **Load Balancer**: Tests automatic failover via global LB

**Run integration tests:**
```bash
go test ./integration/... -v -timeout 5m
```

## What's Tested

### ✅ Unit Tests (Implemented)
- [x] `ParseAmount()` - Validates decimal parsing
- [x] `AuditLog.ToJSON()` - JSON serialization
- [x] `Transaction` JSON marshaling/unmarshaling
- [x] `UUIDArray.Value()` - Database value conversion
- [x] `LoadConfig()` - Configuration loading with defaults
- [x] `LoadSecrets()` - Secrets loading
- [x] Environment variable helpers

### ✅ Integration Tests (Implemented)
- [x] **Both regions up**: Create transaction and verify consistency
- [x] **Both regions up**: Update transaction and verify consistency
- [x] **Both regions up**: List transactions and verify consistency
- [x] **US region down**: Create via EU, verify persistence
- [x] **EU region down**: Create via US, verify persistence
- [x] **Load balancer**: Route to healthy region
- [x] **Load balancer**: Automatic failover when region down
- [x] **Load balancer**: Health check endpoint

## What's Missing (To Implement)

### 🔲 Additional Unit Tests Needed
- [ ] API handlers (with mocks for DB, S3, SQS)
- [ ] Database operations (with test database)
- [ ] S3 client (with LocalStack test instance)
- [ ] SQS client (with LocalStack test instance)

### 🔲 Additional Integration Test Scenarios
- [ ] **Concurrent transactions**: Multiple transactions from both regions simultaneously
- [ ] **Transaction updates during failover**: Update transaction while one region is down
- [ ] **Delete operations**: Delete transactions and verify consistency
- [ ] **Network partition**: Simulate network issues between regions
- [ ] **Data corruption recovery**: Verify CockroachDB handles corruption
- [ ] **Performance under load**: Load test with both regions up
- [ ] **Recovery time**: Measure time to recover after region comes back

### 🔲 End-to-End Tests
- [ ] **Full workflow**: Create → Update → Delete → Verify consistency
- [ ] **Chaos engineering integration**: Integrate with `blast_radius.sh`
- [ ] **ArgoCD self-healing**: Verify ArgoCD redeploys after failures

## Test Coverage Goals

- **Unit Tests**: 80%+ coverage for business logic
- **Integration Tests**: Cover all critical paths
- **E2E Tests**: Cover main user workflows

## Running Tests

### All Tests
```bash
go test ./... -v
```

### Unit Tests Only
```bash
go test ./internal/... -v
```

### Integration Tests Only
```bash
go test ./integration/... -v -timeout 5m
```

### With Coverage
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Skip Integration Tests (Fast)
```bash
go test ./... -short
```

## Test Prerequisites

### For Unit Tests
- Go 1.21+
- No external dependencies

### For Integration Tests
- Both K3d clusters running (`k3d-dc-us`, `k3d-dc-eu`)
- Ledger app deployed to both clusters
- CockroachDB running with replication
- Port forwarding set up (or LoadBalancer IPs)
- Global load balancer deployed

## CI/CD Integration

Add to `.github/workflows/test.yml`:

```yaml
name: Tests
on: [push, pull_request]
jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.21'
      - run: go test ./internal/... -v -coverprofile=coverage.out
      - run: go tool cover -func=coverage.out
  
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Setup clusters
        run: |
          # Setup k3d clusters, deploy apps, etc.
      - name: Run integration tests
        run: go test ./integration/... -v -timeout 10m
```

## Test Data Management

### Cleanup
Integration tests should clean up after themselves:
- Delete test transactions after verification
- Reset test state between runs

### Test Isolation
- Each test should be independent
- Use unique account names/IDs per test
- Don't rely on test execution order

## Debugging Failed Tests

### Unit Tests
```bash
# Run specific test with verbose output
go test -v ./internal/models -run TestParseAmount

# Run with race detector
go test -race ./internal/...
```

### Integration Tests
```bash
# Run specific integration test
go test -v ./integration -run TestMultiRegionConsistency

# Check logs
kubectl logs -l app=ledger-app --context k3d-dc-us
kubectl logs -l app=ledger-app --context k3d-dc-eu
```

## Best Practices

1. **Fast Unit Tests**: Unit tests should run in < 1 second total
2. **Isolated Tests**: Each test should be independent
3. **Clear Test Names**: Use descriptive test names
4. **Test Data**: Use realistic but distinct test data
5. **Error Messages**: Provide helpful error messages
6. **Cleanup**: Always clean up test data
7. **Documentation**: Document test scenarios and prerequisites
