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
