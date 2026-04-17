#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass=0
fail=0

ok()   { echo -e "${GREEN}PASS${NC} $1"; ((pass++)) || true; }
err()  { echo -e "${RED}FAIL${NC} $1"; ((fail++)) || true; }

echo "========================================"
echo " Setup Script Tests"
echo "========================================"
echo ""

# Test 1: setup.sh is executable
if [[ -x "$SCRIPT_DIR/setup.sh" ]]; then
    ok "setup.sh is executable"
else
    err "setup.sh is not executable"
fi

# Test 2: setup-arch.sh is executable
if [[ -x "$SCRIPT_DIR/setup-arch.sh" ]]; then
    ok "setup-arch.sh is executable"
else
    err "setup-arch.sh is not executable"
fi

# Test 3: dev-setup.sh is executable
if [[ -x "$SCRIPT_DIR/dev-setup.sh" ]]; then
    ok "dev-setup.sh is executable"
else
    err "dev-setup.sh is not executable"
fi

# Test 4: setup.sh has valid syntax
if bash -n "$SCRIPT_DIR/setup.sh" 2>&1; then
    ok "setup.sh syntax is valid"
else
    err "setup.sh has syntax errors"
fi

# Test 5: setup-arch.sh has valid syntax
if bash -n "$SCRIPT_DIR/setup-arch.sh" 2>&1; then
    ok "setup-arch.sh syntax is valid"
else
    err "setup-arch.sh has syntax errors"
fi

# Test 6: dev-setup.sh has valid syntax
if bash -n "$SCRIPT_DIR/dev-setup.sh" 2>&1; then
    ok "dev-setup.sh syntax is valid"
else
    err "dev-setup.sh has syntax errors"
fi

# Test 7: setup.sh uses set -euo pipefail
if grep -q "^set -euo pipefail" "$SCRIPT_DIR/setup.sh"; then
    ok "setup.sh uses set -euo pipefail"
else
    err "setup.sh missing set -euo pipefail"
fi

# Test 8: setup-arch.sh uses set -euo pipefail
if grep -q "^set -euo pipefail" "$SCRIPT_DIR/setup-arch.sh"; then
    ok "setup-arch.sh uses set -euo pipefail"
else
    err "setup-arch.sh missing set -euo pipefail"
fi

# Test 9: dev-setup.sh uses set -euo pipefail
if grep -q "^set -euo pipefail" "$SCRIPT_DIR/dev-setup.sh"; then
    ok "dev-setup.sh uses set -euo pipefail"
else
    err "dev-setup.sh missing set -euo pipefail"
fi

# Test 10: dev-setup.sh runs successfully (idempotent)
if bash "$SCRIPT_DIR/dev-setup.sh" >/dev/null 2>&1; then
    ok "dev-setup.sh runs successfully"
else
    err "dev-setup.sh failed to run"
fi

echo ""
echo "========================================"
echo " Results: $pass passed, $fail failed"
echo "========================================"

[[ $fail -eq 0 ]]
