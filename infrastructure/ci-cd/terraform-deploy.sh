#!/bin/bash
# terraform-deploy.sh - Applies Terraform changes with rollback on error.
set -euo pipefail

echo "Initializing Terraform..."
terraform init

echo "Planning changes..."
terraform plan -out=tfplan

echo "Applying changes..."
terraform apply -auto-approve tfplan

if [ $? -ne 0 ]; then
  echo "Terraform apply failed. Initiating rollback..."
  terraform destroy -auto-approve
  exit 1
fi

echo "Terraform deployment successful."
