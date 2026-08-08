#!/usr/bin/env bash
# Quick smoke test for subscriber + email config (run after deploy).
# Usage: ./scripts/e2e-subscribers.sh [API_URL] [TEST_EMAIL]
set -euo pipefail

API_URL="${1:-https://krishblog-production.up.railway.app}"
EMAIL="${2:-krish22092003@gmail.com}"

echo "→ Health check"
curl -sf "$API_URL/health" | head -c 120
echo ""

echo "→ Subscribe $EMAIL"
SUB_RES=$(curl -sf -X POST "$API_URL/v1/subscribe" \
  -H 'content-type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"name\":\"E2E\"}")
echo "$SUB_RES"

echo ""
echo "Done. Check inbox for confirmation link (Resend test mode only delivers to your Resend account email)."
echo "After confirming, publish a post in admin and click Notify subscribers."
