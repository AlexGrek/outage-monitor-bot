#!/bin/bash
set -e

# This script ensures the Helm chart structure exists
# It's idempotent - safe to run multiple times

CHART_DIR="helm/tg-monitor-bot"

echo "Checking Helm chart structure at $CHART_DIR..."

if [ -d "$CHART_DIR" ]; then
    echo "✓ Helm chart already exists at $CHART_DIR"
    echo "  Chart.yaml: $([ -f "$CHART_DIR/Chart.yaml" ] && echo "✓" || echo "✗")"
    echo "  values.yaml: $([ -f "$CHART_DIR/values.yaml" ] && echo "✓" || echo "✗")"
    echo "  templates/: $([ -d "$CHART_DIR/templates" ] && echo "✓" || echo "✗")"
else
    echo "✗ Helm chart not found. Please run this from the repository root."
    exit 1
fi

echo ""
echo "Helm chart is ready!"
echo ""
echo "Next steps:"
echo "  1. Update values in helm/tg-monitor-bot/values.yaml"
echo "  2. Package: make helm-package"
echo "  3. Install: make helm-install"
