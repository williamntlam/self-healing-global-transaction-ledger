resource "aws_s3_bucket" "audit_logs" {
    bucket = "${var.region}-audit-logs"

    tags = {
        Region = var.region
        Purpose = "AuditLogs"
    }
}

resource "aws_s3_bucket_versioning" "audit_logs" {
    bucket = aws_s3_bucket.audit_logs.id

    versioning_configuration {
        status = "Enabled"
    }
}

resource "aws_sqs_queue" "transaction_queue" {
    name = "${var.region}-transaction-queue"

    visibility_timeout_seconds = 30
    message_retention_seconds = 1209600

    tags = {
        Region = var.region
        Purpose = "TransactionQueue"
    }
}