# data_retention.rego
package iaros.data

# Automated data deletion rules per retention policies.
default retention_allowed = false

retention_allowed {
  input.data_age <= 30 # days
}
