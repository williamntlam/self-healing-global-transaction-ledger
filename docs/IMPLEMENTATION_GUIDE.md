# Implementation Guide: Self-Healing Global Transaction Ledger

This guide provides a comprehensive roadmap for implementing Project Atlas from scratch. It includes detailed explanations of concepts you need to master, why they're essential, and step-by-step implementation instructions.

---

## 📚 Table of Contents

1. [Prerequisites & Core Concepts](#prerequisites--core-concepts)
2. [Phase 1: Foundation Setup](#phase-1-foundation-setup)
3. [Phase 2: Cloud Infrastructure (LocalStack + Terraform)](#phase-2-cloud-infrastructure-localstack--terraform)
4. [Phase 3: Kubernetes Orchestration (K3d + Ansible)](#phase-3-kubernetes-orchestration-k3d--ansible)
5. [Phase 4: GitOps & Application Deployment (ArgoCD)](#phase-4-gitops--application-deployment-argocd)
6. [Phase 5: Database Layer (CockroachDB)](#phase-5-database-layer-cockroachdb)
7. [Phase 6: Observability (Prometheus & Grafana)](#phase-6-observability-prometheus--grafana)
8. [Phase 7: Testing & Chaos Engineering](#phase-7-testing--chaos-engineering)
9. [Troubleshooting & Common Issues](#troubleshooting--common-issues)

---

## Prerequisites & Core Concepts

### 1. Docker & Containerization

**Why Learn This:**
- Docker is the foundation for running all components locally (LocalStack, K3d clusters, applications)
- Understanding containers is essential for Kubernetes and modern DevOps practices
- You'll need to manage multiple containers, networks, and volumes

**What to Learn:**
- Docker basics: images, containers, Dockerfile
- Docker Compose for multi-container applications
- Docker networking (bridge, host, overlay networks)
- Volume management (named volumes, bind mounts)
- Container lifecycle (start, stop, restart, logs, exec)

**Key Commands:**
```bash
docker ps, docker images, docker network ls
docker-compose up/down
docker exec -it <container> /bin/sh
docker logs -f <container>
```

**Why It's Needed:**
- LocalStack runs as Docker containers
- K3d creates Kubernetes clusters using Docker containers
- All application components will run in containers
- You'll need to debug container networking between regions

---

### 2. Infrastructure as Code (IaC) with Terraform

**Why Learn This:**
- Terraform manages your cloud infrastructure declaratively
- You'll use provider aliases to create isolated resources per region
- This is industry-standard for production infrastructure management

**What to Learn:**
- Terraform syntax (HCL - HashiCorp Configuration Language)
- Providers, resources, and data sources
- Variables, outputs, and locals
- **Provider aliases** (critical for multi-region setup)
- Modules (for reusable regional stacks)
- State management (terraform.tfstate)
- Workspaces (optional, for environment separation)

**Key Concepts:**
```hcl
# Provider aliases allow you to use the same provider multiple times
provider "aws" {
  alias  = "us_east"
  region = "us-east-1"
  endpoint_urls = { s3 = "http://localhost:4566" }
}

provider "aws" {
  alias  = "eu_central"
  region = "eu-central-1"
  endpoint_urls = { s3 = "http://localhost:4567" }
}

# Use aliases in resources
resource "aws_s3_bucket" "audit_logs" {
  provider = aws.us_east
  bucket   = "audit-logs-us-east"
}
```

**Why It's Needed:**
- Creates S3 buckets and SQS queues in each region
- Manages IAM roles and policies
- Ensures infrastructure is version-controlled and reproducible
- Provider aliases let you simulate multi-region AWS resources on LocalStack

**Learning Resources:**
- Terraform documentation: https://www.terraform.io/docs
- Focus on: Providers, Modules, State Management

---

### 3. LocalStack (AWS Emulation)

**Why Learn This:**
- LocalStack lets you run AWS services locally without real AWS accounts
- Essential for development and testing multi-region scenarios
- You'll configure cross-region replication between two LocalStack instances

**What to Learn:**
- LocalStack architecture and services (S3, SQS, IAM)
- Multi-region setup (running multiple LocalStack instances)
- Endpoint configuration (different ports per region)
- LocalStack persistence and data management
- Integration with Terraform AWS provider

**Key Configuration:**
```yaml
# docker-compose.yml for LocalStack
services:
  localstack-us:
    image: localstack/localstack
    ports:
      - "4566:4566"  # US-East endpoint
    environment:
      - SERVICES=s3,sqs,iam
      - DATA_DIR=/tmp/localstack/data
      
  localstack-eu:
    image: localstack/localstack
    ports:
      - "4567:4566"  # EU-Central endpoint (mapped to different host port)
    environment:
      - SERVICES=s3,sqs,iam
      - DATA_DIR=/tmp/localstack/data
```

**Why It's Needed:**
- Simulates AWS S3 for audit logs storage
- Simulates AWS SQS for transaction queues
- Allows testing multi-region scenarios without AWS costs
- Enables cross-region replication testing

**Learning Resources:**
- LocalStack documentation: https://docs.localstack.cloud/
- Focus on: Multi-region setup, S3 replication, SQS configuration

---

### 4. Kubernetes Fundamentals

**Why Learn This:**
- Kubernetes orchestrates all application components
- You'll manage two clusters (one per region)
- Understanding K8s is essential for modern DevOps

**What to Learn:**
- Core concepts: Pods, Services, Deployments, StatefulSets
- Namespaces (for resource isolation)
- ConfigMaps and Secrets (for configuration)
- Services (ClusterIP, NodePort, LoadBalancer)
- PersistentVolumes and PersistentVolumeClaims
- ServiceAccounts and RBAC
- NetworkPolicies (for security)

**Key Concepts:**
```yaml
# Example StatefulSet (for CockroachDB)
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: cockroachdb
spec:
  serviceName: cockroachdb
  replicas: 3
  template:
    spec:
      containers:
      - name: cockroachdb
        image: cockroachdb/cockroach:latest
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 10Gi
```

**Why It's Needed:**
- CockroachDB runs as a StatefulSet (needs stable network identity)
- Applications run as Deployments
- Services expose applications within and across clusters
- ArgoCD manages Kubernetes resources declaratively

**Learning Resources:**
- Kubernetes documentation: https://kubernetes.io/docs/
- Interactive tutorial: https://kubernetes.io/docs/tutorials/
- Focus on: Pods, Services, Deployments, StatefulSets, ConfigMaps, Secrets

---

### 5. K3d (Lightweight Kubernetes)

**Why Learn This:**
- K3d runs full Kubernetes clusters in Docker containers
- Much lighter than minikube or kind for multi-cluster setups
- Perfect for local multi-region simulation

**What to Learn:**
- K3d cluster creation and management
- Multi-cluster networking (connecting clusters)
- Kubeconfig management (switching between clusters)
- LoadBalancer configuration (using k3d's built-in LB or MetalLB)
- Persistent storage in K3d

**Key Commands:**
```bash
# Create cluster
k3d cluster create dc-us --port "8080:80@loadbalancer"

# Create second cluster
k3d cluster create dc-eu --port "8081:80@loadbalancer"

# List clusters
k3d cluster list

# Get kubeconfig
k3d kubeconfig merge dc-us dc-eu --kubeconfig-merge-default

# Delete cluster
k3d cluster delete dc-us
```

**Why It's Needed:**
- Creates isolated Kubernetes clusters for each region
- Simulates real multi-cluster environments
- Allows testing cross-cluster communication
- Lightweight enough to run two clusters on a laptop

**Learning Resources:**
- K3d documentation: https://k3d.io/
- Focus on: Multi-cluster setup, networking, LoadBalancer configuration

---

### 6. Ansible for Automation

**Why Learn This:**
- Ansible automates "Day 0" infrastructure setup
- Manages K3d cluster creation and configuration
- Installs and configures ArgoCD across clusters

**What to Learn:**
- Ansible playbooks and tasks
- Roles (for reusable automation)
- Inventory management (static and dynamic)
- Modules (especially k3d, kubernetes.core)
- Variables and templates (Jinja2)
- Handlers (for triggered actions)
- Conditionals and loops

**Key Concepts:**
```yaml
# Example playbook structure
- name: Setup US-East cluster
  hosts: localhost
  roles:
    - k3d-cluster
      vars:
        cluster_name: dc-us
        region: us-east-1
    - argocd-install
      vars:
        cluster: dc-us
```

**Why It's Needed:**
- Automates repetitive cluster setup tasks
- Ensures consistent configuration across regions
- Installs ArgoCD and required operators
- Configures networking between clusters
- Sets up LoadBalancer services

**Learning Resources:**
- Ansible documentation: https://docs.ansible.com/
- Focus on: Playbooks, Roles, Modules, Inventory

---

### 7. GitOps with ArgoCD

**Why Learn This:**
- ArgoCD provides continuous delivery using Git as the source of truth
- ApplicationSet pattern manages multiple clusters declaratively
- Industry-standard for Kubernetes deployments

**What to Learn:**
- ArgoCD architecture (Application Controller, Repo Server, API Server)
- Application CRD (Custom Resource Definition)
- **ApplicationSet** (for multi-cluster deployments)
- Generators (List, Cluster, Git, Matrix)
- Sync policies (automatic vs manual)
- Health checks and sync status
- Multi-cluster management

**Key Concepts:**
```yaml
# ApplicationSet example
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: ledger-app
spec:
  generators:
  - clusters:
      selector:
        matchLabels:
          region: us-east-1
  template:
    metadata:
      name: '{{name}}-ledger'
    spec:
      project: default
      source:
        repoURL: https://github.com/your-org/gitops
        path: charts/ledger-app
      destination:
        server: '{{server}}'
        namespace: ledger
```

**Why It's Needed:**
- Manages application state across both clusters
- Automatically syncs changes from Git
- Provides visibility into deployment status
- ApplicationSet pattern scales to many clusters
- Enables disaster recovery (redeploy from Git)

**Learning Resources:**
- ArgoCD documentation: https://argo-cd.readthedocs.io/
- ApplicationSet guide: https://argocd-applicationset.readthedocs.io/
- Focus on: Application CRD, ApplicationSet, Generators, Multi-cluster

---

### 8. CockroachDB (Distributed SQL Database)

**Why Learn This:**
- CockroachDB provides the distributed ledger with strong consistency
- Uses Raft consensus for zero-data-loss failover
- Handles multi-region deployments natively

**What to Learn:**
- CockroachDB architecture (nodes, ranges, replication)
- Raft consensus algorithm basics
- Multi-region deployment patterns
- SQL syntax (PostgreSQL-compatible)
- Connection strings and client libraries
- Backup and restore procedures
- Monitoring and observability

**Key Concepts:**
```sql
-- CockroachDB supports multi-region SQL
CREATE DATABASE ledger;

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region STRING NOT NULL,
    amount DECIMAL NOT NULL,
    timestamp TIMESTAMP DEFAULT now()
) LOCALITY REGIONAL BY ROW;

-- Set survival goals for multi-region
ALTER DATABASE ledger SURVIVE REGION FAILURE;
```

**Why It's Needed:**
- Provides ACID transactions across regions
- Automatic failover with zero data loss
- Handles network partitions gracefully
- Maintains consistency during regional outages
- Scales horizontally across regions

**Learning Resources:**
- CockroachDB documentation: https://www.cockroachlabs.com/docs/
- Multi-region tutorial: https://www.cockroachlabs.com/docs/stable/multiregion-overview.html
- Focus on: Multi-region deployment, Raft consensus, Survival goals

---

### 9. Helm Charts

**Why Learn This:**
- Helm packages Kubernetes applications
- ArgoCD uses Helm charts for deployments
- Standard way to deploy complex applications

**What to Learn:**
- Helm chart structure (Chart.yaml, values.yaml, templates/)
- Template syntax (Go templates)
- Values files and overrides
- Helm install/upgrade/rollback
- Chart dependencies
- Hooks (pre-install, post-install, etc.)

**Key Structure:**
```
charts/ledger-app/
├── Chart.yaml
├── values.yaml
└── templates/
    ├── deployment.yaml
    ├── service.yaml
    └── configmap.yaml
```

**Why It's Needed:**
- Packages CockroachDB deployment
- Packages ledger application
- Allows environment-specific configuration
- ArgoCD deploys from Helm charts
- Enables versioning and rollbacks

**Learning Resources:**
- Helm documentation: https://helm.sh/docs/
- Focus on: Chart structure, Templates, Values

---

### 10. Prometheus & Grafana (Observability)

**Why Learn This:**
- Monitoring is essential for production systems
- Prometheus collects metrics, Grafana visualizes them
- Critical for understanding system health during chaos testing

**What to Learn:**
- Prometheus data model (metrics, labels, time series)
- PromQL (Prometheus Query Language)
- Service discovery and scraping
- Grafana dashboards and panels
- Alerting rules
- Exporters (for application metrics)

**Why It's Needed:**
- Monitors CockroachDB cluster health
- Tracks application performance
- Visualizes metrics during failover tests
- Alerts on anomalies
- Essential for validating self-healing behavior

**Learning Resources:**
- Prometheus documentation: https://prometheus.io/docs/
- Grafana documentation: https://grafana.com/docs/
- Focus on: PromQL, Service discovery, Dashboard creation

---

## Phase 1: Foundation Setup

### Step 1.1: Install Prerequisites

**Install Docker:**
```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
newgrp docker

# Verify
docker --version
docker-compose --version
```

**Install Terraform:**
```bash
# Download and install
wget https://releases.hashicorp.com/terraform/1.6.0/terraform_1.6.0_linux_amd64.zip
unzip terraform_1.6.0_linux_amd64.zip
sudo mv terraform /usr/local/bin/
terraform --version
```

**Install Ansible:**
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y ansible

# Verify
ansible --version
```

**Install K3d:**
```bash
# Using script
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

# Or using package manager
# Ubuntu/Debian
wget -q -O - https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

# Verify
k3d --version
```

**Install kubectl:**
```bash
# Download kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
kubectl version --client
```

**Install Helm:**
```bash
# Download and install
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
helm version
```

### Step 1.2: Create Project Structure

```bash
cd /home/williamntlam/Projects/self-healing-global-transaction-ledger

mkdir -p infrastructure/modules/regional-stack
mkdir -p orchestration/roles/{k3d-cluster,argocd-install,loadbalancer-config}
mkdir -p gitops/appsets
mkdir -p gitops/charts/{ledger-app,cockroachdb}
mkdir -p scripts
```

---

## Phase 2: Cloud Infrastructure (LocalStack + Terraform)

### Step 2.1: Setup LocalStack Multi-Region

**Create `infrastructure/docker-compose.localstack.yml`:**
```yaml
version: '3.8'

services:
  localstack-us:
    image: localstack/localstack:latest
    container_name: localstack-us-east
    ports:
      - "4566:4566"  # US-East endpoint
    environment:
      - SERVICES=s3,sqs,iam
      - DEBUG=1
      - DATA_DIR=/var/lib/localstack/data
      - PERSISTENCE=1
      - LAMBDA_EXECUTOR=docker
      - DOCKER_HOST=unix:///var/run/docker.sock
    volumes:
      - "../localstack-data/us-east:/var/lib/localstack"
      - "/var/run/docker.sock:/var/run/docker.sock"
    networks:
      - atlas-network

  localstack-eu:
    image: localstack/localstack:latest
    container_name: localstack-eu-central
    ports:
      - "4567:4566"  # EU-Central endpoint (different host port)
    environment:
      - SERVICES=s3,sqs,iam
      - DEBUG=1
      - DATA_DIR=/var/lib/localstack/data
      - PERSISTENCE=1
      - LAMBDA_EXECUTOR=docker
      - DOCKER_HOST=unix:///var/run/docker.sock
    volumes:
      - "../localstack-data/eu-central:/var/lib/localstack"
      - "/var/run/docker.sock:/var/run/docker.sock"
    networks:
      - atlas-network

networks:
  atlas-network:
    driver: bridge
```

**Start LocalStack:**
```bash
cd infrastructure
docker compose -f docker-compose.localstack.yml up -d
docker compose -f docker-compose.localstack.yml ps
```

**Verify LocalStack:**
```bash
# Option 1: Using curl (no AWS CLI needed)
curl http://localhost:4566/_localstack/health
curl http://localhost:4567/_localstack/health

# Option 2: Using AWS CLI (if installed)
aws --endpoint-url=http://localhost:4566 s3 ls
aws --endpoint-url=http://localhost:4567 s3 ls
```

**Note:** AWS CLI is optional. You can verify LocalStack is working with curl. Terraform (used later) doesn't require AWS CLI - it uses its own AWS provider.

**Why This Setup:**
- Two separate LocalStack instances simulate two AWS regions
- Different ports (4566 vs 4567) allow both to run simultaneously
- Separate data directories ensure isolation
- Shared Docker network enables cross-region replication simulation

### Step 2.2: Create Terraform Configuration

**Create `infrastructure/providers.tf`:**
```hcl
terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# US-East Provider
provider "aws" {
  alias  = "us_east"
  region = "us-east-1"
  
  # LocalStack endpoints
  endpoints {
    s3  = "http://localhost:4566"
    sqs = "http://localhost:4566"
    iam = "http://localhost:4566"
  }
  
  # LocalStack requires these settings
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_region_validation      = true
  access_key                  = "test"
  secret_key                  = "test"
}

# EU-Central Provider
provider "aws" {
  alias  = "eu_central"
  region = "eu-central-1"
  
  # LocalStack endpoints (different port)
  endpoints {
    s3  = "http://localhost:4567"
    sqs = "http://localhost:4567"
    iam = "http://localhost:4567"
  }
  
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_region_validation      = true
  access_key                  = "test"
  secret_key                  = "test"
}
```

**Why Provider Aliases:**
- Allows using the same AWS provider multiple times
- Each alias points to a different LocalStack instance (region)
- Terraform can manage resources in both regions from one codebase
- This is the standard pattern for multi-region Terraform

**Create `infrastructure/modules/regional-stack/main.tf`:**
```hcl
# S3 Bucket for Audit Logs
resource "aws_s3_bucket" "audit_logs" {
  provider = var.provider
  bucket   = "${var.region}-audit-logs"
  
  tags = {
    Region = var.region
    Purpose = "AuditLogs"
  }
}

# Enable versioning for audit logs
resource "aws_s3_bucket_versioning" "audit_logs" {
  provider = var.provider
  bucket   = aws_s3_bucket.audit_logs.id
  
  versioning_configuration {
    status = "Enabled"
  }
}

# SQS Queue for Transactions
resource "aws_sqs_queue" "transaction_queue" {
  provider = var.provider
  name     = "${var.region}-transaction-queue"
  
  # Visibility timeout (30 seconds)
  visibility_timeout_seconds = 30
  
  # Message retention (14 days)
  message_retention_seconds = 1209600
  
  tags = {
    Region = var.region
    Purpose = "TransactionQueue"
  }
}

# IAM Role for Application
resource "aws_iam_role" "ledger_app_role" {
  provider = var.provider
  name     = "${var.region}-ledger-app-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
    }]
  })
}

# IAM Policy for S3 and SQS access
resource "aws_iam_role_policy" "ledger_app_policy" {
  provider = var.provider
  name     = "${var.region}-ledger-app-policy"
  role     = aws_iam_role.ledger_app_role.id
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.audit_logs.arn,
          "${aws_s3_bucket.audit_logs.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage",
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes"
        ]
        Resource = aws_sqs_queue.transaction_queue.arn
      }
    ]
  })
}
```

**Create `infrastructure/modules/regional-stack/variables.tf`:**
```hcl
variable "provider" {
  description = "AWS provider alias"
  # Type is inferred from usage
}

variable "region" {
  description = "AWS region name"
  type        = string
}
```

**Create `infrastructure/modules/regional-stack/outputs.tf`:**
```hcl
output "s3_bucket_name" {
  description = "S3 bucket name for audit logs"
  value       = aws_s3_bucket.audit_logs.id
}

output "sqs_queue_url" {
  description = "SQS queue URL"
  value       = aws_sqs_queue.transaction_queue.url
}

output "iam_role_arn" {
  description = "IAM role ARN"
  value       = aws_iam_role.ledger_app_role.arn
}
```

**Create `infrastructure/main.tf`:**
```hcl
# US-East Regional Stack
module "us_east" {
  source = "./modules/regional-stack"
  
  provider = aws.us_east
  region   = "us-east-1"
}

# EU-Central Regional Stack
module "eu_central" {
  source = "./modules/regional-stack"
  
  provider = aws.eu_central
  region   = "eu-central-1"
}
```

**Create `infrastructure/outputs.tf`:**
```hcl
output "us_east_s3_bucket" {
  value = module.us_east.s3_bucket_name
}

output "us_east_sqs_queue" {
  value = module.us_east.sqs_queue_url
}

output "eu_central_s3_bucket" {
  value = module.eu_central.s3_bucket_name
}

output "eu_central_sqs_queue" {
  value = module.eu_central.sqs_queue_url
}
```

**Initialize and Apply Terraform:**
```bash
cd infrastructure
terraform init
terraform plan
terraform apply
```

**Verify Resources:**
```bash
# Check US-East resources
aws --endpoint-url=http://localhost:4566 s3 ls
aws --endpoint-url=http://localhost:4566 sqs list-queues

# Check EU-Central resources
aws --endpoint-url=http://localhost:4567 s3 ls
aws --endpoint-url=http://localhost:4567 sqs list-queues
```

---

## Phase 3: Kubernetes Orchestration (K3d + Ansible)

### Step 3.1: Create Ansible Inventory

**Create `orchestration/inventory.yml`:**
```yaml
all:
  children:
    clusters:
      hosts:
        localhost:
          ansible_connection: local
          clusters:
            - name: dc-us
              region: us-east-1
              port_mapping: "8080:80@loadbalancer"
            - name: dc-eu
              region: eu-central-1
              port_mapping: "8081:80@loadbalancer"
```

### Step 3.2: Create K3d Cluster Role

**Create `orchestration/roles/k3d-cluster/tasks/main.yml`:**
```yaml
---
- name: Check if K3d is installed
  command: which k3d
  register: k3d_check
  changed_when: false
  failed_when: false

- name: Fail if K3d is not installed
  fail:
    msg: "K3d is not installed. Please install it first."
  when: k3d_check.rc != 0

- name: Create K3d cluster
  command: >
    k3d cluster create {{ cluster_name }}
    --port {{ port_mapping }}
    --wait
    --timeout 300s
  register: cluster_create
  changed_when: true
  failed_when: cluster_create.rc != 0

- name: Verify cluster is running
  command: k3d cluster list
  register: cluster_list
  changed_when: false

- name: Display cluster status
  debug:
    var: cluster_list.stdout_lines
```

**Create `orchestration/roles/k3d-cluster/defaults/main.yml`:**
```yaml
---
cluster_name: "dc-us"
port_mapping: "8080:80@loadbalancer"
```

**Why Ansible for K3d:**
- Automates cluster creation with consistent configuration
- Can be extended to add nodes, configure networking
- Idempotent (can run multiple times safely)
- Integrates with other automation (ArgoCD install)

### Step 3.3: Create LoadBalancer Configuration Role

**Create `orchestration/roles/loadbalancer-config/tasks/main.yml`:**
```yaml
---
- name: Create MetalLB namespace
  kubernetes.core.k8s:
    name: metallb-system
    api_version: v1
    kind: Namespace
    state: present
    kubeconfig: "{{ kubeconfig_path }}"

- name: Install MetalLB using Helm
  kubernetes.core.helm:
    name: metallb
    chart_ref: metallb/metallb
    release_namespace: metallb-system
    kubeconfig: "{{ kubeconfig_path }}"
    create_namespace: false

- name: Create IPAddressPool for LoadBalancer
  kubernetes.core.k8s:
    definition:
      apiVersion: metallb.io/v1beta1
      kind: IPAddressPool
      metadata:
        name: default-pool
        namespace: metallb-system
      spec:
        addresses:
        - 172.18.0.100-172.18.0.200
    kubeconfig: "{{ kubeconfig_path }}"
```

**Why LoadBalancer:**
- ArgoCD and applications need LoadBalancer services
- K3d's built-in LB may not be sufficient for all use cases
- MetalLB provides proper LoadBalancer support in local clusters

### Step 3.4: Create ArgoCD Install Role

**Create `orchestration/roles/argocd-install/tasks/main.yml`:**
```yaml
---
- name: Create ArgoCD namespace
  kubernetes.core.k8s:
    name: argocd
    api_version: v1
    kind: Namespace
    state: present
    kubeconfig: "{{ kubeconfig_path }}"

- name: Add ArgoCD Helm repository
  kubernetes.core.helm:
    name: argocd
    repo_url: https://argoproj.github.io/argo-helm
    chart_ref: argo-cd
    release_namespace: argocd
    kubeconfig: "{{ kubeconfig_path }}"
    values:
      server:
        service:
          type: LoadBalancer
      configs:
        params:
          server.insecure: true

- name: Wait for ArgoCD server to be ready
  kubernetes.core.k8s_info:
    api_version: v1
    kind: Pod
    namespace: argocd
    label_selectors:
      - app.kubernetes.io/name=argocd-server
    kubeconfig: "{{ kubeconfig_path }}"
  register: argocd_pods
  until: argocd_pods.resources | length > 0
  retries: 30
  delay: 10

- name: Get ArgoCD admin password
  kubernetes.core.k8s:
    name: argocd-initial-admin-secret
    namespace: argocd
    api_version: v1
    kind: Secret
    kubeconfig: "{{ kubeconfig_path }}"
  register: argocd_secret

- name: Display ArgoCD admin password
  debug:
    msg: "ArgoCD admin password: {{ argocd_secret.resource.data.password | b64decode }}"
```

### Step 3.5: Create Master Playbook

**Create `orchestration/site.yml`:**
```yaml
---
- name: Build Project Atlas Infrastructure
  hosts: localhost
  connection: local
  gather_facts: yes
  
  vars:
    kubeconfig_dir: "{{ playbook_dir }}/../.kubeconfigs"
  
  tasks:
    - name: Create kubeconfig directory
      file:
        path: "{{ kubeconfig_dir }}"
        state: directory
    
    - name: Create US-East cluster
      include_role:
        name: k3d-cluster
      vars:
        cluster_name: dc-us
        port_mapping: "8080:80@loadbalancer"
    
    - name: Create EU-Central cluster
      include_role:
        name: k3d-cluster
      vars:
        cluster_name: dc-eu
        port_mapping: "8081:80@loadbalancer"
    
    - name: Merge kubeconfigs
      command: >
        k3d kubeconfig merge dc-us dc-eu
        --kubeconfig-merge-default
      register: kubeconfig_merge
    
    - name: Configure LoadBalancer for US-East
      include_role:
        name: loadbalancer-config
      vars:
        kubeconfig_path: "{{ kubeconfig_dir }}/dc-us.yaml"
        cluster: dc-us
    
    - name: Configure LoadBalancer for EU-Central
      include_role:
        name: loadbalancer-config
      vars:
        kubeconfig_path: "{{ kubeconfig_dir }}/dc-eu.yaml"
        cluster: dc-eu
    
    - name: Install ArgoCD on US-East
      include_role:
        name: argocd-install
      vars:
        kubeconfig_path: "{{ kubeconfig_dir }}/dc-us.yaml"
        cluster: dc-us
    
    - name: Install ArgoCD on EU-Central
      include_role:
        name: argocd-install
      vars:
        kubeconfig_path: "{{ kubeconfig_dir }}/dc-eu.yaml"
        cluster: dc-eu
```

**Run the Playbook:**
```bash
cd orchestration
ansible-playbook site.yml
```

**Verify Clusters:**
```bash
kubectl config get-contexts
kubectl --context k3d-dc-us get nodes
kubectl --context k3d-dc-eu get nodes
```

---

## Phase 4: GitOps & Application Deployment (ArgoCD)

### Step 4.1: Register Clusters in ArgoCD

**Create `gitops/clusters/us-east.yaml`:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dc-us-cluster
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: cluster
type: Opaque
stringData:
  name: dc-us
  server: https://kubernetes.default.svc
  config: |
    {
      "bearerToken": "<token>",
      "tlsClientConfig": {
        "insecure": true
      }
    }
```

**Why Cluster Registration:**
- ArgoCD needs to know about both clusters
- Allows deploying applications to specific clusters
- ApplicationSet uses cluster labels for targeting

### Step 4.2: Create ApplicationSet

**Create `gitops/appsets/ledger-appset.yaml`:**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: ledger-app
  namespace: argocd
spec:
  generators:
  - clusters:
      selector:
        matchLabels:
          region: us-east-1
      values:
        region: us-east-1
        endpoint: http://localhost:4566
  - clusters:
      selector:
        matchLabels:
          region: eu-central-1
      values:
        region: eu-central-1
        endpoint: http://localhost:4567
  
  template:
    metadata:
      name: '{{name}}-ledger-app'
    spec:
      project: default
      source:
        repoURL: https://github.com/your-org/gitops-repo
        targetRevision: main
        path: charts/ledger-app
        helm:
          valueFiles:
            - values-{{region}}.yaml
      destination:
        server: '{{server}}'
        namespace: ledger
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
```

**Why ApplicationSet:**
- Deploys the same application to multiple clusters
- Uses generators to automatically discover clusters
- Each cluster gets region-specific configuration
- Self-healing: automatically redeploys if manually changed

### Step 4.2: Create Helm Charts

**Create `gitops/charts/ledger-app/Chart.yaml`:**
```yaml
apiVersion: v2
name: ledger-app
description: Global Transaction Ledger Application
type: application
version: 0.1.0
appVersion: "1.0.0"
```

**Create `gitops/charts/ledger-app/values.yaml`:**
```yaml
region: us-east-1
endpoint: http://localhost:4566

image:
  repository: ledger-app
  tag: latest
  pullPolicy: IfNotPresent

replicas: 2

service:
  type: LoadBalancer
  port: 80

config:
  cockroachdb:
    host: cockroachdb-public
    port: 26257
  aws:
    region: us-east-1
    s3Bucket: us-east-1-audit-logs
    sqsQueue: us-east-1-transaction-queue
```

**Create `gitops/charts/ledger-app/templates/deployment.yaml`:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Values.region }}-ledger-app
  labels:
    app: ledger-app
    region: {{ .Values.region }}
spec:
  replicas: {{ .Values.replicas }}
  selector:
    matchLabels:
      app: ledger-app
      region: {{ .Values.region }}
  template:
    metadata:
      labels:
        app: ledger-app
        region: {{ .Values.region }}
    spec:
      containers:
      - name: ledger-app
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
        imagePullPolicy: {{ .Values.image.pullPolicy }}
        env:
        - name: REGION
          value: {{ .Values.region | quote }}
        - name: COCKROACHDB_HOST
          value: {{ .Values.config.cockroachdb.host | quote }}
        - name: COCKROACHDB_PORT
          value: {{ .Values.config.cockroachdb.port | quote }}
        - name: AWS_ENDPOINT
          value: {{ .Values.endpoint | quote }}
        - name: S3_BUCKET
          value: {{ .Values.config.aws.s3Bucket | quote }}
        - name: SQS_QUEUE
          value: {{ .Values.config.aws.sqsQueue | quote }}
        ports:
        - containerPort: 8080
          name: http
```

**Apply ApplicationSet:**
```bash
kubectl --context k3d-dc-us apply -f gitops/appsets/ledger-appset.yaml
```

---

## Phase 5: Database Layer (CockroachDB)

### Step 5.1: Create CockroachDB Helm Chart

**Create `gitops/charts/cockroachdb/Chart.yaml`:**
```yaml
apiVersion: v2
name: cockroachdb
description: CockroachDB Multi-Region Cluster
type: application
version: 0.1.0
appVersion: "23.1.0"
```

**Create `gitops/charts/cockroachdb/values.yaml`:**
```yaml
statefulset:
  replicas: 3

conf:
  cache: "256MiB"
  max-sql-memory: "256MiB"

storage:
  persistentVolume:
    size: 10Gi

service:
  type: LoadBalancer
```

**Create `gitops/charts/cockroachdb/templates/statefulset.yaml`:**
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: cockroachdb
spec:
  serviceName: cockroachdb
  replicas: {{ .Values.statefulset.replicas }}
  selector:
    matchLabels:
      app: cockroachdb
  template:
    metadata:
      labels:
        app: cockroachdb
    spec:
      containers:
      - name: cockroachdb
        image: cockroachdb/cockroach:{{ .Values.appVersion }}
        ports:
        - containerPort: 26257
          name: grpc
        - containerPort: 8080
          name: http
        command:
        - /cockroach/cockroach
        - start
        - --join
        - cockroachdb-0.cockroachdb,cockroachdb-1.cockroachdb,cockroachdb-2.cockroachdb
        - --advertise-addr
        - $(POD_NAME).cockroachdb
        - --http-addr
        - 0.0.0.0:8080
        - --cache={{ .Values.conf.cache }}
        - --max-sql-memory={{ .Values.conf.max-sql-memory }}
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        volumeMounts:
        - name: datadir
          mountPath: /cockroach/cockroach-data
  volumeClaimTemplates:
  - metadata:
      name: datadir
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: {{ .Values.storage.persistentVolume.size }}
```

**Why StatefulSet:**
- CockroachDB needs stable network identity (hostnames)
- Each pod needs persistent storage
- Pods are created in order (0, 1, 2)
- Perfect for distributed databases

### Step 5.2: Configure Multi-Region SQL

**Create initialization script:**
```sql
-- Initialize database with multi-region configuration
CREATE DATABASE ledger;

USE ledger;

-- Create transactions table with regional locality
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region STRING NOT NULL,
    amount DECIMAL(19,2) NOT NULL,
    from_account STRING NOT NULL,
    to_account STRING NOT NULL,
    timestamp TIMESTAMP DEFAULT now(),
    status STRING DEFAULT 'pending'
) LOCALITY REGIONAL BY ROW AS region;

