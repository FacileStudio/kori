#!/usr/bin/env bash
# scripts/boite-sync.sh - rsync helper for project → VM sync
#
# Syncs the current project directory to a boite VM's overlay disk
# to enable isolated development environments.
#
# Usage: boite-sync.sh <vm-name> [port] [--dry-run] [--exclude pattern] [--quiet]
#
# Requires: boite (in PATH), rsync, ssh
# Environment: Uses ~/.boite/state.json for VM configuration if port not specified
#
# Exit codes:
#   0 - Success
#   1 - Invalid arguments
#   2 - VM not found in boite state
#   3 - SSH connection failed
#   4 - rsync failed

set -euo pipefail

# Default values
DRY_RUN=false
QUIET=false
EXCLUDES=()

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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
  [port]        SSH port (optional, defaults to value in ~/.boite/state.json or 2226)

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

# Parse arguments
if [[ $# -lt 1 ]]; then
    usage
    exit 1
fi

VM_NAME="$1"
shift

# Parse port (if first remaining argument is a number)
if [[ $# -gt 0 && "$1" =~ ^[0-9]+$ ]]; then
    PORT="$1"
    shift
fi

# Parse remaining options
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

# Validate VM name
if [[ -z "$VM_NAME" ]]; then
    log_error "VM name is required"
    usage
    exit 1
fi

# Get SSH connection details from boite state or defaults
get_boite_config() {
    local home
    home=$(eval echo ~"$USER")
    local state_file="$home/.boite/state.json"
    
    if [[ -f "$state_file" ]]; then
        # Try to parse JSON state file
        if command -v jq >/dev/null 2>&1; then
            local port key
            port=$(jq -r ".VMs.${VM_NAME}.ssh_port // empty" "$state_file" 2>/dev/null || true)
            key=$(jq -r ".VMs.${VM_NAME}.ssh_key_path // empty" "$state_file" 2>/dev/null || true)
            
            if [[ -n "$port" && "$port" != "null" ]]; then
                echo "$port"
                echo "$key"
                return 0
            fi
        fi
    fi
    
    # Fallback to boite list command if JSON parsing fails or jq not available
    if command -v boite >/dev/null 2>&1; then
        local output
        output=$(boite list --json 2>/dev/null || true)
        if [[ -n "$output" ]]; then
            local port key
            port=$(echo "$output" | jq -r ".VMs.${VM_NAME}.ssh_port // empty" 2>/dev/null || true)
            key=$(echo "$output" | jq -r ".VMs.${VM_NAME}.ssh_key_path // empty" 2>/dev/null || true)
            
            if [[ -n "$port" && "$port" != "null" ]]; then
                echo "$port"
                echo "$key"
                return 0
            fi
        fi
    fi
    
    # Default values
    echo "2226"
    echo "~/.ssh/id_ed25519"
}

# Get VM configuration
read -r PORT_DEFAULT KEY_PATH <<< "$(get_boite_config)"
PORT=${PORT:-$PORT_DEFAULT}
SSH_KEY_PATH=${SSH_KEY_PATH:-$KEY_PATH}

# Validate VM exists in boite state
if ! boite list --json 2>/dev/null | jq -e ".VMs.${VM_NAME}" >/dev/null 2>&1; then
    log_error "VM '$VM_NAME' not found in boite state"
    log_info "Available VMs:"
    boite list --json 2>/dev/null | jq -r '.VMs | keys[]' 2>/dev/null || boite list 2>/dev/null
    exit 2
fi

# Validate SSH key exists
SSH_KEY_PATH=$(eval echo "$SSH_KEY_PATH")
if [[ ! -f "$SSH_KEY_PATH" ]]; then
    log_error "SSH key not found: $SSH_KEY_PATH"
    exit 3
fi

# Get current working directory
LOCAL_DIR="$(pwd)/"
REMOTE_PATH="/home/yann/project/"

log_info "Syncing project to VM '$VM_NAME'"
log_info "  Local:  $LOCAL_DIR"
log_info "  Remote: $SSH_KEY_PATH@localhost:$PORT:$REMOTE_PATH"
log_info "  Port:   $PORT"
log_info "  Key:    $SSH_KEY_PATH"

if [[ "$DRY_RUN" == "true" ]]; then
    log_info "DRY RUN MODE - No actual transfer will occur"
fi

# Build rsync command
RSYNC_OPTS=(
    -avz
    --delete
    "${EXCLUDES[@]}"
    --exclude=.boite/
    -e "ssh -p $PORT -i $SSH_KEY_PATH -o StrictHostKeyChecking=accept-new"
)

if [[ "$DRY_RUN" == "true" ]]; then
    RSYNC_OPTS+=(--dry-run)
fi

RSYNC_OPTS+=("$LOCAL_DIR" "")
RSYNC_OPTS+=("${VM_NAME}@localhost:$REMOTE_PATH")

# Execute rsync
if ! rsync "${RSYNC_OPTS[@]}"; then
    log_error "rsync failed"
    exit 4
fi

log_info "Sync completed successfully"
if [[ "$DRY_RUN" == "true" ]]; then
    log_info "Remember to run without --dry-run to perform actual sync"
fi

exit 0