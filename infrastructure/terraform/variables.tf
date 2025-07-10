// variables.tf - Input Variables for Terraform
variable "region" {
  description = "Azure region to deploy resources."
  type        = string
  default     = "eastus"
}

variable "environment" {
  description = "Deployment environment: dev, staging, or production."
  type        = string
  default     = "production"
}

variable "node_count" {
  description = "Number of nodes in the AKS cluster."
  type        = number
  default     = 3
}

variable "vm_size" {
  description = "Virtual machine size for the AKS nodes."
  type        = string
  default     = "Standard_D4s_v3"
}

// Additional variables for failover thresholds, cost controls, etc.
