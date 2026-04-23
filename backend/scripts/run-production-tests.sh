#!/usr/bin/env bash
# ════════════════════════════════════════════════════════════════════════════════
# Production Integration Test Runner
#
# Runs real external integration tests (SendGrid, FCM, Stripe, Kafka).
# These tests require live credentials and running infrastructure.
#
# Usage:
#   ./scripts/run-production-tests.sh              # Run all
#   ./scripts/run-production-tests.sh email         # Run only email tests
#   ./scripts/run-production-tests.sh push          # Run only push tests
#   ./scripts/run-production-tests.sh stripe        # Run only Stripe webhook tests
#   ./scripts/run-production-tests.sh kafka         # Run only Kafka E2E tests
#
# Prerequisites:
#   - DATABASE_URL env var pointing to a staging/test Postgres
#   - REDIS_URL env var pointing to a running Redis instance
#   - EMAIL_PROVIDER + SENDGRID_API_KEY (for email tests)
#   - FIREBASE_SERVICE_ACCOUNT_JSON + FCM_TEST_DEVICE_TOKEN (for push tests)
#   - STRIPE_WEBHOOK_SECRET + STRIPE_API_KEY (for Stripe tests)
#   - KAFKA_BROKERS (for Kafka E2E tests)
#
# Optional:
#   - stripe listen --forward-to localhost:8080/webhooks/stripe  (for live webhooks)
# ════════════════════════════════════════════════════════════════════════════════

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}  GeoCore Production Integration Test Runner${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"

# ── Check required env vars ────────────────────────────────────────────────────
check_env() {
    local name="$1"
    local desc="$2"
    if [ -z "${!name:-}" ]; then
        echo -e "${YELLOW}  ⚠ $name not set — $desc tests will be skipped${NC}"
        return 1
    else
        echo -e "${GREEN}  ✓ $name is set${NC}"
        return 0
    fi
}

echo ""
echo -e "${CYAN}Checking prerequisites...${NC}"
check_env DATABASE_URL "all"
check_env REDIS_URL "all"

EMAIL_OK=true
PUSH_OK=true
STRIPE_OK=true
KAFKA_OK=true

check_env EMAIL_PROVIDER "email" || EMAIL_OK=false
check_env SENDGRID_API_KEY "email" || EMAIL_OK=false
check_env EMAIL_FROM "email" || EMAIL_OK=false
check_env EMAIL_TEST_INBOX "email" || EMAIL_OK=false

check_env FIREBASE_SERVICE_ACCOUNT_JSON "push" || PUSH_OK=false
check_env FCM_TEST_DEVICE_TOKEN "push" || PUSH_OK=false

check_env STRIPE_WEBHOOK_SECRET "stripe" || STRIPE_OK=false
check_env STRIPE_API_KEY "stripe" || STRIPE_OK=false

check_env KAFKA_BROKERS "kafka" || KAFKA_OK=false

echo ""

# ── Determine which tests to run ──────────────────────────────────────────────
FILTER="${1:-}"
TIMEOUT="${TIMEOUT:-300s}"

case "$FILTER" in
    email)
        TEST_PATTERN="TestEmailSuite"
        ;;
    push)
        TEST_PATTERN="TestPushSuite"
        ;;
    stripe)
        TEST_PATTERN="TestStripeWebhookSuite"
        ;;
    kafka)
        TEST_PATTERN="TestKafkaE2ESuite"
        ;;
    *)
        TEST_PATTERN="."
        ;;
esac

# ── Run tests ──────────────────────────────────────────────────────────────────
echo -e "${CYAN}Running production integration tests...${NC}"
echo -e "  Pattern: $TEST_PATTERN"
echo -e "  Timeout: $TIMEOUT"
echo ""

cd "$PROJECT_DIR"

go test \
    -tags=production \
    -run "$TEST_PATTERN" \
    -timeout "$TIMEOUT" \
    -v \
    ./test/production/

EXIT_CODE=$?

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}  ✓ All production integration tests PASSED${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
else
    echo -e "${RED}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${RED}  ✗ Some production integration tests FAILED${NC}"
    echo -e "${RED}═══════════════════════════════════════════════════════════════${NC}"
fi

exit $EXIT_CODE