-- Set survival goals
ALTER DATABASE ledger SURVIVE REGION FAILURE;

-- Create indexes
CREATE INDEX idx_timestamp ON transactions(timestamp);
CREATE INDEX idx_status ON transactions(status);
```

**Why Multi-Region SQL:**
- CockroachDB automatically places data close to users
- `SURVIVE REGION FAILURE` ensures availability during outages
- `REGIONAL BY ROW` optimizes for regional access patterns
- Zero-data-loss failover with Raft consensus

---

## Phase 6: Observability (Prometheus & Grafana)

### Step 6.1: Install Prometheus Operator

**Create `gitops/charts/monitoring/Chart.yaml`:**
```yaml
apiVersion: v2
name: monitoring
description: Prometheus and Grafana Stack
type: application
version: 0.1.0
dependencies:
  - name: kube-prometheus-stack
    version: "55.0.0"
    repository: https://prometheus-community.github.io/helm-charts
```

**Why Prometheus Operator:**
- Manages Prometheus and Grafana automatically
- ServiceMonitor CRD for automatic metric discovery
- Pre-configured dashboards for Kubernetes
- AlertManager for alerting

### Step 6.2: Create ServiceMonitor for CockroachDB

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: cockroachdb
spec:
  selector:
    matchLabels:
      app: cockroachdb
  endpoints:
  - port: http
    path: /_status/vars
```

