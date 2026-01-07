.PHONY: build run test clean docker-build docker-run deps

# Build the application
build:
	go build -o ledger-app ./main.go

# Run the application locally
run:
	go run ./main.go

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f ledger-app
	go clean

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build Docker image
docker-build:
	docker build -t ledger-app:latest .

# Run Docker container
docker-run:
	docker run --rm -p 8080:8080 \
		-e REGION=us-east-1 \
		-e AWS_ENDPOINT=http://host.docker.internal:4566 \
		-e S3_BUCKET=us-east-1-audit-logs \
		-e SQS_QUEUE=us-east-1-transaction-queue \
		-e COCKROACHDB_HOST=host.docker.internal \
		-e COCKROACHDB_PORT=26257 \
		ledger-app:latest

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Initialize go modules (first time setup)
init:
	go mod tidy

