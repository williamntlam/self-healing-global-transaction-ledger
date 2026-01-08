#!/bin/bash
# scripts/setup-cockroachdb-cluster.sh
# Discovers CockroachDB LoadBalancer IPs and updates ApplicationSet for unified cluster
# This connects the US and EU CockroachDB clusters into one unified database

set -euo pipefail

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
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

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Get CockroachDB service LoadBalancer IP
get_cockroachdb_ip() {
    local cluster=$1
    local service_name="cockroachdb-public"
    
    # Try to get LoadBalancer IP
    local ip
    ip=$(kubectl --context "$cluster" get svc "$service_name" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
    
    if [ -n "$ip" ] && [ "$ip" != "null" ]; then
        echo "$ip"
        return 0
    fi
    
    log_warning "LoadBalancer IP not found for $service_name in $cluster"
    return 1
}

# Get CockroachDB pod names and generate join addresses
get_cockroachdb_join_addresses() {
    local cluster=$1
    local replicas=$2
    local use_lb=$3
    
    if [ "$use_lb" = "true" ]; then
        # Use LoadBalancer service IP
        local lb_ip
        lb_ip=$(get_cockroachdb_ip "$cluster")
        if [ -n "$lb_ip" ]; then
            # For LoadBalancer, we use the service IP with port
            # CockroachDB will route to the appropriate node
            echo "${lb_ip}:26257"
            return 0
        fi
    fi
    
    # Fallback: use service name (for same-cluster)
    local service_name="cockroachdb"
    local namespace="default"
    local addresses=()
    
    for i in $(seq 0 $((replicas - 1))); do
        addresses+=("${service_name}-${i}.${service_name}.${namespace}.svc.cluster.local:26257}")
    done
    
    echo "${addresses[*]}" | tr ' ' ','
}

log_info "Setting up unified CockroachDB cluster..."

# Check prerequisites
if ! command -v kubectl &> /dev/null; then
    log_error "kubectl is not installed"
    exit 1
fi

# Check clusters exist
if ! kubectl --context k3d-dc-us get nodes &> /dev/null; then
    log_error "k3d-dc-us cluster not found"
    exit 1
fi

if ! kubectl --context k3d-dc-eu get nodes &> /dev/null; then
    log_error "k3d-dc-eu cluster not found"
    exit 1
fi

# Get replicas (default to 3)
REPLICAS=${REPLICAS:-3}

log_info "Discovering CockroachDB endpoints..."

# Get US cluster endpoint
US_ENDPOINT=$(get_cockroachdb_join_addresses "k3d-dc-us" "$REPLICAS" "true")
if [ -z "$US_ENDPOINT" ]; then
    log_warning "US cluster endpoint not found, using service name fallback"
    US_ENDPOINT="cockroachdb-0.cockroachdb.default.svc.cluster.local:26257,cockroachdb-1.cockroachdb.default.svc.cluster.local:26257,cockroachdb-2.cockroachdb.default.svc.cluster.local:26257"
fi

# Get EU cluster endpoint
EU_ENDPOINT=$(get_cockroachdb_join_addresses "k3d-dc-eu" "$REPLICAS" "true")
if [ -z "$EU_ENDPOINT" ]; then
    log_warning "EU cluster endpoint not found, using service name fallback"
    EU_ENDPOINT="cockroachdb-0.cockroachdb.default.svc.cluster.local:26257,cockroachdb-1.cockroachdb.default.svc.cluster.local:26257,cockroachdb-2.cockroachdb.default.svc.cluster.local:26257"
fi

log_info "US-East endpoint: $US_ENDPOINT"
log_info "EU-Central endpoint: $EU_ENDPOINT"

# Update ApplicationSet
APPSET_FILE="gitops/appsets/cockroachdb-appset.yaml"

if [ ! -f "$APPSET_FILE" ]; then
    log_error "ApplicationSet file not found: $APPSET_FILE"
    exit 1
fi

log_info "Updating ApplicationSet with discovered endpoints..."

# Create backup
cp "$APPSET_FILE" "${APPSET_FILE}.backup"

# Update US cluster remoteJoinAddresses with EU endpoint
sed -i.tmp "s|remoteJoinAddresses: \"\"|remoteJoinAddresses: \"${EU_ENDPOINT}\"|g" "$APPSET_FILE"

# Update EU cluster remoteJoinAddresses with US endpoint  
# This is trickier - we need to update the second occurrence
# For now, we'll use a more specific pattern
sed -i.tmp "s|region: eu-central-1|region: eu-central-1|g" "$APPSET_FILE"
# Manually update the EU section
python3 << EOF
import re

with open('$APPSET_FILE', 'r') as f:
    content = f.read()

# Update EU cluster remoteJoinAddresses
pattern = r'(region: eu-central-1[^\n]*\n[^\n]*\n[^\n]*\n[^\n]*remoteJoinAddresses: )\"\"'
replacement = r'\1"${US_ENDPOINT}"'
content = re.sub(pattern, replacement, content)

with open('$APPSET_FILE', 'w') as f:
    f.write(content)
EOF

# Clean up temp files
rm -f "${APPSET_FILE}.tmp"

log_success "ApplicationSet updated successfully"
log_info "Next steps:"
log_info "1. Review the updated ApplicationSet: $APPSET_FILE"
log_info "2. Apply the updated ApplicationSet:"
log_info "   kubectl --context k3d-dc-us apply -f $APPSET_FILE"
log_info "3. ArgoCD will sync the changes automatically"
log_info "4. Verify cluster connection:"
log_info "   kubectl --context k3d-dc-us exec -it cockroachdb-0 -- cockroach sql --insecure -e 'SHOW CLUSTER SETTING cluster.organization'"
