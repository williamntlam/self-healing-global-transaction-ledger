#!/bin/bash

# Script to verify LocalStack resources without AWS CLI

echo "🔍 Verifying LocalStack Resources"
echo "================================"
echo ""

# Method 1: Terraform Output
echo "📋 1. Terraform Output:"
echo "----------------------"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$INFRA_DIR"
terraform output 2>/dev/null || echo "No terraform output available"
echo ""

# Method 2: Check LocalStack Health
echo "🏥 2. LocalStack Health Status:"
echo "-------------------------------"
echo "US-East (port 4566):"
curl -s http://localhost:4566/_localstack/health | python3 -m json.tool 2>/dev/null | grep -E "(s3|sqs|iam)" || curl -s http://localhost:4566/_localstack/health | grep -o '"s3":"[^"]*"'
echo ""
echo "EU-Central (port 4567):"
curl -s http://localhost:4567/_localstack/health | python3 -m json.tool 2>/dev/null | grep -E "(s3|sqs|iam)" || curl -s http://localhost:4567/_localstack/health | grep -o '"s3":"[^"]*"'
echo ""

# Method 3: List S3 Buckets using curl
echo "🪣 3. S3 Buckets:"
echo "----------------"
echo "US-East buckets:"
curl -s -X GET "http://localhost:4566/" -H "Host: s3.localhost.localstack.cloud:4566" 2>/dev/null | \
    grep -o '<Name>[^<]*</Name>' | sed 's/<Name>//;s/<\/Name>//' | \
    while read bucket; do echo "  ✓ $bucket"; done || echo "  (No buckets found or error)"
echo ""
echo "EU-Central buckets:"
curl -s -X GET "http://localhost:4567/" -H "Host: s3.localhost.localstack.cloud:4567" 2>/dev/null | \
    grep -o '<Name>[^<]*</Name>' | sed 's/<Name>//;s/<\/Name>//' | \
    while read bucket; do echo "  ✓ $bucket"; done || echo "  (No buckets found or error)"
echo ""

# Method 4: Check Docker Logs for created resources
echo "📝 4. Recent Resource Creation (from logs):"
echo "--------------------------------------------"
echo "US-East LocalStack logs:"
docker logs localstack-us-east 2>&1 | grep -iE "(creating|created).*(bucket|queue|role)" | tail -3 || echo "  (No recent logs)"
echo ""
echo "EU-Central LocalStack logs:"
docker logs localstack-eu-central 2>&1 | grep -iE "(creating|created).*(bucket|queue|role)" | tail -3 || echo "  (No recent logs)"
echo ""

# Method 5: Terraform State
echo "📊 5. Resources in Terraform State:"
echo "-----------------------------------"
terraform show 2>/dev/null | grep -E "resource \"aws_(s3_bucket|sqs_queue|iam_role)" | \
    sed 's/^# /  ✓ /' | head -10 || echo "  (No state file)"
echo ""

echo "✅ Verification complete!"

