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
workspace_dir="${GIZCLAW_E2E_SERVER_WORKSPACE:-$testbench_dir/workspace}"
context_home="${GIZCLAW_E2E_CONTEXT_HOME:-$testbench_dir/context}"
listen_addr="${GIZCLAW_E2E_SERVER_ADDR:-127.0.0.1:9820}"
cipher_mode="${GIZCLAW_E2E_SERVER_CIPHER_MODE:-chacha_poly}"
gizclaw_bin_explicit="${GIZCLAW_BIN+x}"
gizclaw_bin="${GIZCLAW_BIN:-$testbench_dir/bin/gizclaw}"

mkdir -p "$workspace_dir" "$(dirname "$gizclaw_bin")"

if [[ -z "$gizclaw_bin_explicit" || ! -x "$gizclaw_bin" ]]; then
  (cd "$repo_root" && go build -o "$gizclaw_bin" ./cmd/gizclaw)
fi

admin_line=""
if [[ -n "${GIZCLAW_E2E_ADMIN_PUBLIC_KEY:-}" ]]; then
  admin_line="admin-public-key: ${GIZCLAW_E2E_ADMIN_PUBLIC_KEY}"$'\n'
fi

python3 - "$script_dir/config.yaml.template" "$workspace_dir/config.yaml" "$listen_addr" "$cipher_mode" "$admin_line" <<'PY'
import pathlib
import sys

template_path, out_path, listen_addr, cipher_mode, admin_line = sys.argv[1:6]
text = pathlib.Path(template_path).read_text()
text = text.replace("__LISTEN_ADDR__", listen_addr)
text = text.replace("__CIPHER_MODE__", cipher_mode)
text = text.replace("__ADMIN_PUBLIC_KEY_LINE__", admin_line)
pathlib.Path(out_path).write_text(text)
PY

"$gizclaw_bin" migrate --workspace "$workspace_dir"

if [[ -n "${GIZCLAW_WORKSPACE_CLIENT_PRIVATE_KEY:-}" ]]; then
  (cd "$repo_root" && go run ./test/gizclaw-e2e/setup/write_context_config.go \
    --context-home "$context_home" \
    --server-workspace "$workspace_dir" \
    --server-addr "$listen_addr" \
    --cipher-mode "$cipher_mode" \
    --context-name "${GIZCLAW_E2E_CLIENT_CONTEXT:-e2e-client}" \
    --client-private-key "${GIZCLAW_WORKSPACE_CLIENT_PRIVATE_KEY}" \
    --client-public-key "${GIZCLAW_E2E_CLIENT_PUBLIC_KEY:-}")
fi

echo "GizClaw e2e server workspace: $workspace_dir"
echo "GizClaw e2e context home:     $context_home"
echo "GizClaw e2e server addr:      $listen_addr"
exec "$gizclaw_bin" serve --force "$workspace_dir"
