# Ansible Learning Guide for Project Atlas

This guide explains the Ansible concepts used in this project and what you need to implement next.

## 📚 Key Ansible Concepts

### 1. **Playbook** (`site.yml`)
- A YAML file that defines automation tasks
- Like a recipe: tells Ansible what to do and in what order
- Contains one or more "plays"

### 2. **Play**
- A section in a playbook that runs on specific hosts
- Our playbook has one play that runs on `localhost`
- Contains: hosts, tasks, variables

### 3. **Tasks**
- Individual actions to perform
- Each task uses a "module" (like a function)
- Examples: create directory, run command, install software

### 4. **Roles**
- Reusable sets of tasks
- Like functions in programming
- We have 3 roles:
  - `k3d-cluster` - Creates K3d Kubernetes clusters
  - `loadbalancer-config` - Sets up MetalLB for LoadBalancer services
  - `argocd-install` - Installs ArgoCD on clusters

### 5. **Modules**
- Built-in functions that do specific things
- Examples:
  - `file` - Manage files/directories
  - `command` - Run shell commands
  - `kubernetes.core.k8s` - Manage Kubernetes resources

### 6. **Variables**
- Dynamic values: `{{ variable_name }}`
- Can be defined in playbooks, roles, or passed in
- Example: `{{ cluster_name }}` becomes `dc-us` or `dc-eu`

### 7. **Idempotency**
- Safe to run multiple times
- If something already exists, Ansible won't recreate it
- This is why Ansible is great for infrastructure

## 🎯 What You Need to Implement Next

The `site.yml` playbook calls 3 roles that don't exist yet. You need to create:

### Role 1: `k3d-cluster` 
**Location:** `orchestration/roles/k3d-cluster/`

**What it does:** Creates a K3d Kubernetes cluster

**Files to create:**
1. `tasks/main.yml` - The actual tasks
2. `defaults/main.yml` - Default variables

**What the tasks should do:**
1. Check if k3d is installed
2. Create the K3d cluster with the given name and port mapping
3. Verify the cluster is running

**Hints:**
- Use `command` module to run `k3d cluster create`
- Use `which k3d` to check if k3d is installed
- Use `k3d cluster list` to verify cluster exists

### Role 2: `loadbalancer-config`
**Location:** `orchestration/roles/loadbalancer-config/`

**What it does:** Installs MetalLB for LoadBalancer support

**Files to create:**
1. `tasks/main.yml` - The actual tasks

**What the tasks should do:**
1. Create MetalLB namespace
2. Install MetalLB using Helm
3. Create IPAddressPool for LoadBalancer IPs

**Hints:**
- Use `kubernetes.core.k8s` module for Kubernetes resources
- Use `kubernetes.core.helm` module for Helm installs
- You'll need the `kubeconfig_path` variable to connect to the cluster

### Role 3: `argocd-install`
**Location:** `orchestration/roles/argocd-install/`

**What it does:** Installs ArgoCD on a Kubernetes cluster

**Files to create:**
1. `tasks/main.yml` - The actual tasks

**What the tasks should do:**
1. Create ArgoCD namespace
2. Install ArgoCD using Helm
3. Wait for ArgoCD to be ready
4. Get and display the admin password

**Hints:**
- Use `kubernetes.core.helm` to install ArgoCD
- Use `kubernetes.core.k8s_info` to wait for pods
- Use `kubernetes.core.k8s` to get the admin password secret

## 📖 Ansible Module Reference

### `file` Module
```yaml
- name: Create directory
  file:
    path: /path/to/dir
    state: directory  # or 'absent' to delete
```

### `command` Module
```yaml
- name: Run a command
  command: ls -la /tmp
  register: result  # Save output to variable
```

### `kubernetes.core.k8s` Module
```yaml
- name: Create namespace
  kubernetes.core.k8s:
    name: my-namespace
    api_version: v1
    kind: Namespace
    state: present
    kubeconfig: /path/to/kubeconfig
```

### `kubernetes.core.helm` Module
```yaml
- name: Install with Helm
  kubernetes.core.helm:
    name: my-app
    chart_ref: repo/chart-name
    release_namespace: my-namespace
    kubeconfig: /path/to/kubeconfig
```

## 🚀 Step-by-Step Implementation Order

1. **Start with `k3d-cluster` role** (simplest)
   - Just runs shell commands
   - Good for learning Ansible basics

2. **Then `loadbalancer-config` role**
   - Introduces Kubernetes modules
   - More complex but manageable

3. **Finally `argocd-install` role**
   - Most complex
   - Uses multiple modules
   - Includes waiting/checking logic

## 💡 Tips for Learning

1. **Test each role individually:**
   ```bash
   ansible-playbook -e "cluster_name=dc-us" \
     orchestration/roles/k3d-cluster/tasks/main.yml
   ```

2. **Use `--check` mode (dry run):**
   ```bash
   ansible-playbook site.yml --check
   ```

3. **Use `-v` for verbose output:**
   ```bash
   ansible-playbook site.yml -v
   ```

4. **Read Ansible docs:**
   - https://docs.ansible.com/ansible/latest/modules/modules_by_category.html

## 📝 Role Structure

Each role should follow this structure:
```
roles/
  role-name/
    tasks/
      main.yml      # Main tasks file
    defaults/
      main.yml      # Default variables (optional)
    handlers/
      main.yml      # Handlers (optional, for notifications)
```

## ✅ Checklist

- [ ] Create `k3d-cluster/tasks/main.yml`
- [ ] Create `k3d-cluster/defaults/main.yml`
- [ ] Create `loadbalancer-config/tasks/main.yml`
- [ ] Create `argocd-install/tasks/main.yml`
- [ ] Test each role individually
- [ ] Run full playbook: `ansible-playbook site.yml`

Good luck! Take it one role at a time. 🎉

