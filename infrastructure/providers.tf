terraform {
    required_version = ">= 1.0"

    required_providers {
        aws = {
            source = "hashicorp/aws"
            version = "~> 5.0"
        }
    }
}

# US-East Provider
provider "aws" {
    alias = "us_east"
    region = "us-east-1"

    endpoints {
        s3 = "http://localhost:4566"
        sqs = "http://localhost:4566"
        iam = "http://localhost:4566"
    }

    skip_credentials_validation = true
    skip_metadata_api_check = true
    skip_region_validation = true
    access_key = "test"
    secret_key = "test"
}

provider "aws" {
    alias = "eu_central"
    region = "eu-central-1"

    endpoints {
        s3 = "http://localhost:4567"
        sqs = "http://localhost:4567"
        iam = "http://localhost:4567"
    }

    skip_credentials_validation = true
    skip_metadata_api_check = true
    skip_region_validation = true
    access_key = "test"
    secret_key = "test"
}