#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
e2e_dir="$repo_root/test/gizclaw-e2e"
testdata_dir="$repo_root/test/gizclaw-e2e/testdata"
workspace_dir="$testdata_dir/server-workspace"
bin_path="$testdata_dir/bin/gizclaw"
pid_file="$workspace_dir/gizclaw-server.pid"
log_file="$workspace_dir/gizclaw-server.log"
env_file="$e2e_dir/.env"

if [[ -f "$env_file" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$env_file"
  set +a
fi

config_home="${GIZCLAW_E2E_CONFIG_HOME:-$testdata_dir/config-home}"
context_name="${GIZCLAW_E2E_ADMIN_SETUP_CONTEXT:-e2e-admin}"
launch_label="com.gizclaw.e2e.server.$(printf '%s' "$repo_root" | cksum | awk '{print $1}')"

launchctl_supported() {
  [[ "$(uname -s)" == "Darwin" ]] && command -v launchctl >/dev/null 2>&1
}

launchctl_pid() {
  launchctl list "$launch_label" 2>/dev/null | awk -F'= ' '/"PID"/ {gsub(/[; \t]/, "", $2); print $2; exit}'
}

wait_ready() {
  for _ in {1..300}; do
    if XDG_CONFIG_HOME="$config_home" "$bin_path" connect ping --context "$context_name" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  echo "gizclaw e2e server did not become ready; log=$log_file" >&2
  return 1
}

if [[ ! -x "$bin_path" ]]; then
  "$script_dir/build.sh" >/dev/null
fi

if [[ -f "$pid_file" ]]; then
  pid="$(cat "$pid_file")"
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    wait_ready
    echo "gizclaw e2e server already running: pid=$pid"
    exit 0
  fi
  rm -f "$pid_file"
fi

"$bin_path" migrate --workspace "$workspace_dir"
if launchctl_supported; then
  launchctl remove "$launch_label" >/dev/null 2>&1 || true
  launchctl submit -l "$launch_label" -o "$log_file" -e "$log_file" -- "$bin_path" serve --force "$workspace_dir"
  pid="$(launchctl_pid)"
else
  cd "$repo_root"
  nohup "$bin_path" serve --force "$workspace_dir" >"$log_file" 2>&1 </dev/null &
  pid="$!"
fi
echo "$pid" >"$pid_file"
wait_ready
if launchctl_supported; then
  pid="$(launchctl_pid)"
  echo "$pid" >"$pid_file"
fi
if ! kill -0 "$pid" 2>/dev/null; then
  echo "gizclaw e2e server exited after readiness; log=$log_file" >&2
  tail -40 "$log_file" >&2 || true
  rm -f "$pid_file"
  exit 1
fi
echo "gizclaw e2e server pid=$pid log=$log_file"
