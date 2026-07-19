#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
#  Zorzer L7 Stresser — setup script
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

# ── 0. Inject go-1.25 into .replit (runs before everything else) ──────────────
if [ -f ".replit" ]; then
  if grep -q "go-1.25" ".replit"; then
    echo -e "${GREEN}${BOLD}  ✔${RESET}  .replit: go-1.25 already present"
  else
    if grep -q "^modules" ".replit"; then
      # Existing modules line — insert go-1.25 at the front of the array
      sed -i 's/^modules = \["/modules = ["go-1.25", "/' ".replit" 2>/dev/null \
        || sed -i 's/^modules = \[/modules = ["go-1.25", /' ".replit"
    else
      # No modules line at all — prepend one
      sed -i '1s/^/modules = ["go-1.25"]\n/' ".replit"
    fi
    echo -e "${GREEN}${BOLD}  ✔${RESET}  .replit: go-1.25 injected into modules"
  fi
else
  echo -e "${YELLOW}${BOLD}  ⚠${RESET}  .replit not found — skipping module injection"
fi

# ── banner ────────────────────────────────────────────────────────────────────
clear 2>/dev/null || true
echo -e "${RED}${BOLD}"
cat << 'EOF'
  ███████╗ ██████╗ ██████╗ ███████╗███████╗██████╗
     ███╔╝██╔═══██╗██╔══██╗   ███╔╝██╔════╝██╔══██╗
    ███╔╝ ██║   ██║██████╔╝  ███╔╝ █████╗  ██████╔╝
   ███╔╝  ██║   ██║██╔══██╗███╔╝   ██╔══╝  ██╔══██╗
  ███████╗╚██████╔╝██║  ██║███████╗███████╗██║  ██║
  ╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝  ╚═╝
               L 7   S T R E S S E R   —   SETUP
EOF
echo -e "${RESET}"
sep

# ── 1. Verify Go toolchain ────────────────────────────────────────────────────
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
  ok "fuser already available"
else
  info "fuser not found — installing psmisc via nix-env"
  if command -v nix-env &>/dev/null; then
    nix-env -iA nixpkgs.psmisc 2>&1 | tail -3
    command -v fuser &>/dev/null && ok "psmisc installed — fuser now available" \
      || warn "psmisc installed but fuser not in PATH yet; restart the workflow"
  else
    warn "nix-env not available — install psmisc manually if you need port auto-kill"
  fi
fi
sep

# ── 4. config.json ────────────────────────────────────────────────────────────
info "Checking config.json"
CONFIG_FILE="${CONFIG_PATH:-config.json}"
if [ ! -f "$CONFIG_FILE" ]; then
  warn "config.json not found — creating template"
  cat > config.json << 'CONFIGEOF'
{
  "supabase": {
    "url": "YOUR_SUPABASE_URL",
    "service_key": "YOUR_SUPABASE_SERVICE_KEY"
  },
  "api": {
    "port": 8080
  },
  "defaults": {
    "method": "httpget",
    "workers": 2048,
    "duration": 30
  }
}
CONFIGEOF
  warn "Edit config.json and fill in your Supabase credentials, then re-run or start the workflow."
else
  SB_URL=$(python3 -c "import json,sys; d=json.load(open('$CONFIG_FILE')); print(d.get('supabase',{}).get('url',''))" 2>/dev/null || echo "")
  SB_KEY=$(python3 -c "import json,sys; d=json.load(open('$CONFIG_FILE')); print(d.get('supabase',{}).get('service_key',''))" 2>/dev/null || echo "")
  [ -n "$SB_URL" ] && ok "supabase.url = $SB_URL" || warn "supabase.url is empty"
  [ -n "$SB_KEY" ] && ok "supabase.service_key is set"  || warn "supabase.service_key is empty"
fi
sep

# ── 5. Go module dependencies ─────────────────────────────────────────────────
info "Running go mod tidy"
go mod tidy 2>&1 | sed 's/^/     /'
ok "go.mod + go.sum up to date"
sep

# ── 6. Build ──────────────────────────────────────────────────────────────────
info "Building zorzer binary"
go build -o zorzer . 2>&1 | sed 's/^/     /'
chmod +x zorzer
BINARY_SIZE=$(du -sh zorzer | cut -f1)
ok "Binary built → ./zorzer (${BINARY_SIZE})"
sep

# ── 7. Summary ────────────────────────────────────────────────────────────────
echo
echo -e "${GREEN}${BOLD}  Setup complete.${RESET}"
echo
echo -e "  Run the API server:"
echo -e "  ${CYAN}${BOLD}    ./zorzer -api${RESET}"
echo
echo -e "  Or let the workflow handle it automatically."
echo
