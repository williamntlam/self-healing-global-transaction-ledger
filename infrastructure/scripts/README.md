# Infrastructure Scripts

Helper scripts for managing Terraform and LocalStack infrastructure.

## Scripts

### `terraform.sh`
Main helper script for Terraform operations with LocalStack management.

**Usage:**
```bash
./scripts/terraform.sh init      # Start LocalStack and initialize Terraform
./scripts/terraform.sh plan      # Plan changes (auto-starts LocalStack if needed)
./scripts/terraform.sh apply     # Apply changes (auto-starts LocalStack if needed)
./scripts/terraform.sh destroy   # Destroy resources (auto-starts LocalStack if needed)
./scripts/terraform.sh stop     # Stop LocalStack containers
./scripts/terraform.sh status    # Check LocalStack and Terraform status
```

### `verify.sh`
Verify that all resources were created successfully in LocalStack (without AWS CLI).

**Usage:**
```bash
./scripts/verify.sh
```

Shows:
- Terraform outputs
- LocalStack health status
- S3 buckets in both regions
- Recent resource creation logs
- Resources in Terraform state

### `destroy.sh`
Interactive script to destroy all provisioned resources.

**Usage:**
```bash
./scripts/destroy.sh
```

Features:
- Lists resources before destroying
- Asks for confirmation
- Optionally stops LocalStack
- Optionally removes state files

### `destroy-all.sh`
Complete cleanup script that destroys everything.

**Usage:**
```bash
./scripts/destroy-all.sh
```

Features:
- Destroys all Terraform resources (auto-approve)
- Stops LocalStack containers
- Removes Terraform state files
- Optionally removes LocalStack data volumes

## Running Scripts

All scripts can be run from anywhere, but they will automatically change to the `infrastructure/` directory:

```bash
# From project root
./infrastructure/scripts/terraform.sh plan

# From infrastructure directory
./scripts/terraform.sh plan

# From scripts directory
./terraform.sh plan
```

## Notes

- All scripts automatically handle LocalStack startup if needed
- Scripts change to the infrastructure directory before running Terraform commands
- Scripts use relative paths, so they work regardless of where you run them from

