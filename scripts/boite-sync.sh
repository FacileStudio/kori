#!/usr/bin/env bash
# scripts/boite-sync.sh - rsync helper for project → VM sync
#
# Syncs the current project directory to a boite VM's overlay disk
# to enable isolated development environments.
#
# Usage: boite-sync.sh <vm-name> [port] [--dry-run] [--exclude pattern] [--quiet]
#
# Requires: boite (in PATH) or ~/.boite state, rsync, ssh
# Environment: Uses ~/.boite/instances/<name>/state.json or ~/.boite/state.json
#
# Exit codes:
#   0 - Success
#   1 - Invalid arguments
#   2 - VM not found in boite state
#   3 - SSH connection failed
#   4 - rsync failed

set -euo pipefail

DRY_RUN=false
QUIET=false
EXCLUDES=()

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    if [[ "$QUIET" == "false" ]]; then
        echo -e "${GREEN}[INFO]${NC} $*"
    fi
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

usage() {
    cat << EOF
Usage: $0 <vm-name> [port] [--dry-run] [--exclude pattern] [--quiet]

Syncs the current project directory to a boite VM's overlay disk.

Arguments:
  <vm-name>     Name of the boite VM (required)
  [port]        SSH port (optional, defaults to value in boite state or 2226)

Options:
  --dry-run     Show what would be transferred without actually syncing
  --exclude     Additional rsync exclude pattern (can be used multiple times)
  --quiet       Suppress informational output
  --help        Show this help message

Examples:
  $0 pingu
  $0 pingu 2226
  $0 pingu --dry-run
  $0 pingu --exclude "*.log" --exclude "tmp/"
  $0 pingu 2226 --quiet

Exit codes:
  0  Success
  1  Invalid arguments
  2  VM not found in boite state
  3  SSH connection failed
  4  rsync failed
EOF
}

