// waf_rules.tf
// Terraform configuration for API Gateway WAF rules enforcing OWASP Top 10.
resource "aws_wafregional_web_acl" "iaros_waf" {
  name        = "iaros-api-waf"
  metric_name = "IarosApiWaf"
  default_action {
    type = "BLOCK"
  }
  rules {
    action {
      type = "ALLOW"
    }
    priority = 1
    rule_id  = aws_wafregional_rule.allowed_rule.id
    type     = "REGULAR"
  }
}
