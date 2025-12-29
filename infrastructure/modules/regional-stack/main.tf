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

resource "aws_iam_role" "ledger_app_role" {
    name = "${var.region}-ledge-app-role"

    assume_role_policy = jsonencode({
        Version = "2012-10-17"
        Statement = [{
            Action = "sts:AssumeRole"
            Effect = "Allow"
            Principal = {
                Service = "ec2.amazonaws.com"
            }
        }]
    })
}

resource "aws_iam_role_policy" "ledger_app_policy" {
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