**Why ServiceMonitor:**
- Automatically discovers and scrapes CockroachDB metrics
- No manual Prometheus configuration needed
- Updates automatically when services change

---

## Phase 7: Testing & Chaos Engineering

### Step 7.1: Create Chaos Script

**Create `scripts/blast_radius.sh`:**
```bash
#!/bin/bash

# Simulate regional outage by stopping a cluster

REGION=${1:-us-east-1}
CLUSTER_NAME="dc-${REGION%%-*}"

echo "Simulating outage in region: $REGION"
echo "Stopping cluster: $CLUSTER_NAME"

# Stop the cluster
k3d cluster stop $CLUSTER_NAME

echo "Cluster stopped. Monitoring failover..."
echo "Check ArgoCD and CockroachDB for failover status"

# Wait for user to restore
read -p "Press Enter to restore the cluster..."

# Restore the cluster
k3d cluster start $CLUSTER_NAME

echo "Cluster restored. Monitoring recovery..."
```

**Why Chaos Engineering:**
- Validates self-healing capabilities
- Tests disaster recovery procedures
- Ensures zero-data-loss during failover
- Validates ArgoCD's self-healing behavior

### Step 7.2: Test Scenarios

1. **Regional Outage Test:**
   ```bash
   ./scripts/blast_radius.sh us-east-1
   # Verify: EU cluster handles all traffic
   # Verify: No data loss in CockroachDB
   # Verify: ArgoCD redeploys when cluster restored
   ```

