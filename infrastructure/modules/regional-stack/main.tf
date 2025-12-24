resource "aws_s3_bucket" "audit_logs" {
    bucket = "${var.region}-audit-logs"

    tags = {
        Region = var.region
        Purpose = "AuditLogs"
    }
}