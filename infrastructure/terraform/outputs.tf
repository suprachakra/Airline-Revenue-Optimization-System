// outputs.tf - Export Resource Endpoints and Connection Strings
output "aks_cluster_endpoint" {
  value       = azurerm_kubernetes_cluster.iaros_aks.kube_config.0.host
  description = "The endpoint of the IAROS AKS cluster."
}

output "resource_group_name" {
  value       = azurerm_resource_group.iaros_rg.name
  description = "Resource group name."
}

// Additional outputs (e.g., storage account connection strings) are managed via secrets.
