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