# Ledger App Helm Chart

This Helm chart deploys the Global Transaction Ledger application that connects to CockroachDB and uses AWS services (S3, SQS) via LocalStack.

## Features

- **Multi-region support** with region-specific configuration
- **CockroachDB integration** for transaction storage
- **AWS/LocalStack integration** for S3 (audit logs) and SQS (transaction queues)
- **Health checks** (liveness and readiness probes)
- **Pod Disruption Budget** for high availability
- **LoadBalancer service** for external access
- **Resource limits** and security contexts

## Installation

### Basic Installation

```bash
helm install ledger-app . \
  --namespace default \
  --create-namespace
```

### Multi-Region Installation

For US-East region:
```bash
helm install ledger-app-us . \
  --namespace default \
  --set region.name=us-east-1 \
  --set region.awsRegion=us-east-1 \
  --set aws.endpoint=http://localhost:4566 \
  --set aws.s3Bucket=us-east-1-audit-logs \
  --set aws.sqsQueue=us-east-1-transaction-queue
```

For EU-Central region:
```bash
helm install ledger-app-eu . \
  --namespace default \
  --set region.name=eu-central-1 \
  --set region.awsRegion=eu-central-1 \
  --set aws.endpoint=http://localhost:4567 \
  --set aws.s3Bucket=eu-central-1-audit-logs \
  --set aws.sqsQueue=eu-central-1-transaction-queue
```

## Configuration

### Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicas` | Number of application replicas | `2` |
| `image.repository` | Container image repository | `ledger-app` |
| `image.tag` | Container image tag | `latest` |
| `service.type` | Service type | `LoadBalancer` |
| `region.name` | Region identifier | `"us-east-1"` |
| `cockroachdb.host` | CockroachDB service name | `"cockroachdb-public"` |
| `aws.endpoint` | LocalStack endpoint URL | `"http://localhost:4566"` |
| `aws.s3Bucket` | S3 bucket name | `"us-east-1-audit-logs"` |
| `aws.sqsQueue` | SQS queue name | `"us-east-1-transaction-queue"` |

### Example values.yaml

```yaml
region:
  name: "us-east-1"
  awsRegion: "us-east-1"

replicas: 3

image:
  repository: ledger-app
  tag: "v1.0.0"

cockroachdb:
  host: "cockroachdb-public"
  port: 26257
  database: "ledger"

aws:
  endpoint: "http://localhost:4566"
  s3Bucket: "us-east-1-audit-logs"
  sqsQueue: "us-east-1-transaction-queue"
  useSecrets: true
  secretName: "aws-credentials"

resources:
  requests:
    cpu: "200m"
    memory: "256Mi"
  limits:
    cpu: "1000m"
    memory: "512Mi"
```

## Prerequisites

### 1. CockroachDB must be deployed

The ledger app connects to CockroachDB. Ensure CockroachDB is deployed first:

```bash
# Deploy CockroachDB first
helm install cockroachdb ./gitops/charts/cockroachdb

# Then deploy ledger app
helm install ledger-app ./gitops/charts/ledger-app
```

### 2. AWS credentials secret must exist

The app uses AWS credentials from a Kubernetes secret. Create it using the `secrets-config` Ansible role or manually:

```bash
kubectl create secret generic aws-credentials \
  --from-literal=access-key-id=test \
  --from-literal=secret-access-key=test \
  --from-literal=region=us-east-1 \
  --namespace=default
```

### 3. LocalStack must be running

LocalStack provides S3 and SQS services. Ensure it's running:

```bash
docker ps | grep localstack
```

## Environment Variables

The application uses these environment variables:

| Variable | Description | Source |
|----------|-------------|--------|
| `REGION` | Region identifier | `values.yaml` |
| `AWS_REGION` | AWS region | `values.yaml` |
| `COCKROACHDB_HOST` | CockroachDB service name | `values.yaml` |
| `COCKROACHDB_PORT` | CockroachDB port | `values.yaml` |
| `COCKROACHDB_DATABASE` | Database name | `values.yaml` |
| `AWS_ENDPOINT` | LocalStack endpoint | `values.yaml` |
| `S3_BUCKET` | S3 bucket name | `values.yaml` |
| `SQS_QUEUE` | SQS queue name | `values.yaml` |
| `AWS_ACCESS_KEY_ID` | AWS access key | Kubernetes secret |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key | Kubernetes secret |

## Accessing the Application

### Via LoadBalancer Service

After installation, get the external IP:

```bash
kubectl get svc ledger-app
```

Access the application:
```bash
EXTERNAL_IP=$(kubectl get svc ledger-app -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
curl http://$EXTERNAL_IP/health
```

### Via Port Forward

```bash
kubectl port-forward svc/ledger-app 8080:80
curl http://localhost:8080/health
```

## Troubleshooting

### Check Pod Status
```bash
kubectl get pods -l app.kubernetes.io/name=ledger-app
```

### View Logs
```bash
kubectl logs -l app.kubernetes.io/name=ledger-app
```

### Check Deployment
```bash
kubectl get deployment ledger-app
kubectl describe deployment ledger-app
```

### Verify Environment Variables
```bash
kubectl exec -it deployment/ledger-app -- env | grep -E "COCKROACHDB|AWS|S3|SQS"
```

### Test CockroachDB Connection
```bash
kubectl exec -it deployment/ledger-app -- \
  sh -c 'echo "SELECT 1;" | /usr/bin/cockroach sql --insecure --host=$COCKROACHDB_HOST:$COCKROACHDB_PORT'
```

## Resources Created

- **Deployment**: `ledger-app` (or release name)
- **Service**: `ledger-app` (LoadBalancer)
- **ServiceAccount**: `ledger-app` (if enabled)
- **PodDisruptionBudget**: `ledger-app` (if enabled)

## Multi-Region Deployment

For multi-region deployments, deploy to each cluster with region-specific values:

**US-East:**
```bash
helm install ledger-app-us . \
  --set region.name=us-east-1 \
  --set aws.endpoint=http://localhost:4566 \
  --set aws.s3Bucket=us-east-1-audit-logs \
  --set aws.sqsQueue=us-east-1-transaction-queue
```

**EU-Central:**
```bash
helm install ledger-app-eu . \
  --set region.name=eu-central-1 \
  --set aws.endpoint=http://localhost:4567 \
  --set aws.s3Bucket=eu-central-1-audit-logs \
  --set aws.sqsQueue=eu-central-1-transaction-queue
```

## Notes

- The application connects to CockroachDB using the service name `cockroachdb-public`
- AWS credentials are loaded from Kubernetes secrets (created by `secrets-config` role)
- LocalStack endpoints differ by region (4566 for US-East, 4567 for EU-Central)
- S3 bucket and SQS queue names should match Terraform outputs