if [[ $# -lt 1 ]]; then
    usage
    exit 1
fi

VM_NAME="$1"
shift

if [[ $# -gt 0 && "$1" =~ ^[0-9]+$ ]]; then
    PORT="$1"
    shift
fi

while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --exclude)
            if [[ $# -lt 2 ]]; then
                log_error "Missing argument for --exclude"
                exit 1
            fi
            EXCLUDES+=("--exclude" "$2")
            shift 2
            ;;
        --quiet)
            QUIET=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

if [[ -z "$VM_NAME" ]]; then
    log_error "VM name is required"
    usage
    exit 1
fi

get_boite_config() {
    local home="${HOME:-$(eval echo ~"$USER")}"
    local instance_state="$home/.boite/instances/${VM_NAME}/state.json"
    local state_file="$home/.boite/state.json"

    if [[ -f "$instance_state" ]] && command -v jq >/dev/null 2>&1; then
        local port key user workspace
        port=$(jq -r '.ssh_port // empty' "$instance_state" 2>/dev/null || true)
        key=$(jq -r '.key_path // empty' "$instance_state" 2>/dev/null || true)
        user=$(jq -r '.user // empty' "$instance_state" 2>/dev/null || true)
        workspace=$(jq -r '.workspace // empty' "$instance_state" 2>/dev/null || true)
        if [[ -n "$port" && "$port" != "null" ]]; then
            echo "$port"
            echo "${key:-~/.ssh/id_ed25519}"
            echo "${user:-boite}"
            echo "${workspace:-/workspace}"
            return 0
        fi
    fi

    if [[ -f "$state_file" ]] && command -v jq >/dev/null 2>&1; then
        local port key user workspace
        port=$(jq -r ".VMs.${VM_NAME}.ssh_port // empty" "$state_file" 2>/dev/null || true)
        key=$(jq -r ".VMs.${VM_NAME}.ssh_key_path // empty" "$state_file" 2>/dev/null || true)
        user=$(jq -r ".VMs.${VM_NAME}.user // empty" "$state_file" 2>/dev/null || true)
        workspace=$(jq -r ".VMs.${VM_NAME}.workspace // empty" "$state_file" 2>/dev/null || true)
        if [[ -n "$port" && "$port" != "null" ]]; then
            echo "$port"
            echo "${key:-~/.ssh/id_ed25519}"
            echo "${user:-boite}"
            echo "${workspace:-/workspace}"
            return 0
        fi
    fi

    if command -v boite >/dev/null 2>&1; then
        local output
        output=$(boite list --json 2>/dev/null || true)
        if [[ -n "$output" ]]; then
            local port key user workspace
            port=$(echo "$output" | jq -r ".VMs.${VM_NAME}.ssh_port // empty" 2>/dev/null || true)
            key=$(echo "$output" | jq -r ".VMs.${VM_NAME}.ssh_key_path // empty" 2>/dev/null || true)
            user=$(echo "$output" | jq -r ".VMs.${VM_NAME}.user // empty" 2>/dev/null || true)
            workspace=$(echo "$output" | jq -r ".VMs.${VM_NAME}.workspace // empty" 2>/dev/null || true)
            if [[ -n "$port" && "$port" != "null" ]]; then
                echo "$port"
                echo "${key:-~/.ssh/id_ed25519}"
                echo "${user:-boite}"
                echo "${workspace:-/workspace}"
                return 0
            fi
        fi
    fi

    echo "2226"
    echo "~/.ssh/id_ed25519"
    echo "boite"
    echo "/workspace"
}

read -r PORT_DEFAULT KEY_PATH_DEFAULT USER_DEFAULT WORKSPACE_DEFAULT <<< "$(get_boite_config)"
PORT=${PORT:-$PORT_DEFAULT}
SSH_KEY_PATH=${SSH_KEY_PATH:-$KEY_PATH_DEFAULT}
SSH_USER="${SSH_USER:-$USER_DEFAULT}"
REMOTE_PATH="${REMOTE_PATH:-$WORKSPACE_DEFAULT}"

home_dir="${HOME:-$(eval echo ~"$USER")}"
instance_state_file="$home_dir/.boite/instances/${VM_NAME}/state.json"
legacy_state_file="$home_dir/.boite/state.json"

if [[ ! -f "$instance_state_file" ]]; then
    if [[ -f "$legacy_state_file" ]] && command -v jq >/dev/null 2>&1; then
        if ! jq -e ".VMs.${VM_NAME}" "$legacy_state_file" >/dev/null 2>&1; then
            log_warn "VM '$VM_NAME' not found in legacy state file, proceeding with port $PORT"
        fi
    fi
fi

SSH_KEY_PATH=$(eval echo "$SSH_KEY_PATH")
if [[ ! -f "$SSH_KEY_PATH" ]]; then
    log_warn "SSH key not found at $SSH_KEY_PATH, ssh will use default key identity"
fi

LOCAL_DIR="$(pwd)/"

log_info "Syncing project to VM '$VM_NAME'"
log_info "  Local:  $LOCAL_DIR"
log_info "  Remote: $SSH_USER@127.0.0.1:$PORT:$REMOTE_PATH"
log_info "  Port:   $PORT"

if [[ "$DRY_RUN" == "true" ]]; then
    log_info "DRY RUN MODE - No actual transfer will occur"
fi

SSH_CMD="ssh -p $PORT -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=/dev/null"
if [[ -f "$SSH_KEY_PATH" ]]; then
    SSH_CMD="$SSH_CMD -i $SSH_KEY_PATH"
fi

RSYNC_OPTS=(
    -avz
    --delete
    "${EXCLUDES[@]}"
    --exclude=.boite/
    --exclude=.git/
    -e "$SSH_CMD"
)

if [[ "$DRY_RUN" == "true" ]]; then
    RSYNC_OPTS+=(--dry-run)
fi

RSYNC_OPTS+=("$LOCAL_DIR")
RSYNC_OPTS+=("${SSH_USER}@127.0.0.1:$REMOTE_PATH")

if ! rsync "${RSYNC_OPTS[@]}"; then
    log_error "rsync failed"
    exit 4
fi

log_info "Sync completed successfully"
exit 0