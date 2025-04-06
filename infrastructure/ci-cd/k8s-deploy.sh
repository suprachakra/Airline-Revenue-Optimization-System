#!/bin/bash
# k8s-deploy.sh - Deploys or updates Kubernetes resources.
set -euo pipefail

echo "Validating Kubernetes manifests..."
kubeval k8s/*.yaml

echo "Deploying manifests to the cluster..."
kubectl apply -f k8s/

echo "Deployment complete. Monitoring rollout status..."
kubectl rollout status deployment/pricing-service -n iaros-prod
