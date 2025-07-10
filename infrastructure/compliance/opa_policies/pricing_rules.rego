# pricing_rules.rego
package iaros.pricing

default allow = false

# Enforce ATPCO compliance rules for pricing offers.
allow {
  input.offer.discount <= input.offer.max_discount
  input.offer.ndc_compliant == true
}
