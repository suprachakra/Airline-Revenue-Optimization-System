// bare_metal_provisioning.tf
// Terraform configuration for provisioning bare metal servers using PXE boot.
resource "null_resource" "bare_metal" {
  provisioner "local-exec" {
    command = "bash setup_bare_metal.sh"
  }
}
