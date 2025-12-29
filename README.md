# Self-Healing Global Transaction Ledger

**Project Atlas** is a simulated global infrastructure platform designed to maintain a consistent, high-availability SQL ledger across two geographic "Data Centers" (US-East and EU-Central). This project demonstrates a production-grade DevOps lifecycle—from automated multi-cluster bootstrapping to multi-region disaster recovery—all running locally via Docker.



## 🏗 The Architecture
The system simulates two independent regions, each with its own compute and storage, managed by a centralized GitOps control plane.

* **Cloud Emulation:** [LocalStack](https://localstack.cloud/) provides AWS S3 (Audit Logs) and SQS (Transaction Queues) with cross-region replication enabled.
* **Infrastructure as Code:** [Terraform](https://www.terraform.io/) provisions the virtual hardware using **Provider Aliases** to manage regional resource isolation.
* **Orchestration:** [Ansible](https://www.ansible.com/) automates the "Day 0" setup, spinning up two **K3d** Kubernetes clusters (`dc-us` and `dc-eu`) and bridging their local networks.
* **Continuous Delivery:** [ArgoCD](https://argo-cd.readthedocs.io/) manages the application state across both clusters using the **ApplicationSet** pattern.
* **Global Database:** [CockroachDB](https://www.cockroachlabs.com/) serves as the distributed ledger, maintaining Raft consensus between the US and EU nodes for zero-data-loss failover.



---

## 🛠 Tech Stack
| Category | Tooling |
| :--- | :--- |
| **Cloud Provider** | LocalStack (S3, SQS, IAM) |
| **Provisioning** | Terraform (Modules, Provider Aliases) |
| **Configuration** | Ansible (Roles, K3d-module) |
| **Orchestration** | Kubernetes (K3d), ArgoCD (ApplicationSets) |
| **Database** | CockroachDB (Multi-Region StatefulSet) |
| **Observability** | Prometheus & Grafana |

---

## 📂 Project Structure
```text
.
├── infrastructure/          # Terraform: Cloud resource provisioning
│   ├── modules/             # Reusable regional stack (S3, SQS, IAM)
│   ├── providers.tf         # Multi-region LocalStack endpoint config
│   └── main.tf              # Global resource orchestration
├── orchestration/           # Ansible: Cluster & Network bootstrapping
│   ├── roles/               # K3d setup, LoadBalancer config, & ArgoCD install
│   └── site.yml             # Master playbook to build the "World"
├── gitops/                  # ArgoCD: Declarative application state
│   ├── appsets/             # ApplicationSet generators (Multi-cluster logic)
│   └── charts/              # Helm charts for Ledger App & CockroachDB
└── scripts/                 # Chaos Engineering & Testing
    └── blast_radius.sh      # Chaos script to simulate regional outage

---

## 🚀 Potential Enhancements

This section outlines additional features and improvements that can be added to Project Atlas to make it more production-ready, secure, and feature-rich. Each enhancement includes a detailed explanation of why it's needed and how it benefits the system.

### 🔒 Security Enhancements

#### Network Policies
**Why:** Kubernetes Network Policies restrict pod-to-pod communication, preventing unauthorized access between services. In a multi-region setup, this is critical for preventing lateral movement if one service is compromised.

**What to add:**
- Kubernetes NetworkPolicy resources to define allowed traffic patterns
- Default deny-all policies with explicit allow rules
- Region-specific network isolation

#### RBAC Policies
**Why:** Role-Based Access Control (RBAC) ensures that only authorized users and services can perform specific actions. This is essential for production systems where multiple teams may need different levels of access.

**What to add:**
- Kubernetes RBAC roles and role bindings
- Service account permissions for applications
- Cluster-admin vs. namespace-scoped permissions

#### Secret Management
**Why:** Hardcoding secrets in code or config files is a security risk. A proper secret management system ensures secrets are encrypted, rotated, and audited.

**What to add:**
- HashiCorp Vault or Sealed Secrets for secret encryption
- Automatic secret rotation
- Secret injection at runtime (not in Git)

#### mTLS (Mutual TLS)
**Why:** Encrypts all communication between services, preventing man-in-the-middle attacks. Critical for multi-region systems where data travels across networks.

**What to add:**
- Service mesh (Istio/Linkerd) for automatic mTLS
- Certificate management and rotation
- Zero-trust networking model

#### Pod Security Standards
**Why:** Enforces security best practices at the pod level, preventing containers from running with excessive privileges or accessing host resources.

**What to add:**
- Pod Security Standards (restricted, baseline, privileged)
- Security context constraints
- Admission controllers for policy enforcement

#### Encryption at Rest
**Why:** Protects data even if storage is compromised. Essential for audit logs and transaction data that may contain sensitive information.

**What to add:**
- S3 bucket encryption (SSE-S3 or SSE-KMS)
- CockroachDB encryption at rest
- Encrypted persistent volumes

---

### 🔄 CI/CD & Automation

#### GitHub Actions / GitLab CI
**Why:** Automates testing, building, and deployment processes. Ensures code quality and reduces human error in deployments.

**What to add:**
- Automated Terraform validation and plan
- Container image building and scanning
- Automated testing before deployment
- Deployment pipelines for different environments

#### Automated Chaos Testing
**Why:** Validates system resilience automatically. Instead of manually running chaos experiments, they run in CI to catch regressions early.

**What to add:**
- Chaos experiments in CI pipeline
- Automated failover testing
- Performance regression detection

#### Automated Backups
**Why:** Ensures data can be recovered even if the distributed database fails. Automated backups reduce the risk of data loss and ensure compliance.

**What to add:**
- Scheduled CockroachDB backups to S3
- Backup verification and testing
- Point-in-time recovery procedures

#### Infrastructure Testing
**Why:** Validates Terraform code before applying changes. Prevents infrastructure misconfigurations that could cause outages.

**What to add:**
- Terratest for Terraform validation
- Infrastructure unit tests
- Compliance checking (e.g., no public S3 buckets)

#### Automated Failover Testing
**Why:** Regularly validates that disaster recovery procedures work. Catches issues before a real disaster occurs.

**What to add:**
- Scripts to simulate regional outages
- Automated validation of failover behavior
- Performance impact measurement during failover

---

### 📊 Advanced Observability

#### Distributed Tracing
**Why:** Tracks requests across multiple services and regions. Essential for debugging issues in a distributed system where a request may touch multiple services.

**What to add:**
- Jaeger or Zipkin for distributed tracing
- OpenTelemetry instrumentation
- Trace correlation across regions

#### Log Aggregation
**Why:** Centralizes logs from all services and regions. Makes it easier to debug issues and perform security audits.

**What to add:**
- ELK stack (Elasticsearch, Logstash, Kibana) or Loki
- Log parsing and indexing
- Log retention policies

#### Custom Dashboards
**Why:** Business-specific metrics help track system health from a business perspective, not just technical metrics.

**What to add:**
- Transaction volume dashboards
- Regional performance comparisons
- Business KPI tracking

#### Alerting Rules
**Why:** Proactively notifies teams of issues before they impact users. Critical for maintaining high availability.

**What to add:**
- Prometheus alerting rules
- AlertManager configuration
- Integration with PagerDuty, Slack, etc.

#### Application Performance Monitoring (APM)
**Why:** Provides deep insights into application performance, identifying bottlenecks and slow queries.

**What to add:**
- APM tools (Datadog, New Relic, or open-source alternatives)
- Application-level metrics
- Database query performance tracking

---

### ⚡ Performance & Optimization

#### Caching Layer
**Why:** Reduces database load and improves response times for frequently accessed data. Essential for high-traffic systems.

**What to add:**
- Redis for caching frequently accessed data
- Cache invalidation strategies
- Cache warming on startup

#### Connection Pooling
**Why:** Reduces database connection overhead. Improves performance and prevents connection exhaustion.

**What to add:**
- Database connection pooling (PgBouncer for CockroachDB)
- Connection pool monitoring
- Optimal pool size configuration

#### Query Optimization
**Why:** Slow queries can degrade system performance. Regular optimization ensures the system scales efficiently.

**What to add:**
- Query performance analysis
- Index optimization
- CockroachDB query plan analysis

#### Load Testing
**Why:** Validates system performance under load. Ensures the system can handle expected traffic volumes.

**What to add:**
- k6 or Locust for load testing
- Performance benchmarks
- Capacity planning based on load tests

#### Auto-Scaling
**Why:** Automatically adjusts resources based on demand. Reduces costs and ensures performance during traffic spikes.

**What to add:**
- Horizontal Pod Autoscaler (HPA) for applications
- Vertical Pod Autoscaler (VPA) for resource optimization
- Cluster autoscaling

---

### 🌐 Advanced Features

#### Service Mesh
**Why:** Provides advanced traffic management, security, and observability. Enables features like canary deployments, circuit breakers, and automatic retries.

**What to add:**
- Istio or Linkerd service mesh
- Traffic splitting and canary deployments
- Circuit breakers and retry policies

#### API Gateway
**Why:** Centralizes API management, authentication, rate limiting, and request routing. Essential for production APIs.

**What to add:**
- Kong or Ambassador API Gateway
- API versioning
- Request/response transformation

#### Event Streaming
**Why:** Enables event-driven architecture. Allows services to communicate asynchronously and decouple components.

**What to add:**
- Apache Kafka for event streaming
- Event sourcing patterns
- CQRS (Command Query Responsibility Segregation)

#### GraphQL API
**Why:** Provides a flexible API layer that allows clients to request exactly the data they need. Reduces over-fetching and under-fetching.

**What to add:**
- GraphQL server (Apollo, Hasura, or custom)
- GraphQL schema for transaction ledger
- Query optimization

#### Rate Limiting
**Why:** Protects APIs from abuse and ensures fair resource usage. Prevents a single client from overwhelming the system.

**What to add:**
- Rate limiting middleware
- Per-client rate limits
- Distributed rate limiting across regions

---

### 📚 Documentation & Operations

#### Runbooks
**Why:** Provides step-by-step procedures for common operational tasks. Reduces mean time to resolution (MTTR) during incidents.

**What to add:**
- Runbooks for common issues
- Troubleshooting guides
- Escalation procedures

#### Architecture Diagrams
**Why:** Visual documentation helps team members understand the system quickly. Essential for onboarding and knowledge sharing.

**What to add:**
- System architecture diagrams
- Data flow diagrams
- Network topology diagrams

#### API Documentation
**Why:** Makes it easy for developers to integrate with the system. Reduces support burden and improves developer experience.

**What to add:**
- OpenAPI/Swagger specifications
- Interactive API documentation
- Code examples and tutorials

#### Disaster Recovery Playbook
**Why:** Provides detailed procedures for recovering from disasters. Ensures teams know exactly what to do during a crisis.

**What to add:**
- Step-by-step recovery procedures
- Recovery time objectives (RTO) and recovery point objectives (RPO)
- Communication plans during incidents

#### Cost Monitoring
**Why:** Tracks infrastructure costs to optimize spending. Helps identify cost-saving opportunities.

**What to add:**
- Cost tracking dashboards
- Resource usage analysis
- Cost optimization recommendations

---

### 🧪 Testing & Quality

#### Integration Tests
**Why:** Validates that all components work together correctly. Catches integration issues before they reach production.

**What to add:**
- End-to-end integration tests
- Multi-region integration tests
- Database integration tests

#### Contract Testing
**Why:** Ensures service contracts remain compatible. Prevents breaking changes from propagating through the system.

**What to add:**
- Pact or similar contract testing tools
- API contract validation
- Consumer-driven contracts

#### Performance Benchmarks
**Why:** Tracks performance over time. Identifies performance regressions and validates optimizations.

**What to add:**
- Automated performance benchmarks
- Performance regression detection
- Historical performance tracking

#### Security Scanning
**Why:** Identifies vulnerabilities before they're exploited. Essential for maintaining a secure system.

**What to add:**
- Container image scanning
- Dependency vulnerability scanning
- Infrastructure security scanning

---

## 🎯 Recommended Priority Order

For teams implementing these enhancements, consider this priority order:

1. **Security Hardening** (Network Policies, Secrets Management) - Critical for production
2. **CI/CD Pipeline** (Automated Testing, Deployment) - Reduces risk and improves velocity
3. **Distributed Tracing** (Observability) - Essential for debugging distributed systems
4. **Load Testing** (Performance Validation) - Ensures system can handle expected load
5. **Service Mesh** (Advanced Traffic Management) - Enables advanced deployment patterns

---

## 📖 Additional Resources

- [CockroachDB Multi-Region Documentation](https://www.cockroachlabs.com/docs/stable/multiregion-overview.html)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/security/)
- [Terraform Best Practices](https://www.terraform.io/docs/cloud/guides/recommended-practices/)
- [ArgoCD Best Practices](https://argo-cd.readthedocs.io/en/stable/user-guide/best_practices/)
