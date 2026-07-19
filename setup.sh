#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'; CYAN='\033[0;36m'; YELLOW='\033[0;33m'; BOLD='\033[1m'; RESET='\033[0m'
ok()   { echo -e "${GREEN}${BOLD}  ✔${RESET}  $*"; }
info() { echo -e "${CYAN}${BOLD}  ▸${RESET}  $*"; }

# ── 1. Write .replit with go-1.25 to workspace root ───────────────────────────
WORKSPACE_ROOT="${REPL_HOME:-/home/runner/workspace}"
info "Writing .replit with go-1.25 to $WORKSPACE_ROOT"
cat > "$WORKSPACE_ROOT/.replit" << 'EOF'
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
EOF
ok ".replit written — close and reopen shell if go is not found below"

# ── 2. Clone repo ─────────────────────────────────────────────────────────────
info "Cloning Zorzer_Layer_7"
git clone https://github.com/victusxxgaming-design/Zorzer_Layer_7.git
cd Zorzer_Layer_7
ok "Cloned and entered Zorzer_Layer_7/"

# ── 3. go mod tidy ────────────────────────────────────────────────────────────
info "Running go mod tidy"
go mod tidy
ok "go.mod tidy"

# ── 4. Build ──────────────────────────────────────────────────────────────────
info "Building zorzer binary"
go build -o zorzer .
chmod +x zorzer
ok "Binary built → ./zorzer"

# ── Done ──────────────────────────────────────────────────────────────────────
echo
echo -e "${GREEN}${BOLD}  Done.${RESET}"
echo
echo -e "${YELLOW}${BOLD}  Next steps:${RESET}"
echo -e "  1. Create a Supabase project and a table called ${BOLD}layer7_apis${RESET}"
echo -e "  2. Open ${CYAN}Zorzer_Layer_7/config.json${RESET} and fill in:"
echo -e "       supabase.url        → your Supabase project URL"
echo -e "       supabase.service_key → your Supabase service role key"
echo -e "  3. Start the server:  ${CYAN}${BOLD}./zorzer -api${RESET}  (or hit Run)"
echo
