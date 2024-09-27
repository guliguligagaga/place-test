#!/bin/bash

set -e

# Set variables
NAMESPACE="r-clone"
CHARTS_DIR="infra"

# Create namespace if it doesn't exist
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# Deploy charts
for chart in "$CHARTS_DIR"/*/; do
  if [ -f "$chart/Chart.yaml" ]; then
    chart_name=$(basename "$chart")
    echo "Deploying chart: $chart_name"
    helm upgrade --install "$chart_name" "$chart" --namespace "$NAMESPACE" --wait
  fi
done

echo "Deployment completed successfully."