# CockroachDB Unified Cluster Architecture

## How Clusters Are Combined

The US and EU CockroachDB clusters form **one unified cluster** through CockroachDB's cluster membership and Raft consensus protocol. It's **not just network connectivity** - it's actual cluster membership where all nodes participate in the same consensus group.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│           Unified CockroachDB Cluster                        │
│         (One Raft Consensus Group)                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────┐      ┌──────────────────────┐   │
│  │   US-East Region     │      │  EU-Central Region   │   │
│  │  (k3d-dc-us)        │      │  (k3d-dc-eu)         │   │
│  │                      │      │                      │   │
│  │  cockroachdb-0       │◄────►│  cockroachdb-0       │   │
│  │  cockroachdb-1       │      │  cockroachdb-1       │   │
│  │  cockroachdb-2       │      │  cockroachdb-2       │   │
│  │                      │      │                      │   │
│  │  Locality:           │      │  Locality:            │   │
│  │  region=us-east-1    │      │  region=eu-central-1│   │
│  └──────────────────────┘      └──────────────────────┘   │
│           │                              │                  │
│           └──────────┬───────────────────┘                  │
│                      │                                      │
│           ┌──────────▼──────────┐                          │
│           │  Raft Consensus     │                          │
│           │  (All 6 nodes)      │                          │
│           └─────────────────────┘                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## How It Works

### 1. Cluster Membership (Not Just Networking)

When a CockroachDB node starts with the `--join` flag, it:

1. **Contacts join addresses** - Connects to the specified nodes
2. **Discovers cluster** - Gets the full list of cluster members
3. **Joins consensus group** - Becomes part of the Raft consensus protocol
4. **Registers with cluster** - Other nodes learn about the new member

**Example:**
```bash
# US node starting up
cockroach start \
  --join=192.168.1.10:26257,192.168.1.20:26257 \
  --locality=region=us-east-1

# This node will:
# 1. Connect to 192.168.1.10 (EU LoadBalancer)
# 2. Discover all 6 nodes in the cluster
# 3. Join the Raft consensus group
# 4. Start participating in transactions
```

### 2. The `--join` Flag

The `--join` flag tells a node **which nodes to contact** to discover the cluster. It doesn't need all nodes - just enough to find the cluster.

**Current Configuration:**
```yaml
# US nodes join with:
--join=cockroachdb-0.cockroachdb:26257,cockroachdb-1.cockroachdb:26257,cockroachdb-2.cockroachdb:26257,192.168.1.20:26257
#                                                                    ↑
#                                                          EU LoadBalancer IP

# EU nodes join with:
--join=cockroachdb-0.cockroachdb:26257,cockroachdb-1.cockroachdb:26257,cockroachdb-2.cockroachdb:26257,192.168.1.10:26257
#                                                                    ↑
#                                                          US LoadBalancer IP
```

### 3. Node Discovery Process

```
New Node Starting
    │
    ├─► Contacts --join addresses
    │   │
    │   ├─► Local nodes (cockroachdb-0, cockroachdb-1, cockroachdb-2)
    │   └─► Remote nodes (via LoadBalancer IP)
    │
    ├─► Receives cluster membership list
    │   └─► Discovers all 6 nodes
    │
    ├─► Joins Raft consensus group
    │   └─► Participates in leader election
    │
    └─► Ready to serve requests
```

### 4. Raft Consensus Across Regions

All 6 nodes participate in **one Raft consensus group**. **Raft is the core mechanism that ensures transaction consistency and durability** - it's what makes transactions ACID-compliant.

```
┌─────────────────────────────────────────┐
│      Raft Consensus Group               │
│   (Transaction Proprietor)               │
├─────────────────────────────────────────┤
│                                         │
│  Leader: cockroachdb-us-0              │
│  (Proposes transactions)                │
│                                         │
│  Followers:                             │
│  ├─ cockroachdb-us-1                   │
│  ├─ cockroachdb-us-2                   │
│  ├─ cockroachdb-eu-0                   │
│  ├─ cockroachdb-eu-1                   │
│  └─ cockroachdb-eu-2                   │
│  (Vote on transactions)                │
│                                         │
│  All nodes vote on:                     │
│  - Transaction commits ⭐ PRIMARY       │
│  - Data replication                     │
│  - Leader election                      │
│  - Cluster membership changes           │
│                                         │
└─────────────────────────────────────────┘
```

