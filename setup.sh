#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
#  Zorzer L7 Stresser — setup script
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

# ── 0. Write .replit with go-1.25 module to workspace root (absolute first) ───
# Replit only reads .replit from the workspace root, not subdirectories.
WORKSPACE_ROOT="${REPL_HOME:-/home/runner/workspace}"
cat > "$WORKSPACE_ROOT/.replit" << 'REPLITEOF'
modules = ["go-1.25"]

[workflows]
runButton = "Project"

[[workflows.workflow]]
name = "Project"
mode = "parallel"
author = "agent"

[[workflows.workflow.tasks]]
task = "workflow.run"
args = "Start application"

[[workflows.workflow]]
name = "Start application"
author = "agent"

[workflows.workflow.metadata]
outputType = "console"

[[workflows.workflow.tasks]]
task = "shell.exec"
args = "fuser -k 8080/tcp 2>/dev/null || true; go build -o zorzer . && chmod +x zorzer && ./zorzer -api"
waitForPort = 8080

[[ports]]
localPort = 8080
externalPort = 80
REPLITEOF
ok ".replit written to $WORKSPACE_ROOT with modules = [\"go-1.25\"]"

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

# .replit was just written — Go may not be in PATH yet for this shell session.
# Check fast known locations before giving up.
if ! command -v go &>/dev/null; then
  info "go not in PATH — checking known locations"
  for _GOCANDIDATE in \
    "$HOME/.nix-profile/bin/go" \
    "/run/current-system/sw/bin/go" \
    "/nix/var/nix/profiles/default/bin/go" \
    "/home/runner/.replit/modules/go-1.25/bin/go" \
    "/opt/buildhome/go/bin/go"; do
    if [ -x "$_GOCANDIDATE" ]; then
      export PATH="$(dirname "$_GOCANDIDATE"):$PATH"
      ok "Found Go at $_GOCANDIDATE"
      break
    fi
  done
fi

if ! command -v go &>/dev/null; then
  echo
  echo -e "${YELLOW}${BOLD}  .replit has been written with go-1.25.${RESET}"
  echo -e "${YELLOW}  Replit needs a fresh shell to activate the module.${RESET}"
  echo -e "${YELLOW}  Close this shell, open a new one, then re-run:${RESET}"
  echo -e "${CYAN}${BOLD}    bash setup.sh${RESET}"
  echo
  exit 1
fi

GO_VER=$(go version | awk '{print $3}')
ok "Found ${GO_VER}"
sep

# ── 2. System tools (fuser / psmisc) ─────────────────────────────────────────
info "Checking system tools"
if command -v fuser &>/dev/null; then
  ok "fuser already available"
else
  info "fuser not found — installing psmisc via nix-env"
  if command -v nix-env &>/dev/null; then
    nix-env -iA nixpkgs.psmisc 2>&1 | tail -3
    export PATH="$HOME/.nix-profile/bin:$PATH"
    command -v fuser &>/dev/null \
      && ok "psmisc installed — fuser now available" \
      || warn "psmisc installed but fuser not in PATH yet; restart the workflow"
  else
    warn "nix-env not available — install psmisc manually if needed"
  fi
fi
sep

# ── 3. config.json ────────────────────────────────────────────────────────────
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
  warn "Edit config.json and fill in your Supabase credentials, then re-run."
else
  SB_URL=$(python3 -c "import json; d=json.load(open('$CONFIG_FILE')); print(d.get('supabase',{}).get('url',''))" 2>/dev/null || echo "")
  SB_KEY=$(python3 -c "import json; d=json.load(open('$CONFIG_FILE')); print(d.get('supabase',{}).get('service_key',''))" 2>/dev/null || echo "")
  [ -n "$SB_URL" ] && ok "supabase.url = $SB_URL" || warn "supabase.url is empty"
  [ -n "$SB_KEY" ] && ok "supabase.service_key is set" || warn "supabase.service_key is empty"
fi
sep

# ── 4. Go module dependencies ─────────────────────────────────────────────────
info "Running go mod tidy"
go mod tidy 2>&1 | sed 's/^/     /'
ok "go.mod + go.sum up to date"
sep

# ── 5. Build ──────────────────────────────────────────────────────────────────
info "Building zorzer binary"
go build -o zorzer . 2>&1 | sed 's/^/     /'
chmod +x zorzer
BINARY_SIZE=$(du -sh zorzer | cut -f1)
ok "Binary built → ./zorzer (${BINARY_SIZE})"
sep

# ── 6. Summary ────────────────────────────────────────────────────────────────
echo
echo -e "${GREEN}${BOLD}  Setup complete.${RESET}"
echo
echo -e "  Run the API server:"
echo -e "  ${CYAN}${BOLD}    ./zorzer -api${RESET}"
echo
echo -e "  Or let the workflow handle it automatically."
echo
