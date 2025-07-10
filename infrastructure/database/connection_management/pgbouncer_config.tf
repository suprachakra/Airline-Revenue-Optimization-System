// pgbouncer_config.tf
// Terraform configuration for PostgreSQL connection pooling using PgBouncer.
resource "aws_instance" "pgbouncer" {
  ami           = var.pgbouncer_ami
  instance_type = var.instance_type
  subnet_id     = var.subnet_id
  tags = {
    Name = "pgbouncer-instance"
  }
}

// Additional configuration for PgBouncer can be managed via configuration management tools.
