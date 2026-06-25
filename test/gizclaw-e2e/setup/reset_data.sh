#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
e2e_dir="$repo_root/test/gizclaw-e2e"
testdata_dir="$e2e_dir/testdata"
workspace_dir="$testdata_dir/server-workspace"
resource_dir="$testdata_dir/resources"
bin_path="$testdata_dir/bin/gizclaw"
env_file="${GIZCLAW_E2E_ENV:-$e2e_dir/.env}"
mode="${1:-reset}"

case "$mode" in
  clear|init|reset) ;;
  *)
    echo "usage: $0 [clear|init|reset]" >&2
    exit 2
    ;;
esac

if [[ -f "$env_file" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$env_file"
  set +a
fi

admin_setup_config_home="${GIZCLAW_E2E_ADMIN_SETUP_CONFIG_HOME:-$testdata_dir/admin-config-home}"
admin_setup_context="${GIZCLAW_E2E_ADMIN_SETUP_CONTEXT:-e2e-admin}"
default_client_config_home="$testdata_dir/gizclaw-config-home"
default_client_context="e2e-client"
client_config_home="${GIZCLAW_E2E_CLIENT_CONFIG_HOME:-$testdata_dir/gizclaw-config-home}"
client_context="${GIZCLAW_E2E_CLIENT_CONTEXT:-e2e-client}"

# Preserve Flowcraft runtime placeholders while admin apply expands provider
# credential placeholders from the setup environment.
export input='${input}'

if [[ ! -x "$bin_path" ]]; then
  "$script_dir/build.sh" >/dev/null
fi

clear_data() {
  "$script_dir/stop.sh" server >/dev/null || true
  rm -rf "$workspace_dir/data" "$workspace_dir/gizclaw-server.log" "$workspace_dir/gizclaw-server.pid"
  "$bin_path" migrate --workspace "$workspace_dir"
}

openai_ready() {
  [[ -n "${GIZCLAW_E2E_OPENAI_API_KEY:-}" ]]
}

volc_ready() {
  local name
  for name in \
    GIZCLAW_E2E_VOLC_APP_ID \
    GIZCLAW_E2E_VOLC_ARK_API_KEY \
    GIZCLAW_E2E_VOLC_ACCESS_KEY_ID \
    GIZCLAW_E2E_VOLC_SECRET_ACCESS_KEY \
    GIZCLAW_E2E_VOLC_TOKEN; do
    if [[ -z "${!name:-}" ]]; then
      return 1
    fi
  done
  return 0
}

init_data() {
  "$script_dir/start-server.sh" >/dev/null

  XDG_CONFIG_HOME="$default_client_config_home" \
    "$bin_path" connect set-name e2e-client --context "$default_client_context" >/dev/null
  if [[ "$client_config_home" != "$default_client_config_home" || "$client_context" != "$default_client_context" ]]; then
    XDG_CONFIG_HOME="$client_config_home" \
      "$bin_path" connect set-name e2e-client --context "$client_context" >/dev/null
  fi

  shopt -s nullglob
  local resource_files=("$resource_dir"/*.json)
  shopt -u nullglob
  if [[ ${#resource_files[@]} -eq 0 ]]; then
    echo "no resource fixtures found in $resource_dir" >&2
    exit 2
  fi

  apply_resource() {
    local resource_file="$1"
    case "$(basename "$resource_file")" in
      000-openai-credential.json|001-openai-tenant.json|010-chat-model.json|050-view-acl-credential-openai.json|052-view-acl-model-chat.json)
        if ! openai_ready; then
          return 0
        fi
        ;;
      002-volc-credential.json|003-volc-tenant.json|011-tts-model.json|012-asr-model.json|013-realtime-model.json|014-doubao-2-lite-chat-model.json|015-ast-translate-model.json|051-view-acl-credential-volc.json|053-view-acl-model-tts.json|054-view-acl-model-asr.json|055-view-acl-model-ast-translate.json|055-view-acl-model-realtime.json|059-view-acl-model-doubao-2-lite-chat.json)
        if ! volc_ready; then
          return 0
        fi
        ;;
    esac
    XDG_CONFIG_HOME="$admin_setup_config_home" \
      "$bin_path" admin apply --context "$admin_setup_context" -f "$resource_file"
  }

  local voice_acl_files=()
  for resource_file in "${resource_files[@]}"; do
    case "$(basename "$resource_file")" in
      056-view-acl-voice*.json)
        voice_acl_files+=("$resource_file")
        continue
        ;;
    esac
    apply_resource "$resource_file"
  done

  if ! volc_ready; then
    voice_acl_files=()
  else
    XDG_CONFIG_HOME="$admin_setup_config_home" \
      "$bin_path" admin volc-tenants --context "$admin_setup_context" sync-voices e2e-volc-tenant >/dev/null
  fi

  if [[ ${#voice_acl_files[@]} -gt 0 ]]; then
    for resource_file in "${voice_acl_files[@]}"; do
      apply_resource "$resource_file"
    done
  fi

  upload_firmware_asset() {
    local firmware_id="e2e-rpc-firmware"
    local channel="stable"
    local bin="main"
    local asset_path="$resource_dir/assets/firmware/e2e-rpc-firmware-main.tar"
    if [[ ! -f "$asset_path" ]]; then
      echo "missing firmware fixture asset: $asset_path" >&2
      exit 2
    fi
    XDG_CONFIG_HOME="$admin_setup_config_home" \
      "$bin_path" admin firmwares upload-bin "$firmware_id" --channel "$channel" --bin "$bin" -f "$asset_path" --context "$admin_setup_context" >/dev/null
  }

  upload_firmware_asset

  (
    cd "$repo_root"
    go run ./test/gizclaw-e2e/setup/seed_rpc_history.go
  )

}

if [[ "$mode" == "clear" || "$mode" == "reset" ]]; then
  clear_data
fi
if [[ "$mode" == "init" || "$mode" == "reset" ]]; then
  init_data
fi
