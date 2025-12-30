output "s3_bucket_name" {
  description = "S3 bucket name for audit logs"
  value       = aws_s3_bucket.audit_logs.id
}

output "sqs_queue_url" {
  description = "SQS queue URL"
  value       = aws_sqs_queue.transaction_queue.url
}

output "iam_role_arn" {
  description = "IAM role ARN"
  value       = aws_iam_role.ledger_app_role.arn
}