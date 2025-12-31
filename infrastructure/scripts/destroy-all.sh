#!/bin/bash

# Script to destroy everything including LocalStack (non-interactive version)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$INFRA_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🗑️  Destroying ALL Resources (Including LocalStack)${NC}"
echo "=================================================="
echo ""

# Check if LocalStack is running, start if needed
if ! docker ps | grep -q localstack; then
    echo -e "${YELLOW}⚠️  LocalStack is not running. Starting it temporarily...${NC}"
    docker compose -f docker-compose.localstack.yml up -d
    echo "⏳ Waiting for LocalStack to be ready..."
    sleep 5
fi

# Destroy Terraform resources
if [ -f terraform.tfstate ] || [ -f .terraform/terraform.tfstate ]; then
    echo -e "${YELLOW}🔥 Destroying Terraform resources...${NC}"
    terraform destroy -auto-approve || echo -e "${RED}⚠️  Terraform destroy had issues (may be expected if state is empty)${NC}"
    echo ""
else
    echo -e "${YELLOW}ℹ️  No Terraform state found.${NC}"
    echo ""
fi

# Stop LocalStack
echo -e "${YELLOW}🛑 Stopping LocalStack containers...${NC}"
docker compose -f docker-compose.localstack.yml down
echo -e "${GREEN}✅ LocalStack stopped.${NC}"
echo ""

# Clean up state files
echo -e "${YELLOW}🧹 Cleaning up Terraform state files...${NC}"
rm -f terraform.tfstate terraform.tfstate.backup .terraform.lock.hcl
rm -rf .terraform/
echo -e "${GREEN}✅ State files removed.${NC}"
echo ""

# Optional: Clean up LocalStack data
read -p "Do you want to remove LocalStack data volumes? (y/n): " cleanup_volumes
if [ "$cleanup_volumes" = "y" ] || [ "$cleanup_volumes" = "Y" ]; then
    echo ""
    echo -e "${YELLOW}🧹 Removing LocalStack data volumes...${NC}"
    docker volume ls | grep -E "localstack|atlas" | awk '{print $2}' | xargs -r docker volume rm 2>/dev/null || true
    rm -rf ../localstack-data/
    echo -e "${GREEN}✅ LocalStack data removed.${NC}"
fi

echo ""
echo -e "${GREEN}✨ Complete cleanup finished!${NC}"
echo ""
echo -e "${YELLOW}ℹ️  To start fresh, run:${NC}"
echo "  1. docker compose -f docker-compose.localstack.yml up -d"
echo "  2. terraform init"
echo "  3. terraform apply"

