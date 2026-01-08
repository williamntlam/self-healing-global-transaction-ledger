#!/bin/bash
# scripts/blast_radius.sh
# Simulates regional outages and validates failover behavior
#
# Usage:
#   ./scripts/blast_radius.sh [region]
#   ./scripts/blast_radius.sh us-east-1
#   ./scripts/blast_radius.sh eu-central-1

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
REGION=${1:-us-east-1}
REGION_PREFIX="${REGION%%-*}"  # Extract "us" or "eu"
CLUSTER_NAME="k3d-dc-${REGION_PREFIX}"

# Determine surviving region
if [ "$REGION_PREFIX" = "us" ]; then
    SURVIVING_REGION="eu-central-1"
    SURVIVING_PREFIX="eu"
else
    SURVIVING_REGION="us-east-1"
    SURVIVING_PREFIX="us"
fi
SURVIVING_CLUSTER="k3d-dc-${SURVIVING_PREFIX}"

# Test transaction IDs (stored for validation)
US_TX_ID=""
EU_TX_ID=""
FAILOVER_TX_ID=""

# Functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_step() {
    echo -e "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

# Check prerequisites
check_prerequisites() {
    log_step "Checking Prerequisites"
    
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi
    
    if ! command -v k3d &> /dev/null; then
        log_error "k3d is not installed"
        exit 1
    fi
    
    if ! command -v curl &> /dev/null; then
        log_error "curl is not installed"
        exit 1
    fi
    
    # Verify both clusters exist
    if ! k3d cluster list | grep -q "$CLUSTER_NAME"; then
        log_error "Cluster $CLUSTER_NAME does not exist"
        log_info "Available clusters:"
        k3d cluster list
        exit 1
    fi
    
    if ! k3d cluster list | grep -q "$SURVIVING_CLUSTER"; then
        log_error "Surviving cluster $SURVIVING_CLUSTER does not exist"
        log_info "Available clusters:"
        k3d cluster list
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Get application endpoint
get_app_endpoint() {
    local cluster=$1
    local endpoint
    
    # Try to get LoadBalancer IP
    endpoint=$(kubectl --context "$cluster" get svc ledger-app -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
    
    # If no LoadBalancer IP, try to get via port-forward or use localhost
    if [ -z "$endpoint" ] || [ "$endpoint" = "null" ]; then
        # Check if service exists
        if kubectl --context "$cluster" get svc ledger-app &> /dev/null; then
            endpoint="localhost"
        else
            log_warning "Service ledger-app not found in $cluster"
            echo ""
            return 1
        fi
    fi
    
    echo "$endpoint"
}

# Get service port
get_app_port() {
    local cluster=$1
    kubectl --context "$cluster" get svc ledger-app -o jsonpath='{.spec.ports[0].port}' 2>/dev/null || echo "80"
}

# Health check function
check_health() {
    local cluster=$1
    local endpoint=$2
    local port=$3
    
    log_info "Checking health for $cluster at $endpoint:$port"
    
    if curl -sf --max-time 5 "http://$endpoint:$port/health" > /dev/null 2>&1; then
        log_success "$cluster is healthy"
        return 0
    else
        log_error "$cluster is unhealthy or unreachable"
        return 1
    fi
}

# Create test transaction
create_test_transaction() {
    local cluster=$1
    local endpoint=$2
    local port=$3
    local region_name=$4
    
    log_info "Creating test transaction on $cluster (region: $region_name)..."
    
    local response
    response=$(curl -s --max-time 10 -X POST "http://$endpoint:$port/transactions" \
        -H "Content-Type: application/json" \
        -d "{
            \"from_account\": \"test-account-$(date +%s)\",
            \"to_account\": \"test-account-recipient\",
            \"amount\": \"$(shuf -i 100-999 -n 1).$(shuf -i 10-99 -n 1)\"
        }" 2>&1)
    
    if echo "$response" | grep -q '"id"'; then
        local tx_id
        tx_id=$(echo "$response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "")
        if [ -n "$tx_id" ]; then
            log_success "Transaction created successfully: $tx_id"
            echo "$tx_id"
            return 0
        fi
    fi
    
    log_error "Failed to create transaction"
    log_info "Response: $response"
    return 1
}

# Verify transaction exists
verify_transaction() {
    local cluster=$1
    local endpoint=$2
    local transaction_id=$3
    local port=$4
    
    log_info "Verifying transaction $transaction_id on $cluster..."
    
    local response
    response=$(curl -s --max-time 10 "http://$endpoint:$port/transactions/$transaction_id" 2>&1)
    
    if echo "$response" | grep -q "$transaction_id"; then
        log_success "Transaction verified: $transaction_id"
        return 0
    else
        log_error "Transaction not found: $transaction_id"
        log_info "Response: $response"
        return 1
    fi
}

# Check CockroachDB status
check_cockroachdb() {
    local cluster=$1
    
    log_info "Checking CockroachDB status in $cluster..."
    
    # Check if CockroachDB pods exist
    local pods
    pods=$(kubectl --context "$cluster" get pods -l app=cockroachdb -o name 2>/dev/null | wc -l || echo "0")
    
    if [ "$pods" -gt 0 ]; then
        log_success "CockroachDB has $pods pod(s) running"
        
        # Check if at least one pod is ready
        local ready_pods
        ready_pods=$(kubectl --context "$cluster" get pods -l app=cockroachdb -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -o "True" | wc -l || echo "0")
        
        if [ "$ready_pods" -gt 0 ]; then
            log_success "$ready_pods CockroachDB pod(s) are ready"
            return 0
        else
            log_warning "CockroachDB pods exist but none are ready yet"
            return 1
        fi
    else
        log_error "CockroachDB is not running (no pods found)"
        return 1
    fi
}

# Check ArgoCD applications
check_argocd_apps() {
    local cluster=$1
    
    log_info "Checking ArgoCD applications in $cluster..."
    
    if ! kubectl --context "$cluster" get namespace argocd &> /dev/null; then
        log_warning "ArgoCD namespace not found in $cluster"
        return 0
    fi
    
    local apps
    apps=$(kubectl --context "$cluster" get applications -n argocd -o json 2>/dev/null || echo "{}")
    
    if [ "$apps" != "{}" ] && [ -n "$apps" ]; then
        echo "$apps" | grep -o '"name":"[^"]*"' | cut -d'"' -f4 | while read -r app_name; do
            local status
            status=$(kubectl --context "$cluster" get application "$app_name" -n argocd -o jsonpath='{.status.sync.status},{.status.health.status}' 2>/dev/null || echo "Unknown,Unknown")
            
            if echo "$status" | grep -q "Synced,Healthy"; then
                log_success "ArgoCD app '$app_name': $status"
            else
                log_warning "ArgoCD app '$app_name': $status"
            fi
        done
    else
        log_warning "No ArgoCD applications found"
    fi
}

# Pre-chaos baseline
establish_baseline() {
    log_step "Establishing Baseline"
    
    log_info "Creating baseline transactions in both regions..."
    
    # Get endpoints
    local us_endpoint us_port eu_endpoint eu_port
    
    if kubectl --context k3d-dc-us get svc ledger-app &> /dev/null; then
        us_endpoint=$(get_app_endpoint "k3d-dc-us")
        us_port=$(get_app_port "k3d-dc-us")
        
        if [ -n "$us_endpoint" ]; then
            log_info "Creating transaction in US-East region..."
            US_TX_ID=$(create_test_transaction "k3d-dc-us" "$us_endpoint" "$us_port" "us-east-1" || echo "")
            if [ -n "$US_TX_ID" ]; then
                log_success "US baseline transaction: $US_TX_ID"
            fi
        fi
    else
        log_warning "ledger-app service not found in k3d-dc-us"
    fi
    
    if kubectl --context k3d-dc-eu get svc ledger-app &> /dev/null; then
        eu_endpoint=$(get_app_endpoint "k3d-dc-eu")
        eu_port=$(get_app_port "k3d-dc-eu")
        
        if [ -n "$eu_endpoint" ]; then
            log_info "Creating transaction in EU-Central region..."
            EU_TX_ID=$(create_test_transaction "k3d-dc-eu" "$eu_endpoint" "$eu_port" "eu-central-1" || echo "")
            if [ -n "$EU_TX_ID" ]; then
                log_success "EU baseline transaction: $EU_TX_ID"
            fi
        fi
    else
        log_warning "ledger-app service not found in k3d-dc-eu"
    fi
    
    log_success "Baseline established"
}

# Simulate outage
simulate_outage() {
    log_step "Simulating Outage in $REGION"
    
    log_warning "Stopping cluster: $CLUSTER_NAME"
    
    if k3d cluster stop "$CLUSTER_NAME" 2>/dev/null; then
        log_success "Cluster $CLUSTER_NAME stopped"
    else
        log_error "Failed to stop cluster $CLUSTER_NAME"
        log_info "Cluster may already be stopped"
    fi
    
    log_info "Waiting 10 seconds for system to detect outage..."
    sleep 10
}

# Validate failover
validate_failover() {
    log_step "Validating Failover"
    
    local surviving_endpoint surviving_port
    
    if ! kubectl --context "$SURVIVING_CLUSTER" get svc ledger-app &> /dev/null; then
        log_error "ledger-app service not found in $SURVIVING_CLUSTER"
        log_warning "Skipping failover validation"
        return 1
    fi
    
    surviving_endpoint=$(get_app_endpoint "$SURVIVING_CLUSTER")
    surviving_port=$(get_app_port "$SURVIVING_CLUSTER")
    
    if [ -z "$surviving_endpoint" ]; then
        log_error "Could not determine endpoint for $SURVIVING_CLUSTER"
        return 1
    fi
    
    # 1. Check surviving region is healthy
    log_info "Step 1: Check surviving region health"
    if check_health "$SURVIVING_CLUSTER" "$surviving_endpoint" "$surviving_port"; then
        log_success "Surviving region is operational"
    else
        log_error "Surviving region is not operational"
        log_warning "You may need to set up port-forwarding:"
        log_info "kubectl --context $SURVIVING_CLUSTER port-forward svc/ledger-app 8080:80"
        return 1
    fi
    
    # 2. Check CockroachDB in surviving region
    log_info "Step 2: Check CockroachDB in surviving region"
    if check_cockroachdb "$SURVIVING_CLUSTER"; then
        log_success "CockroachDB is operational"
    else
        log_warning "CockroachDB check failed (may still be starting)"
    fi
    
    # 3. Create transaction in surviving region
    log_info "Step 3: Test transaction creation in surviving region"
    FAILOVER_TX_ID=$(create_test_transaction "$SURVIVING_CLUSTER" "$surviving_endpoint" "$surviving_port" "$SURVIVING_REGION" || echo "")
    
    if [ -n "$FAILOVER_TX_ID" ]; then
        log_success "Transaction created during outage: $FAILOVER_TX_ID"
        
        # 4. Verify transaction persists
        log_info "Step 4: Verify transaction persistence"
        if verify_transaction "$SURVIVING_CLUSTER" "$surviving_endpoint" "$FAILOVER_TX_ID" "$surviving_port"; then
            log_success "Transaction persisted correctly"
        else
            log_error "Transaction not found after creation"
            return 1
        fi
    else
        log_error "Failed to create transaction during outage"
        log_warning "This may indicate a problem with failover"
        return 1
    fi
    
    # 5. Check ArgoCD self-healing
    log_info "Step 5: Check ArgoCD self-healing"
    check_argocd_apps "$SURVIVING_CLUSTER"
    
    log_success "Failover validation complete"
}

# Restore cluster
restore_cluster() {
    log_step "Restoring Cluster"
    
    log_info "Starting cluster: $CLUSTER_NAME"
    
    if k3d cluster start "$CLUSTER_NAME" 2>/dev/null; then
        log_success "Cluster $CLUSTER_NAME started"
    else
        log_error "Failed to start cluster $CLUSTER_NAME"
        return 1
    fi
    
    log_info "Waiting 30 seconds for cluster to be ready..."
    sleep 30
    
    # Wait for pods to be ready (with timeout)
    log_info "Waiting for pods to be ready..."
    if kubectl --context "$CLUSTER_NAME" wait --for=condition=ready pod -l app=ledger-app --timeout=120s 2>/dev/null; then
        log_success "Pods are ready"
    else
        log_warning "Some pods may not be ready yet (this is okay)"
    fi
    
    log_success "Cluster $CLUSTER_NAME restored"
}

# Validate recovery
validate_recovery() {
    log_step "Validating Recovery"
    
    local restored_endpoint restored_port
    
    if ! kubectl --context "$CLUSTER_NAME" get svc ledger-app &> /dev/null; then
        log_error "ledger-app service not found in $CLUSTER_NAME"
        log_warning "Skipping recovery validation"
        return 1
    fi
    
    restored_endpoint=$(get_app_endpoint "$CLUSTER_NAME")
    restored_port=$(get_app_port "$CLUSTER_NAME")
    
    if [ -z "$restored_endpoint" ]; then
        log_error "Could not determine endpoint for $CLUSTER_NAME"
        return 1
    fi
    
    # 1. Check restored region is healthy
    log_info "Step 1: Check restored region health"
    if check_health "$CLUSTER_NAME" "$restored_endpoint" "$restored_port"; then
        log_success "Restored region is operational"
    else
        log_error "Restored region is not operational"
        log_warning "You may need to set up port-forwarding:"
        log_info "kubectl --context $CLUSTER_NAME port-forward svc/ledger-app 8080:80"
        return 1
    fi
    
    # 2. Create transaction in restored region
    log_info "Step 2: Test transaction creation in restored region"
    local recovery_tx_id
    recovery_tx_id=$(create_test_transaction "$CLUSTER_NAME" "$restored_endpoint" "$restored_port" "$REGION" || echo "")
    
    if [ -n "$recovery_tx_id" ]; then
        log_success "Transaction created after recovery: $recovery_tx_id"
    else
        log_error "Failed to create transaction after recovery"
        return 1
    fi
    
    # 3. Verify data consistency
    log_info "Step 3: Verify data consistency"
    if [ -n "$FAILOVER_TX_ID" ]; then
        log_info "Checking if failover transaction is visible in restored region..."
        if verify_transaction "$CLUSTER_NAME" "$restored_endpoint" "$FAILOVER_TX_ID" "$restored_port"; then
            log_success "Failover transaction is visible in restored region (data consistency verified)"
        else
            log_warning "Failover transaction not immediately visible (may need time to sync)"
        fi
    fi
    
    log_success "Recovery validation complete"
}

# Print summary
print_summary() {
    log_step "Test Summary"
    
    echo "Target Region: $REGION"
    echo "Target Cluster: $CLUSTER_NAME"
    echo "Surviving Cluster: $SURVIVING_CLUSTER"
    echo ""
    echo "Baseline Transactions:"
    [ -n "$US_TX_ID" ] && echo "  US-East: $US_TX_ID"
    [ -n "$EU_TX_ID" ] && echo "  EU-Central: $EU_TX_ID"
    echo ""
    [ -n "$FAILOVER_TX_ID" ] && echo "Failover Transaction: $FAILOVER_TX_ID"
    echo ""
    log_success "Chaos engineering test completed!"
}

# Main execution
main() {
    log_step "Starting Chaos Engineering Test"
    
    log_info "Configuration:"
    log_info "  Target Region: $REGION"
    log_info "  Target Cluster: $CLUSTER_NAME"
    log_info "  Surviving Cluster: $SURVIVING_CLUSTER"
    echo ""
    
    check_prerequisites
    echo ""
    
    establish_baseline
    echo ""
    
    log_warning "⚠️  About to simulate outage in $REGION"
    log_warning "Press Enter to continue, or Ctrl+C to cancel..."
    read -r
    
    simulate_outage
    echo ""
    
    if validate_failover; then
        log_success "Failover validation passed"
    else
        log_error "Failover validation had issues (check logs above)"
    fi
    echo ""
    
    log_warning "⚠️  Ready to restore cluster $CLUSTER_NAME"
    log_warning "Press Enter to restore, or Ctrl+C to keep it down..."
    read -r
    
    if restore_cluster; then
        echo ""
        if validate_recovery; then
            log_success "Recovery validation passed"
        else
            log_error "Recovery validation had issues (check logs above)"
        fi
    else
        log_error "Failed to restore cluster"
    fi
    echo ""
    
    print_summary
}

# Run main function
main "$@"

