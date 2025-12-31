#!/bin/bash

# Script to destroy all provisioned resources in LocalStack

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$INFRA_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🗑️  Destroying Provisioned Resources${NC}"
echo "=================================="
echo ""

# Check if LocalStack is running
if ! docker ps | grep -q localstack; then
    echo -e "${YELLOW}⚠️  LocalStack is not running. Starting it temporarily...${NC}"
    docker compose -f docker-compose.localstack.yml up -d
    echo "⏳ Waiting for LocalStack to be ready..."
    sleep 5
    TEMP_STARTED=true
else
    TEMP_STARTED=false
fi

# Check if Terraform state exists
if [ ! -f terraform.tfstate ] && [ ! -f .terraform/terraform.tfstate ]; then
    echo -e "${YELLOW}⚠️  No Terraform state file found. Nothing to destroy.${NC}"
    exit 0
fi

# Show what will be destroyed
echo -e "${YELLOW}📋 Resources that will be destroyed:${NC}"
terraform state list 2>/dev/null | while read resource; do
    echo "  - $resource"
done || echo "  (Unable to list resources)"
echo ""

# Ask for confirmation
read -p "Are you sure you want to destroy all resources? (type 'yes' to confirm): " confirmation

if [ "$confirmation" != "yes" ]; then
    echo -e "${RED}❌ Destruction cancelled.${NC}"
    exit 1
fi

echo ""
echo -e "${YELLOW}🔥 Destroying resources...${NC}"
echo ""

# Run terraform destroy
if terraform destroy -auto-approve; then
    echo ""
    echo -e "${GREEN}✅ All resources destroyed successfully!${NC}"
    echo ""
    
    # Show what was destroyed
    echo "Destroyed resources:"
    echo "  - S3 buckets (us-east-1-audit-logs, eu-central-1-audit-logs)"
    echo "  - SQS queues (transaction queues for both regions)"
    echo "  - IAM roles (ledger-app-role for both regions)"
    echo "  - IAM policies (attached to roles)"
    echo "  - S3 bucket versioning configurations"
    echo ""
    
    # Ask about stopping LocalStack
    if [ "$TEMP_STARTED" = "false" ]; then
        read -p "Do you want to stop LocalStack containers? (y/n): " stop_localstack
        if [ "$stop_localstack" = "y" ] || [ "$stop_localstack" = "Y" ]; then
            echo ""
            echo -e "${YELLOW}🛑 Stopping LocalStack...${NC}"
            docker compose -f docker-compose.localstack.yml down
            echo -e "${GREEN}✅ LocalStack stopped.${NC}"
        fi
    else
        echo -e "${YELLOW}🛑 Stopping temporarily started LocalStack...${NC}"
        docker compose -f docker-compose.localstack.yml down
        echo -e "${GREEN}✅ LocalStack stopped.${NC}"
    fi
    
    # Ask about cleaning up state files
    echo ""
    read -p "Do you want to remove Terraform state files? (y/n): " cleanup_state
    if [ "$cleanup_state" = "y" ] || [ "$cleanup_state" = "Y" ]; then
        echo ""
        echo -e "${YELLOW}🧹 Cleaning up state files...${NC}"
        rm -f terraform.tfstate terraform.tfstate.backup .terraform.lock.hcl
        rm -rf .terraform/
        echo -e "${GREEN}✅ State files removed.${NC}"
        echo ""
        echo -e "${YELLOW}ℹ️  Note: You'll need to run 'terraform init' again before next use.${NC}"
    fi
    
else
    echo ""
    echo -e "${RED}❌ Error during destruction. Check the output above.${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✨ Cleanup complete!${NC}"

