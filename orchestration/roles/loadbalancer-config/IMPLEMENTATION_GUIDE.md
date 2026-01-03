# Step-by-Step Guide: Implementing loadbalancer-config Role

## What is MetalLB?

MetalLB is a load balancer implementation for bare-metal Kubernetes clusters. Since K3d runs locally (not in a cloud), it doesn't have a built-in LoadBalancer. MetalLB provides this functionality.

**Why we need it:**
- ArgoCD needs a LoadBalancer service to be accessible
- Applications will need LoadBalancer services
- Without it, LoadBalancer services stay in "Pending" state

---

## Step 1: Understand the Role Structure

Create this structure:
```
orchestration/roles/loadbalancer-config/
└── tasks/
    └── main.yml
```

**Note:** This role doesn't need `defaults/main.yml` because all variables come from `site.yml`.

---

## Step 2: Understand the Variables

From `site.yml`, this role receives:
- `kubeconfig_path`: Path to the cluster's kubeconfig file
- `cluster`: Cluster name (for reference/logging)

You'll use `kubeconfig_path` to connect to the Kubernetes cluster.

---

## Step 3: Task 1 - Create MetalLB Namespace

**What to do:**
Create a Kubernetes namespace called `metallb-system`.

**Module to use:** `kubernetes.core.k8s`

**Example structure:**
```yaml
- name: Create MetalLB namespace
  kubernetes.core.k8s:
    name: metallb-system
    api_version: v1
    kind: Namespace
    state: present
    kubeconfig: "{{ kubeconfig_path }}"
```

**Explanation:**
- `name`: Namespace name
- `api_version`: Kubernetes API version (v1 for namespaces)
- `kind`: Resource type (Namespace)
- `state: present`: Ensure it exists (create if missing)
- `kubeconfig`: Path to kubeconfig file (from site.yml)

**Key concept:** `state: present` means "make sure it exists" - idempotent!

---

## Step 4: Task 2 - Install MetalLB using Helm

**What to do:**
Install MetalLB using Helm chart.

**Module to use:** `kubernetes.core.helm`

**Helm chart info:**
- Repository: `https://metallb.github.io/metallb`
- Chart name: `metallb/metallb`
- Namespace: `metallb-system` (the one you just created)

**Example structure:**
```yaml
- name: Add MetalLB Helm repository
  kubernetes.core.helm:
    name: metallb
    repo_url: https://metallb.github.io/metallb
    chart_ref: metallb/metallb
    release_namespace: metallb-system
    kubeconfig: "{{ kubeconfig_path }}"
    create_namespace: false  # We already created it
```

**Explanation:**
- `name`: Release name (what Helm calls this installation)
- `repo_url`: Helm repository URL
- `chart_ref`: Chart name (repo/chart format)
- `release_namespace`: Where to install
- `create_namespace: false`: Don't create namespace (we did it manually)
- `kubeconfig`: Path to kubeconfig

**Note:** You might need to add the repo first, then install. Check Helm module docs.

---

## Step 5: Task 3 - Wait for MetalLB to be Ready

**What to do:**
Wait for MetalLB pods to be running before proceeding.

**Module to use:** `kubernetes.core.k8s_info` with `until` loop

**Example structure:**
```yaml
- name: Wait for MetalLB controller to be ready
  kubernetes.core.k8s_info:
    api_version: v1
    kind: Pod
    namespace: metallb-system
    label_selectors:
      - app.kubernetes.io/name=metallb
      - component=controller
    kubeconfig: "{{ kubeconfig_path }}"
  register: metallb_pods
  until: >
    metallb_pods.resources | length > 0 and
    metallb_pods.resources[0].status.phase == "Running"
  retries: 30
  delay: 10
```

**Explanation:**
- `k8s_info`: Gets information about Kubernetes resources
- `label_selectors`: Find pods with specific labels
- `register`: Save result to variable
- `until`: Keep retrying until condition is true
- `retries`: Maximum number of attempts
- `delay`: Seconds between retries

**Key concept:** `until` loop keeps checking until pods are running.

---

## Step 6: Task 4 - Create IPAddressPool

**What to do:**
Create an IPAddressPool resource that tells MetalLB which IPs to use.

**Module to use:** `kubernetes.core.k8s`

**Resource info:**
- API: `metallb.io/v1beta1`
- Kind: `IPAddressPool`
- Namespace: `metallb-system`

