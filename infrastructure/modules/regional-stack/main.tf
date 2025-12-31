resource "aws_s3_bucket" "audit_logs" {
    provider = aws
    bucket = "${var.region}-audit-logs"

    tags = {
        Region = var.region
        Purpose = var.audit_logs_tag
    }
    
    # Force path-style addressing for LocalStack compatibility
    force_destroy = false
}

resource "aws_s3_bucket_versioning" "audit_logs" {
    provider = aws
    bucket = aws_s3_bucket.audit_logs.id

    versioning_configuration {
        status = var.s3_versioning_enabled ? "Enabled" : "Disabled"
    }
}

resource "aws_sqs_queue" "transaction_queue" {
    provider = aws
    name = "${var.region}-transaction-queue"

    visibility_timeout_seconds = var.sqs_visibility_timeout_seconds
    message_retention_seconds = var.sqs_message_retention_seconds

    tags = {
        Region = var.region
        Purpose = var.transaction_queue_tag
    }
}

resource "aws_iam_role" "ledger_app_role" {
    provider = aws
    name = "${var.region}-ledger-app-role"

    assume_role_policy = jsonencode({
        Version = "2012-10-17"
        Statement = [{
            Action = "sts:AssumeRole"
            Effect = "Allow"
            Principal = {
                Service = var.iam_service_principal
            }
        }]
    })
}

resource "aws_iam_role_policy" "ledger_app_policy" {
    provider = aws
    name = "${var.region}-ledger-app-policy"
    role = aws_iam_role.ledger_app_role.id

    policy = jsonencode({
        Version = "2012-10-17"
        Statement = [{
            Effect = "Allow"
            Action = [
                "s3:PutObject",
                "s3:GetObject",
                "s3:ListBucket"
            ]
            Resource = [
                aws_s3_bucket.audit_logs.arn,
                "${aws_s3_bucket.audit_logs.arn}/*"
            ]
        },
        {
            Effect = "Allow"
            Action = [
                "sqs:SendMessage",
                "sqs:ReceiveMessage",
                "sqs:DeleteMessage",
                "sqs:GetQueueAttributes"
            ]
            Resource = [
                aws_sqs_queue.transaction_queue.arn
            ]
        }]
    })
}