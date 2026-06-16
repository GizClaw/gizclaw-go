#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"

env_file="${GIZCLAW_E2E_ENV:-$script_dir/.env}"
env_explicit=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --env|-e)
      if [[ $# -lt 2 ]]; then
        echo "missing value for $1" >&2
        exit 2
      fi
      env_file="$2"
      env_explicit=1
      shift 2
      ;;
    --no-env)
      env_file=""
      shift
      ;;
    *)
      echo "unexpected argument: $1" >&2
      exit 2
      ;;
  esac
done

if [[ -n "$env_file" ]]; then
  if [[ -f "$env_file" ]]; then
    set -a
    # shellcheck disable=SC1090
    source "$env_file"
    set +a
  elif [[ "$env_explicit" == "1" ]]; then
    echo "env file not found: $env_file" >&2
    exit 2
  fi
fi

testbench_dir="${GIZCLAW_E2E_TESTBENCH:-$repo_root/test/gizclaw-e2e/.testbench}"
gizclaw_bin="${GIZCLAW_BIN:-$testbench_dir/bin/gizclaw}"
context_name="${GIZCLAW_E2E_ADMIN_CONTEXT:-e2e-admin}"
resource_dir="${GIZCLAW_E2E_RESOURCE_DIR:-$script_dir/resources}"

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "missing required env: $name" >&2
    exit 2
  fi
}

require_env GIZCLAW_E2E_OPENAI_API_KEY
require_env GIZCLAW_E2E_OPENAI_BASE_URL
require_env GIZCLAW_E2E_OPENAI_UPSTREAM_MODEL
require_env GIZCLAW_E2E_VOLC_APP_ID
require_env GIZCLAW_E2E_VOLC_TOKEN
require_env GIZCLAW_E2E_VOLC_VOICE_ID
require_env GIZCLAW_E2E_VOICE_RESOURCE
require_env GIZCLAW_E2E_CLIENT_PUBLIC_KEY

if [[ ! -d "$resource_dir" ]]; then
  echo "resource directory not found: $resource_dir" >&2
  exit 2
fi

shopt -s nullglob
resource_files=("$resource_dir"/*.json)
shopt -u nullglob
if [[ ${#resource_files[@]} -eq 0 ]]; then
  echo "no resource files found in: $resource_dir" >&2
  exit 2
fi

if [[ ! -x "$gizclaw_bin" ]]; then
  mkdir -p "$(dirname "$gizclaw_bin")"
  (cd "$repo_root" && go build -o "$gizclaw_bin" ./cmd/gizclaw)
fi

for resource_file in "${resource_files[@]}"; do
  "$gizclaw_bin" admin apply --context "$context_name" -f "$resource_file"
done

echo "Applied e2e shared setup resources in context: $context_name"
echo "resource_dir=$resource_dir"
echo "chat_model=${GIZCLAW_E2E_CHAT_MODEL:-e2e-chat}"
echo "tts_model=${GIZCLAW_E2E_TTS_MODEL:-e2e-tts}"
echo "asr_model=${GIZCLAW_E2E_ASR_MODEL:-e2e-asr}"
echo "realtime_model=${GIZCLAW_E2E_REALTIME_MODEL:-e2e-realtime}"
echo "acl_view=${GIZCLAW_E2E_ACL_VIEW:-e2e-client}"
echo "voice=${GIZCLAW_E2E_VOICE_RESOURCE}"
echo "workspace=${GIZCLAW_E2E_WORKSPACE:-doubao-realtime}"
echo "workflow=${GIZCLAW_E2E_WORKFLOW:-e2e-doubao-realtime}"
