module "us_east" {
    source = "./modules/regional-stack"
    region = "us-east-1"


    providers = {
        aws = aws.us_east
    }
}

module "eu_central" {
    source = "./modules/regional-stack"
    region = "eu-central-1"

    providers = {
        aws = aws.eu_central
    }
}