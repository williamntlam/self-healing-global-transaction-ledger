# Ledger Application

A Go-based transaction ledger application that integrates with CockroachDB, AWS S3 (via LocalStack), and AWS SQS (via LocalStack).

## Features

- **RESTful API** for transaction management
- **CockroachDB integration** for persistent storage
- **S3 audit logging** for compliance and auditing
- **SQS message queue** for asynchronous processing
- **Health checks** for Kubernetes liveness/readiness probes
- **Multi-region support** with region-specific configuration

## API Endpoints

### Health Checks
- `GET /health` - Comprehensive health check (database, S3, SQS)
- `GET /ready` - Readiness probe (checks database connectivity)
- `GET /live` - Liveness probe (always returns OK)

### Transactions
- `POST /transactions` - Create a new transaction
- `GET /transactions` - List transactions (with pagination)
- `GET /transactions/{id}` - Get a specific transaction
- `GET /stats` - Get transaction statistics

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_PORT` | HTTP server port | `8080` |
| `REGION` | Region identifier | `us-east-1` |
| `AWS_REGION` | AWS region | `us-east-1` |
| `AWS_ENDPOINT` | LocalStack endpoint | `http://localhost:4566` |
| `S3_BUCKET` | S3 bucket name | `us-east-1-audit-logs` |
| `SQS_QUEUE` | SQS queue name | `us-east-1-transaction-queue` |
| `COCKROACHDB_HOST` | CockroachDB host | `cockroachdb-public` |
| `COCKROACHDB_PORT` | CockroachDB port | `26257` |
| `COCKROACHDB_DATABASE` | Database name | `ledger` |
| `COCKROACHDB_USER` | Database user | `root` |
| `COCKROACHDB_PASSWORD` | Database password | (empty) |

## Building

```bash
# Build the Docker image
docker build -t ledger-app:latest .

# Or build locally
go build -o ledger-app ./main.go
```

## Running Locally

```bash
# Set environment variables
export REGION=us-east-1
export AWS_ENDPOINT=http://localhost:4566
export S3_BUCKET=us-east-1-audit-logs
export SQS_QUEUE=us-east-1-transaction-queue
export COCKROACHDB_HOST=localhost
export COCKROACHDB_PORT=26257

# Run the application
./ledger-app
```

## Testing

```bash
# Create a transaction
curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "from_account": "account-1",
    "to_account": "account-2",
    "amount": "100.50"
  }'

# Get a transaction
curl http://localhost:8080/transactions/{transaction-id}

# List transactions
curl http://localhost:8080/transactions?limit=10&offset=0

# Get statistics
curl http://localhost:8080/stats

# Health check
curl http://localhost:8080/health
```

## Database Schema

The application expects the following table structure in CockroachDB:

```sql
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region STRING NOT NULL,
    amount DECIMAL(19,2) NOT NULL,
    from_account STRING NOT NULL,
    to_account STRING NOT NULL,
    status STRING DEFAULT 'pending',
    timestamp TIMESTAMP DEFAULT now()
) LOCALITY REGIONAL BY ROW AS region;
```

**Note:** The `amount` field uses `DECIMAL(19,2)` for precise financial calculations. The Go application uses the `shopspring/decimal` library which automatically handles conversion to/from the database.

## Architecture

- **main.go**: Application entry point, server setup, graceful shutdown
- **internal/database/**: Database connection and transaction operations
- **internal/s3/**: S3 client for audit log storage
- **internal/sqs/**: SQS client for message queue operations
- **internal/api/**: HTTP handlers and routing
- **internal/models/**: Data models and structures

## Development

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Format code
go fmt ./...

# Lint code
golangci-lint run
```

