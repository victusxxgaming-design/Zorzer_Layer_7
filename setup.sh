#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
#  Slayer L7 — setup script
#  Installs all dependencies, configures Replit modules, builds the binary.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

# ── colours ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; DIM='\033[2m'; RESET='\033[0m'

ok()   { echo -e "${GREEN}${BOLD}  ✔${RESET}  $*"; }
info() { echo -e "${CYAN}${BOLD}  ▸${RESET}  $*"; }
warn() { echo -e "${YELLOW}${BOLD}  ⚠${RESET}  $*"; }
fail() { echo -e "${RED}${BOLD}  ✘${RESET}  $*"; exit 1; }
sep()  { echo -e "${DIM}  ────────────────────────────────────────${RESET}"; }

# ── banner ────────────────────────────────────────────────────────────────────
clear 2>/dev/null || true
echo -e "${RED}${BOLD}"
cat << 'EOF'
   _____ __
  / ___// /___ ___  _____  _____
  \__ \/ / __ `/ / / / _ \/ ___/
 ___/ / / /_/ / /_/ /  __/ /
/____/_/\__,_/\__, /\___/_/    SETUP
             /____/
EOF
echo -e "${RESET}"
echo -e "${DIM}  Slayer L7 — full environment setup${RESET}"
sep

# ── 1. Replit module: go-1.25 ─────────────────────────────────────────────────
info "Checking Replit module: go-1.25"

REPLIT_FILE=".replit"
if [ -f "$REPLIT_FILE" ]; then
  if grep -q '"go-1.25"' "$REPLIT_FILE" || grep -q "go-1.25" "$REPLIT_FILE"; then
    ok "go-1.25 already in $REPLIT_FILE"
  else
    # Add go-1.25 to the modules line
    sed -i 's/^modules = \[/modules = ["go-1.25", /' "$REPLIT_FILE" 2>/dev/null \
      || warn "Could not patch .replit — add go-1.25 manually via the Replit Packages panel"
    ok "Added go-1.25 to $REPLIT_FILE"
  fi
else
  warn ".replit not found — skipping module config (not running inside Replit?)"
fi
sep

# ── 2. Verify Go toolchain ────────────────────────────────────────────────────
info "Verifying Go toolchain"

if ! command -v go &>/dev/null; then
  fail "Go not found. Install go-1.25 via the Replit Packages panel and re-run."
fi

GO_VER=$(go version | awk '{print $3}')
ok "Found ${GO_VER}"
sep

# ── 3. System tools (fuser / psmisc) ─────────────────────────────────────────
info "Checking system tools"

if command -v fuser &>/dev/null; then
  ok "fuser already available ($(command -v fuser))"
else
  info "fuser not found — installing psmisc via nix-env"
  if command -v nix-env &>/dev/null; then
    nix-env -iA nixpkgs.psmisc 2>&1 | tail -3
    if command -v fuser &>/dev/null; then
      ok "psmisc installed — fuser now available"
    else
      warn "psmisc installed but fuser not in PATH yet; re-source shell or restart workflow"
    fi
  else
    warn "nix-env not available — install psmisc manually if you need port auto-kill"
  fi
fi
sep

# ── 4. Go module dependencies ─────────────────────────────────────────────────
info "Running go mod tidy"
go mod tidy 2>&1 | sed 's/^/     /'
ok "go.mod + go.sum up to date"
sep

# ── 5. Build ──────────────────────────────────────────────────────────────────
info "Building slayer binary"
go build -o slayer . 2>&1 | sed 's/^/     /'
chmod +x slayer
BINARY_SIZE=$(du -sh slayer | cut -f1)
ok "Binary built → ./slayer (${BINARY_SIZE})"
sep

# ── 6. Environment variables ──────────────────────────────────────────────────
info "Checking environment variables"

MISSING=()

if [ -z "${SUPABASE_URL:-}" ]; then
  MISSING+=("SUPABASE_URL")
  warn "SUPABASE_URL not set"
else
  ok "SUPABASE_URL = ${SUPABASE_URL}"
fi

if [ -z "${SUPABASE_SERVICE_KEY:-}" ]; then
  MISSING+=("SUPABASE_SERVICE_KEY")
  warn "SUPABASE_SERVICE_KEY not set"
else
  ok "SUPABASE_SERVICE_KEY is set"
fi

if [ ${#MISSING[@]} -gt 0 ]; then
  echo
  warn "Missing secrets: ${MISSING[*]}"
  echo -e "     ${DIM}→ Add them in the Replit Secrets panel (🔒) and re-run.${RESET}"
fi
sep

# ── 7. Summary ────────────────────────────────────────────────────────────────
echo
echo -e "${GREEN}${BOLD}  Setup complete.${RESET}"
echo
echo -e "  Run the API server:"
echo -e "  ${CYAN}${BOLD}  ./slayer -api${RESET}"
echo
echo -e "  Or let the workflow handle it automatically."
echo
