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

# Create or update the ApplicationSet with discovered IPs
cat > /tmp/global-lb-appset-updated.yaml <<EOF
---
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: global-lb
  namespace: argocd
spec:
  generators:
  - list:
      elements:
      - cluster: k3d-dc-us
        url: https://kubernetes.default.svc
        usHost: "$US_IP"
        usPort: "$US_PORT"
        euHost: "$EU_IP"
        euPort: "$EU_PORT"
  
  template:
    metadata:
      name: '{{cluster}}-global-lb'
      labels:
        app: global-lb
    spec:
      project: default
      source:
        repoURL: file:///gitops
        targetRevision: HEAD
        path: charts/global-lb
        helm:
          valueFiles:
            - values.yaml
          values: |
            upstreams:
              us-east-1:
                host: {{usHost}}
                port: {{usPort}}
              eu-central-1:
                host: {{euHost}}
                port: {{euPort}}
      destination:
        server: '{{url}}'
        namespace: default
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
          - PrunePropagationPolicy=foreground
          - PruneLast=true
EOF

log_info "Apply the updated ApplicationSet:"
log_info "kubectl --context k3d-dc-us apply -f /tmp/global-lb-appset-updated.yaml"
log_info ""
log_info "Or manually update gitops/appsets/global-lb-appset.yaml with:"
log_info "  usHost: $US_IP"
log_info "  usPort: $US_PORT"
log_info "  euHost: $EU_IP"
log_info "  euPort: $EU_PORT"

log_success "Configuration ready!"

