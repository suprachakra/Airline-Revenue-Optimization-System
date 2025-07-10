// main.tf - Core Cloud Resource Provisioning
// Provisions core resources such as Azure Resource Groups with redundancy.
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "iaros_rg" {
  name     = "iaros-resources"
  location = var.region
  tags     = { environment = var.environment }
}

// Additional resources (e.g., VMs, storage accounts) to be defined here.
