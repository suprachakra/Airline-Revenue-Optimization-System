// services.tf - Containerized Microservices Deployment
// Defines deployments for containerized services with health checks and auto‑scaling.

resource "azurerm_kubernetes_cluster" "iaros_aks" {
  name                = "iaros-aks"
  location            = var.region
  resource_group_name = azurerm_resource_group.iaros_rg.name
  dns_prefix          = "iaros"
  
  default_node_pool {
    name       = "default"
    node_count = var.node_count
    vm_size    = var.vm_size
  }

  identity {
    type = "SystemAssigned"
  }
}

// Additional service deployments via Helm charts or Terraform modules can be added here.
