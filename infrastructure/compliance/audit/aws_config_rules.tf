// aws_config_rules.tf
// Terraform rules to enforce CIS Benchmarks on AWS resources.
resource "aws_config_config_rule" "required_tags" {
  name = "required-tags"
  source {
    owner             = "AWS"
    source_identifier = "REQUIRED_TAGS"
  }
}