2. **Network Partition Test:**
   - Simulate network issues between clusters
   - Verify CockroachDB maintains consistency
   - Check Raft consensus behavior

3. **Application Failure Test:**
   - Delete a pod manually
   - Verify Kubernetes recreates it
   - Verify ArgoCD detects and fixes drift

---

## Troubleshooting & Common Issues

### Issue: LocalStack endpoints not accessible

**Solution:**
```bash
# Check if LocalStack is running
docker ps | grep localstack

# Check network connectivity
curl http://localhost:4566/health

# Verify Docker network
docker network inspect atlas-network
```

### Issue: K3d clusters can't communicate

**Solution:**
```bash
# Check cluster networks
docker network ls | grep k3d

# Verify kubeconfig
kubectl config get-contexts

# Test connectivity
kubectl --context k3d-dc-us get nodes
kubectl --context k3d-dc-eu get nodes
```

### Issue: ArgoCD can't sync applications

**Solution:**
```bash
# Check ArgoCD server logs
kubectl --context k3d-dc-us logs -n argocd -l app.kubernetes.io/name=argocd-server

# Verify repository access
argocd repo list

# Check application status
argocd app list
argocd app get <app-name>
```

### Issue: CockroachDB nodes not joining

**Solution:**
```bash
# Check StatefulSet status
kubectl get statefulset cockroachdb

# Check pod logs
kubectl logs cockroachdb-0

# Verify DNS resolution
kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup cockroachdb-0.cockroachdb
```

