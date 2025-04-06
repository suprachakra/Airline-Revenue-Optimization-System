# api_gateway_firewall.rego
package iaros.api_gateway.firewall

default allow = false

allow {
  input.method == "GET"
  input.path == "/healthcheck"
}

allow {
  input.headers["x-api-key"] == "secure-key" # Replace with dynamic key management
}

# Additional rules for OWASP Top 10 filtering.
