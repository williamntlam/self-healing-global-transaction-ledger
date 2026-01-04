# Secrets Management Guide

## Overview

This role creates Kubernetes secrets for the ledger application. **Never commit secrets to Git in plain text!**

## Quick Start

### Option 1: Using Ansible Vault (Recommended)

1. **Encrypt your secrets:**
```bash
# Encrypt a password
ansible-vault encrypt_string 'my-secret-password' --name 'cockroachdb_password'

# This outputs:
# cockroachdb_password: !vault |
#   $ANSIBLE_VAULT;1.1;AES256
#   663864396539663161616462...
```

2. **Create a vault file:**
```bash
# Create orchestration/group_vars/all/vault.yml
ansible-vault create orchestration/group_vars/all/vault.yml
```

3. **Add encrypted secrets:**
```yaml
# orchestration/group_vars/all/vault.yml
cockroachdb_username: admin
cockroachdb_password: !vault |
  $ANSIBLE_VAULT;1.1;AES256
  663864396539663161616462...

aws_access_key_id: test
aws_secret_access_key: test
```

4. **Run the playbook:**
```bash
ansible-playbook site.yml --ask-vault-pass
```

### Option 2: Using Environment Variables

1. **Set environment variables:**
```bash
export COCKROACHDB_USERNAME=admin
export COCKROACHDB_PASSWORD=my-secret-password
```

2. **Pass to Ansible:**
```bash
ansible-playbook site.yml \
  -e "cockroachdb_username=$COCKROACHDB_USERNAME" \
  -e "cockroachdb_password=$COCKROACHDB_PASSWORD"
```

### Option 3: Using kubectl (Manual)

```bash
# Create secret directly
kubectl create secret generic cockroachdb-credentials \
  --from-literal=username=admin \
  --from-literal=password=my-secret-password \
  --namespace=default \
  --context=k3d-dc-us
```

## Using Secrets in Your Application

Once secrets are created, reference them in your Helm charts:

```yaml
# gitops/charts/ledger-app/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: ledger-app
        env:
        # Reference the secret
        - name: DB_USERNAME
          valueFrom:
            secretKeyRef:
              name: cockroachdb-credentials
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: cockroachdb-credentials
              key: password
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: access-key-id
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: secret-access-key
```

## Security Best Practices

1. ✅ **DO:**
   - Use Ansible Vault for encryption
   - Store vault password in a secure location (password manager)
   - Use external secret management (HashiCorp Vault, AWS Secrets Manager)
   - Rotate secrets regularly
   - Use least-privilege access

2. ❌ **DON'T:**
   - Commit secrets to Git in plain text
   - Share secrets via email or chat
   - Use the same secrets across environments
   - Hardcode secrets in application code

## For LocalStack (Development)

Since LocalStack uses test credentials, you can use:
- `access_key: test`
- `secret_key: test`

These are safe to use in development but should never be used in production.

