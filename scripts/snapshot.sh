#!/usr/bin/env bash
# scripts/snapshot.sh — Create a timestamped Neon branch before running migrations.
# Usage: ./scripts/snapshot.sh [branch-name-prefix]
#
# Requires:
#   - neon CLI installed (brew install neonctl)
#   - NEON_PROJECT_ID env var (or set it below)
#   - neon auth already done (neon auth)

set -euo pipefail

PROJECT_ID="${NEON_PROJECT_ID:-damp-frost-89475570}"
PREFIX="${1:-pre-migrate}"
TIMESTAMP=$(date -u +"%Y%m%dT%H%M%SZ")
BRANCH_NAME="${PREFIX}-${TIMESTAMP}"

echo "📸 Creating Neon snapshot: ${BRANCH_NAME}"

OUTPUT=$(neon branches create \
  --project-id "$PROJECT_ID" \
  --name "$BRANCH_NAME" \
  2>&1)

echo "$OUTPUT"

# Extract just the branch name from output for confirmation
CREATED=$(echo "$OUTPUT" | grep -oE '[a-z0-9-]+-[0-9]{8}T[0-9]{6}Z' | head -1 || echo "$BRANCH_NAME")
echo ""
echo "✅ Snapshot created: ${BRANCH_NAME}"
echo "   Restore anytime via: neon branches --project-id ${PROJECT_ID}"
echo "   Or in the Neon console → your project → Branches"
echo ""

