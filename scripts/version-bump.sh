#!/bin/bash
set -e

# Script to bump version across all project files
# Usage: ./scripts/version-bump.sh [patch|minor|major]

BUMP_TYPE=${1:-patch}
VERSION_FILE="VERSION"
CHART_FILE="helm/tg-monitor-bot/Chart.yaml"

# Check if VERSION file exists
if [ ! -f "$VERSION_FILE" ]; then
    echo "Error: VERSION file not found"
    exit 1
fi

# Read current version
CURRENT_VERSION=$(cat "$VERSION_FILE")

# Parse version components
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# Bump version based on type
case "$BUMP_TYPE" in
    patch)
        PATCH=$((PATCH + 1))
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    *)
        echo "Error: Invalid bump type. Use: patch, minor, or major"
        exit 1
        ;;
esac

# Construct new version
NEW_VERSION="$MAJOR.$MINOR.$PATCH"

echo "Bumping $BUMP_TYPE version: $CURRENT_VERSION → $NEW_VERSION"

# Update VERSION file
echo "$NEW_VERSION" > "$VERSION_FILE"
echo "✓ Updated $VERSION_FILE"

# Update Chart.yaml if it exists
if [ -f "$CHART_FILE" ]; then
    # Detect OS for sed compatibility
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s/^version: .*/version: $NEW_VERSION/" "$CHART_FILE"
        sed -i '' "s/^appVersion: .*/appVersion: \"$NEW_VERSION\"/" "$CHART_FILE"
    else
        # Linux
        sed -i "s/^version: .*/version: $NEW_VERSION/" "$CHART_FILE"
        sed -i "s/^appVersion: .*/appVersion: \"$NEW_VERSION\"/" "$CHART_FILE"
    fi
    echo "✓ Updated $CHART_FILE"
else
    echo "⚠ Helm Chart.yaml not found, skipping"
fi

echo ""
echo "Version bumped successfully to $NEW_VERSION"
echo ""
echo "Files updated:"
echo "  - $VERSION_FILE"
[ -f "$CHART_FILE" ] && echo "  - $CHART_FILE"
echo ""
echo "Next steps:"
echo "  1. Review changes: git diff"
echo "  2. Build multi-arch image: make docker-build-multiarch"
echo "  3. Update Helm release: make helm-upgrade"
echo "  4. Commit changes: git add . && git commit -m 'Bump version to $NEW_VERSION'"