---

## Next Steps & Advanced Topics

1. **Cross-Region Replication:**
   - Implement S3 cross-region replication in LocalStack
   - Configure SQS cross-region queues
   - Test data consistency

2. **Service Mesh:**
   - Add Istio or Linkerd for advanced traffic management
   - Implement mTLS between regions
   - Add distributed tracing

3. **CI/CD Integration:**
   - Set up GitHub Actions for automated testing
   - Implement GitOps workflows
   - Add automated chaos testing

4. **Security Hardening:**
   - Implement network policies
   - Add RBAC policies
   - Enable Pod Security Standards
   - Add secret management (Vault)

5. **Performance Optimization:**
   - Tune CockroachDB for multi-region
   - Optimize application queries
   - Implement connection pooling
   - Add caching layers

---

## Learning Path Summary

**Week 1-2: Foundations**
- Docker and containerization
- Basic Kubernetes concepts
- Terraform basics

**Week 3-4: Infrastructure**
- LocalStack setup and configuration
- Terraform modules and provider aliases
- Multi-region infrastructure patterns

**Week 5-6: Orchestration**
- K3d multi-cluster setup
- Ansible automation
- Kubernetes networking

**Week 7-8: GitOps**
- ArgoCD fundamentals
- ApplicationSet patterns
- Helm chart development

**Week 9-10: Database**
- CockroachDB architecture
- Multi-region SQL
- Raft consensus basics

**Week 11-12: Observability & Testing**
- Prometheus and Grafana
- Chaos engineering principles
- Disaster recovery testing

---

## Resources for Deep Learning

1. **Kubernetes:**
   - "Kubernetes: Up and Running" by Kelsey Hightower
   - CNCF Kubernetes courses

2. **Terraform:**
   - HashiCorp Learn: https://learn.hashicorp.com/terraform
   - "Terraform: Up and Running" by Yevgeniy Brikman

3. **ArgoCD:**
   - Official documentation and tutorials
   - ArgoCD community examples

4. **CockroachDB:**
   - CockroachDB University
   - Multi-region deployment guides

5. **Observability:**
   - "Prometheus: Up and Running" by Brian Brazil
   - Grafana Labs tutorials

---

Good luck with your implementation! This project will give you hands-on experience with production-grade DevOps practices and multi-region infrastructure patterns.
