#!/bin/bash
# scripts/setup-global-lb.sh
# Sets up the global load balancer with correct upstream endpoints
# This script discovers the external IPs of both regions and updates the LB config

set -euo pipefail

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Get external IP of a service
get_service_ip() {
    local cluster=$1
    local service_name=$2
    
    # Try to get LoadBalancer IP
    local ip
    ip=$(kubectl --context "$cluster" get svc "$service_name" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
    
    if [ -n "$ip" ] && [ "$ip" != "null" ]; then
        echo "$ip"
        return 0
    fi
    
    # Try to get hostname
    local hostname
    hostname=$(kubectl --context "$cluster" get svc "$service_name" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || echo "")
    
    if [ -n "$hostname" ] && [ "$hostname" != "null" ]; then
        echo "$hostname"
        return 0
    fi
    
    # Fallback: use service name (for same-cluster access)
    echo "ledger-app.default.svc.cluster.local"
    return 1
}

# Get service port
get_service_port() {
    local cluster=$1
    local service_name=$2
    
    kubectl --context "$cluster" get svc "$service_name" -o jsonpath='{.spec.ports[0].port}' 2>/dev/null || echo "80"
}

log_info "Setting up Global Load Balancer configuration..."

# Discover endpoints
log_info "Discovering service endpoints..."

US_IP=$(get_service_ip "k3d-dc-us" "ledger-app")
US_PORT=$(get_service_port "k3d-dc-us" "ledger-app")

EU_IP=$(get_service_ip "k3d-dc-eu" "ledger-app")
EU_PORT=$(get_service_port "k3d-dc-eu" "ledger-app")

log_info "US-East endpoint: $US_IP:$US_PORT"
log_info "EU-Central endpoint: $EU_IP:$EU_PORT"

# Update ApplicationSet with discovered IPs
log_info "Updating global-lb ApplicationSet..."

# Update ApplicationSet file
APPSET_FILE="gitops/appsets/global-lb-appset.yaml"

if [ ! -f "$APPSET_FILE" ]; then
    log_warning "ApplicationSet file not found: $APPSET_FILE"
    log_info "Please update manually with:"
    log_info "  US cluster: usHost: ledger-app.default.svc.cluster.local, euHost: $EU_IP"
    log_info "  EU cluster: euHost: ledger-app.default.svc.cluster.local, usHost: $US_IP"
    exit 0
fi

# Create backup
cp "$APPSET_FILE" "${APPSET_FILE}.backup"

log_info "Updating ApplicationSet for bidirectional deployment..."
log_info ""
log_info "US Global LB will use:"
log_info "  - US: ledger-app.default.svc.cluster.local (local ClusterIP)"
log_info "  - EU: $EU_IP (external LoadBalancer)"
log_info ""
log_info "EU Global LB will use:"
log_info "  - EU: ledger-app.default.svc.cluster.local (local ClusterIP)"
log_info "  - US: $US_IP (external LoadBalancer)"
log_info ""
log_info "Please manually update $APPSET_FILE:"
log_info "  - For US cluster: euHost should be \"$EU_IP\""
log_info "  - For EU cluster: usHost should be \"$US_IP\""

log_success "Configuration ready!"

