#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
e2e_dir="$repo_root/test/gizclaw-e2e"
testdata_dir="$repo_root/test/gizclaw-e2e/testdata"
bin_path="$testdata_dir/bin/gizclaw"
env_file="${GIZCLAW_E2E_ENV:-$e2e_dir/.env}"

if [[ -f "$env_file" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$env_file"
  set +a
fi

config_home="$testdata_dir/gizclaw-config-home"
config_home="${GIZCLAW_E2E_PLAY_UI_CONFIG_HOME:-$config_home}"
context_name="${GIZCLAW_E2E_PLAY_UI_CONTEXT:-e2e-client}"
pid_file="$testdata_dir/play-ui.pid"
log_file="$testdata_dir/play-ui.log"
listen_addr="127.0.0.1:8081"
ready_marker="GizClaw Play UI"
launch_label="com.gizclaw.e2e.play-ui.$(printf '%s' "$repo_root" | cksum | awk '{print $1}')"

launchctl_supported() {
  [[ "$(uname -s)" == "Darwin" ]] && command -v launchctl >/dev/null 2>&1
}

launchctl_pid() {
  launchctl list "$launch_label" 2>/dev/null | awk -F'= ' '/"PID"/ {gsub(/[; \t]/, "", $2); print $2; exit}'
}

ui_ready() {
  curl -fsS "http://$listen_addr/" 2>/dev/null | grep -q "$ready_marker"
}

wait_ready() {
  for _ in {1..100}; do
    if launchctl_supported; then
      pid="$(launchctl_pid)"
    fi
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null && ui_ready; then
      return 0
    fi
    sleep 0.1
  done
  echo "gizclaw e2e play UI did not become ready; log=$log_file" >&2
  tail -40 "$log_file" >&2 || true
  rm -f "$pid_file"
  exit 1
}

if [[ ! -x "$bin_path" ]]; then
  "$script_dir/build.sh" >/dev/null
fi

"$script_dir/stop.sh" play-ui >/dev/null || true

if launchctl_supported; then
  launchctl remove "$launch_label" >/dev/null 2>&1 || true
  launchctl submit -l "$launch_label" -o "$log_file" -e "$log_file" -- /usr/bin/env XDG_CONFIG_HOME="$config_home" "$bin_path" play --context "$context_name" --listen "$listen_addr"
  pid="$(launchctl_pid)"
else
  (
    cd "$repo_root"
    export XDG_CONFIG_HOME="$config_home"
    exec nohup "$bin_path" play --context "$context_name" --listen "$listen_addr"
  ) >"$log_file" 2>&1 </dev/null &
  pid="$!"
fi
echo "$pid" >"$pid_file"
wait_ready
echo "$pid" >"$pid_file"
echo "gizclaw e2e play UI pid=$pid url=http://$listen_addr log=$log_file"
