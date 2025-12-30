# US-East outputs
output "us_east_s3_bucket" {
    description = "S3 bucket name for US-East audit logs"
    value        = module.us_east.s3_bucket_name
}

output "us_east_sqs_queue" {
    description = "SQS queue URL for US-East"
    value       = module.us_east.sqs_queue_url
}

output "us_east_iam_role_arn" {
    description = "IAM role ARN for US-East"
    value       = module.us_east.iam_role_arn
}

# EU-Central outputs
output "eu_central_s3_bucket" {
    description = "S3 bucket name for EU-Central audit logs"
    value       = module.eu_central.s3_bucket_name
}

output "eu_central_sqs_queue" {
    description = "SQS queue URL for EU-Central"
    value       = module.eu_central.sqs_queue_url
}

output "eu_central_iam_role_arn" {
    description = "IAM role ARN for EU-Central"
    value       = module.eu_central.iam_role_arn
}