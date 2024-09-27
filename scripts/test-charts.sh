#!/bin/bash

set -e

for chart in infra/*/; do
  if [ -f "$chart/Chart.yaml" ]; then
    echo "Testing chart: $chart"
    helm lint "$chart"
    helm template "$chart" > /dev/null
  fi
done

echo "All charts passed linting and template generation."