// aws_us_east_to_eu_central.tf
// Terraform configuration for multi-cloud failover between AWS US East and EU Central.
provider "aws" {
  region = "us-east-1"
}

provider "aws" {
  alias  = "eu"
  region = "eu-central-1"
}

resource "aws_route53_health_check" "failover_check" {
  fqdn              = "api.iaros.ai"
  type              = "HTTP"
  resource_path     = "/healthcheck"
  failure_threshold = 3
  request_interval  = 30
}

// Additional resources for failover routing can be defined here.