**Example structure:**
```yaml
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

**Explanation:**
- `definition`: The Kubernetes resource definition (YAML)
- `apiVersion`: MetalLB API version
- `kind`: Resource type
- `metadata`: Name and namespace
- `spec.addresses`: IP range for LoadBalancer services
- IP range `172.18.0.100-172.18.0.200` is safe for Docker networks

**Key concept:** This tells MetalLB which IPs it can assign to LoadBalancer services.

---

## Step 7: Task 5 - Create L2Advertisement (Optional but Recommended)

**What to do:**
Create an L2Advertisement to advertise the IP pool.

**Why:** MetalLB needs this to actually use the IPAddressPool.

**Example structure:**
```yaml
- name: Create L2Advertisement
  kubernetes.core.k8s:
    definition:
      apiVersion: metallb.io/v1beta1
      kind: L2Advertisement
      metadata:
        name: default
        namespace: metallb-system
      spec:
        ipAddressPools:
        - default-pool
    kubeconfig: "{{ kubeconfig_path }}"
```

**Explanation:**
- Links the L2Advertisement to the IPAddressPool
- Tells MetalLB to use L2 mode (Layer 2 networking)

---

## Step 8: Task 6 - Display Success Message

**What to do:**
Show a message confirming MetalLB is installed.

**Module to use:** `debug`

**Example:**
```yaml
- name: Display MetalLB installation status
  debug:
    msg: >
      ✅ MetalLB installed successfully on {{ cluster }}!
      IP Pool: 172.18.0.100-172.18.0.200
```

---

## Complete File Structure

Your `tasks/main.yml` should have tasks in this order:

1. Create namespace
2. Install MetalLB (Helm)
3. Wait for MetalLB to be ready
4. Create IPAddressPool
5. Create L2Advertisement (optional)
6. Display success message

---

## Testing Your Role

### Test 1: Dry Run
```bash
cd orchestration
ansible-playbook site.yml --check
```

### Test 2: Test Just This Role
```bash
cd orchestration
ansible-playbook -e "kubeconfig_path=~/.kube/config cluster=test" \
  roles/loadbalancer-config/tasks/main.yml
```

### Test 3: Verify in Kubernetes
After running, check:
```bash
kubectl --context k3d-dc-us get pods -n metallb-system
kubectl --context k3d-dc-us get ipaddresspool -n metallb-system
```

---

## Common Issues & Solutions

### Issue: "Helm chart not found"
**Solution:** Make sure you add the Helm repo first, or use `chart_ref` with full path.

### Issue: "Namespace already exists"
**Solution:** That's fine! `state: present` is idempotent - it won't error.

### Issue: "Pods not ready"
**Solution:** Increase `retries` or `delay` in the wait task.

### Issue: "IPAddressPool not working"
**Solution:** Make sure L2Advertisement is created and references the pool.

---

## Key Ansible Modules Used

### `kubernetes.core.k8s`
- For creating Kubernetes resources
- Uses `definition:` for resource YAML
- Requires `kubeconfig:` parameter

### `kubernetes.core.helm`
- For installing Helm charts
- Handles repository management
- Requires `kubeconfig:` parameter

### `kubernetes.core.k8s_info`
- For querying Kubernetes resources
- Used with `until` loops for waiting
- Returns resource information

---

## Helpful Resources

1. **Ansible Kubernetes Module Docs:**
   https://docs.ansible.com/ansible/latest/collections/kubernetes/core/k8s_module.html

2. **Ansible Helm Module Docs:**
   https://docs.ansible.com/ansible/latest/collections/kubernetes/core/helm_module.html

3. **MetalLB Documentation:**
   https://metallb.universe.tf/

4. **MetalLB Installation:**
   https://metallb.universe.tf/installation/

---

## Checklist

- [ ] Create `tasks/main.yml` file
- [ ] Task 1: Create namespace
- [ ] Task 2: Install MetalLB with Helm
- [ ] Task 3: Wait for MetalLB pods
- [ ] Task 4: Create IPAddressPool
- [ ] Task 5: Create L2Advertisement (optional)
- [ ] Task 6: Display success message
- [ ] Test with dry run
- [ ] Test with actual execution
- [ ] Verify in Kubernetes

---

## Tips

1. **Start simple:** Get namespace creation working first
2. **Test incrementally:** Add one task at a time, test, then add next
3. **Use `--check` mode:** Test without making changes
4. **Check logs:** If something fails, check Kubernetes events:
   ```bash
   kubectl get events -n metallb-system
   ```
5. **Idempotency:** Make sure tasks are idempotent (safe to run multiple times)

Good luck! Take it one task at a time. 🚀