**Key Points:**
- **Raft is the transaction proprietor** - Every transaction must go through Raft consensus
- **One consensus group** - All 6 nodes participate in transaction decisions
- **Quorum required** - Need majority (4 out of 6) to commit a transaction
- **Cross-region voting** - US and EU nodes vote together on every transaction
- **Leader can be in any region** - Raft elects leader based on availability
- **ACID guarantees** - Raft ensures Atomicity, Consistency, Isolation, and Durability

### 5. Network Connectivity Requirements

For nodes to join the cluster, they need:

1. **Network reachability** - Can connect to join addresses
2. **Port access** - Port 26257 (gRPC) must be accessible
3. **LoadBalancer services** - For cross-cluster communication

**Current Setup:**
```
US Cluster (k3d-dc-us)
    │
    ├─► cockroachdb-public (LoadBalancer)
    │   └─► External IP: 192.168.1.10
    │
    └─► EU nodes connect via: 192.168.1.10:26257

EU Cluster (k3d-dc-eu)
    │
    ├─► cockroachdb-public (LoadBalancer)
    │   └─► External IP: 192.168.1.20
    │
    └─► US nodes connect via: 192.168.1.20:26257
```

## Data Flow Example

### Creating a Transaction in US Region

```
1. Client → ledger-app (US)
   │
2. ledger-app → cockroachdb-us-0 (local)
   │
3. cockroachdb-us-0 (Leader)
   │
   ├─► Proposes transaction to Raft
   │
   ├─► Sends to followers:
   │   ├─► cockroachdb-us-1 ✅
   │   ├─► cockroachdb-us-2 ✅
   │   ├─► cockroachdb-eu-0 ✅ (via network)
   │   ├─► cockroachdb-eu-1 ✅ (via network)
   │   └─► cockroachdb-eu-2 ✅ (via network)
   │
   ├─► Receives votes (quorum: 4/6)
   │
   └─► Commits transaction
       └─► Data stored in US region (REGIONAL BY ROW)
```

### Reading from EU Region

```
1. Client → ledger-app (EU)
   │
2. ledger-app → cockroachdb-eu-0 (local)
   │
3. cockroachdb-eu-0
   │
   ├─► Reads from local replica (if available)
   │   OR
   └─► Fetches from US region (if not local)
       └─► Returns data to client
```

## Key Differences: Network vs Cluster Membership

### Just Network Connectivity (What We DON'T Have)
```
US Cluster          EU Cluster
    │                   │
    └─── Network ────────┘
    
❌ Separate databases
❌ No shared consensus
❌ Manual replication needed
```

### Cluster Membership (What We HAVE)
```
US Cluster          EU Cluster
    │                   │
    └─── Network ────────┘
         │
    ┌────▼────┐
    │  Raft   │ ← One consensus group
    │ Consensus│
    └─────────┘
    
✅ One unified database
✅ Shared consensus
✅ Automatic replication
```

## Verification

### Check Cluster Membership

```bash
# Connect to any node
kubectl --context k3d-dc-us exec -it cockroachdb-0 -- cockroach sql --insecure

# Should show all 6 nodes
SHOW NODES;

# Output:
# id | address | locality
# ---|---------|----------
# 1  | ...     | region=us-east-1
# 2  | ...     | region=us-east-1
# 3  | ...     | region=us-east-1
# 4  | ...     | region=eu-central-1
# 5  | ...     | region=eu-central-1
# 6  | ...     | region=eu-central-1
```

### Check Raft Consensus

```bash
# Check cluster status
SHOW CLUSTER SETTING cluster.organization;

# Check regions
SHOW REGIONS;

# Check ranges (data distribution)
SHOW RANGES FROM TABLE transactions;
```

## Summary

**It's NOT just two separate clusters connected via networks.**

It's **one unified CockroachDB cluster** where:
- All 6 nodes participate in the same Raft consensus group
- Transactions require votes from nodes in both regions
- Data is partitioned by region but accessible from both
- Network connectivity enables cluster membership, but cluster membership is what makes it unified

**The `--join` flag is the key** - it tells nodes how to discover and join the unified cluster, not just how to reach other nodes on the network.
