// transit_gateway_config.tf
// Terraform configuration for multi-AZ connectivity via transit gateways.
resource "aws_ec2_transit_gateway" "iaros_tgw" {
  description = "IAROS Transit Gateway"
  amazon_side_asn = 64512
}

resource "aws_ec2_transit_gateway_vpc_attachment" "tgw_attachment" {
  transit_gateway_id = aws_ec2_transit_gateway.iaros_tgw.id
  vpc_id             = var.vpc_id
  subnet_ids         = var.subnet_ids
}
