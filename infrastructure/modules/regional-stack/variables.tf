variable "region" {
  description = "AWS region name"
  type        = string
}

variable "sqs_visibility_timeout_seconds" {
  description = "SQS queue visibility timeout in seconds"
  type        = number
  default     = 30
}

variable "sqs_message_retention_seconds" {
  description = "SQS message retention in seconds (14 days default)"
  type        = number
  default     = 1209600
}

variable "s3_versioning_enabled" {
  description = "Enable S3 bucket versioning"
  type        = bool
  default     = true
}

variable "audit_logs_tag" {
  description = "Tag value for audit logs bucket purpose"
  type        = string
  default     = "AuditLogs"
}

variable "transaction_queue_tag" {
  description = "Tag value for transaction queue purpose"
  type        = string
  default     = "TransactionQueue"
}

variable "iam_service_principal" {
  description = "IAM service principal for assume role policy"
  type        = string
  default     = "ec2.amazonaws.com"
}

