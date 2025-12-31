#!/bin/bash

# Helper script to manage LocalStack and Terraform together

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$INFRA_DIR"

case "${1:-plan}" in
    init)
        echo "🚀 Starting LocalStack..."
        docker compose -f docker-compose.localstack.yml up -d
        
        echo "⏳ Waiting for LocalStack to be ready..."
        sleep 5
        
        echo "✅ Initializing Terraform..."
        terraform init
        ;;
    
    plan|apply|destroy)
        echo "🔍 Checking if LocalStack is running..."
        if ! docker ps | grep -q localstack; then
            echo "🚀 LocalStack not running. Starting it now..."
            docker compose -f docker-compose.localstack.yml up -d
            echo "⏳ Waiting for LocalStack to be ready..."
            sleep 5
        else
            echo "✅ LocalStack is already running"
        fi
        
        echo "🔧 Running terraform ${1}..."
        terraform "$1" "${@:2}"
        ;;
    
    stop)
        echo "🛑 Stopping LocalStack..."
        docker compose -f docker-compose.localstack.yml down
        ;;
    
    status)
        echo "📊 LocalStack Status:"
        docker ps | grep localstack || echo "❌ LocalStack is not running"
        echo ""
        echo "📊 Terraform Status:"
        if [ -f terraform.tfstate ]; then
            echo "✅ Terraform state file exists"
            terraform show 2>/dev/null | head -5 || echo "⚠️  State file exists but may be empty"
        else
            echo "❌ No Terraform state file (run 'terraform init' first)"
        fi
        ;;
    
    *)
        echo "Usage: $0 {init|plan|apply|destroy|stop|status}"
        echo ""
        echo "Commands:"
        echo "  init    - Start LocalStack and initialize Terraform"
        echo "  plan    - Start LocalStack (if needed) and run terraform plan"
        echo "  apply   - Start LocalStack (if needed) and run terraform apply"
        echo "  destroy - Start LocalStack (if needed) and run terraform destroy"
        echo "  stop    - Stop LocalStack containers"
        echo "  status  - Check LocalStack and Terraform status"
        exit 1
        ;;
esac

