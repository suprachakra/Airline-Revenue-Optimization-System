// security_groups.tf
// Define ACLs for internal/external traffic.
resource "aws_security_group" "internal_sg" {
  name        = "iaros-internal-sg"
  description = "Internal service ACL for IAROS"
  vpc_id      = var.vpc_id

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["10.0.0.0/8"]
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
