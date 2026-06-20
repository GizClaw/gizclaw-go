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

require_env GIZCLAW_E2E_VOLC_APP_ID
require_env GIZCLAW_E2E_VOLC_ARK_API_KEY
GIZCLAW_E2E_VOLC_ACCESS_KEY_ID="${GIZCLAW_E2E_VOLC_ACCESS_KEY_ID:-${VOLC_ACCESS_KEY_ID:-${VOLC_ACCESS_KEY:-}}}"
GIZCLAW_E2E_VOLC_SECRET_ACCESS_KEY="${GIZCLAW_E2E_VOLC_SECRET_ACCESS_KEY:-${VOLC_SECRET_ACCESS_KEY:-${VOLC_SECRET_KEY:-}}}"
export GIZCLAW_E2E_VOLC_ACCESS_KEY_ID
export GIZCLAW_E2E_VOLC_SECRET_ACCESS_KEY
require_env GIZCLAW_E2E_VOLC_ACCESS_KEY_ID
require_env GIZCLAW_E2E_VOLC_SECRET_ACCESS_KEY
require_env GIZCLAW_E2E_VOLC_TOKEN
require_env GIZCLAW_E2E_VOLC_VOICE_ID
require_env GIZCLAW_E2E_CLIENT_PUBLIC_KEY

volc_tenant="${GIZCLAW_E2E_VOLC_TENANT:-e2e-volc-tenant}"
GIZCLAW_E2E_VOICE_RESOURCE="${GIZCLAW_E2E_VOICE_RESOURCE:-volc-tenant:${volc_tenant}:${GIZCLAW_E2E_VOLC_VOICE_ID}}"
GIZCLAW_E2E_STORY_NARRATOR_VOICE_ID="${GIZCLAW_E2E_STORY_NARRATOR_VOICE_ID:-zh_female_shaoergushi_mars_bigtts}"
GIZCLAW_E2E_STORY_WUKONG_VOICE_ID="${GIZCLAW_E2E_STORY_WUKONG_VOICE_ID:-zh_male_sunwukong_mars_bigtts}"
GIZCLAW_E2E_STORY_TANGSENG_VOICE_ID="${GIZCLAW_E2E_STORY_TANGSENG_VOICE_ID:-zh_male_tangseng_mars_bigtts}"
GIZCLAW_E2E_STORY_BAJIE_VOICE_ID="${GIZCLAW_E2E_STORY_BAJIE_VOICE_ID:-zh_male_zhubajie_mars_bigtts}"
GIZCLAW_E2E_STORY_MONSTER_VOICE_ID="${GIZCLAW_E2E_STORY_MONSTER_VOICE_ID:-ICL_zh_female_bingjiao3_tob}"
export GIZCLAW_E2E_VOICE_RESOURCE
export GIZCLAW_E2E_STORY_NARRATOR_VOICE_ID
export GIZCLAW_E2E_STORY_WUKONG_VOICE_ID
export GIZCLAW_E2E_STORY_TANGSENG_VOICE_ID
export GIZCLAW_E2E_STORY_BAJIE_VOICE_ID
export GIZCLAW_E2E_STORY_MONSTER_VOICE_ID

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

apply_resource_file() {
  local resource_file="$1"
  case "$(basename "$resource_file")" in
    00-openai-credential.json|01-openai-tenant.json|10-chat-model.json|50-view-acl-credential-openai.json|52-view-acl-model-chat.json)
      if [[ -z "${GIZCLAW_E2E_OPENAI_API_KEY:-}" ]]; then
        return 0
      fi
      require_env GIZCLAW_E2E_OPENAI_BASE_URL
      require_env GIZCLAW_E2E_OPENAI_UPSTREAM_MODEL
      ;;
  esac
  "$gizclaw_bin" admin apply --context "$context_name" -f "$resource_file"
}

voice_resource_id() {
  local speaker_id="$1"
  printf 'volc-tenant:%s:%s' "$volc_tenant" "$speaker_id"
}

configured_voice_resources=(
  "$GIZCLAW_E2E_VOICE_RESOURCE"
  "$(voice_resource_id "$GIZCLAW_E2E_STORY_NARRATOR_VOICE_ID")"
  "$(voice_resource_id "$GIZCLAW_E2E_STORY_WUKONG_VOICE_ID")"
  "$(voice_resource_id "$GIZCLAW_E2E_STORY_TANGSENG_VOICE_ID")"
  "$(voice_resource_id "$GIZCLAW_E2E_STORY_BAJIE_VOICE_ID")"
  "$(voice_resource_id "$GIZCLAW_E2E_STORY_MONSTER_VOICE_ID")"
)

delete_voice_if_present() {
  local voice="$1"
  local output
  if output="$("$gizclaw_bin" admin voices --context "$context_name" get "$voice" 2>&1)"; then
    if grep -Eq '"source"[[:space:]]*:[[:space:]]*"sync"' <<<"$output"; then
      return 0
    fi
    "$gizclaw_bin" admin delete --context "$context_name" Voice "$voice" >/dev/null
    echo "deleted stale non-sync voice=$voice"
    return 0
  fi
  if grep -Eiq 'not[ _-]?found|NOT_FOUND|404' <<<"$output"; then
    return 0
  fi
  printf '%s\n' "$output" >&2
  return 1
}

voice_acl_files=()
for resource_file in "${resource_files[@]}"; do
  case "$(basename "$resource_file")" in
    20-volc-voice.json|2[1-5]-volc-voice-story-*.json)
      continue
      ;;
    56-view-acl-voice*.json)
      voice_acl_files+=("$resource_file")
      continue
      ;;
  esac
  apply_resource_file "$resource_file"
done

for voice in "${configured_voice_resources[@]}"; do
  delete_voice_if_present "$voice"
done

sync_output_dir="$testbench_dir/setup"
rm -rf "$sync_output_dir/voices"
mkdir -p "$sync_output_dir/voices"
sync_output="$sync_output_dir/volc-voices-sync.json"
"$gizclaw_bin" admin volc-tenants --context "$context_name" sync-voices "$volc_tenant" >"$sync_output"

for voice in "${configured_voice_resources[@]}"; do
  safe_voice="${voice//[^A-Za-z0-9_.-]/_}"
  "$gizclaw_bin" admin voices --context "$context_name" get "$voice" >"$sync_output_dir/voices/${safe_voice}.json"
done

for resource_file in "${voice_acl_files[@]}"; do
  apply_resource_file "$resource_file"
done

echo "Applied e2e shared setup resources in context: $context_name"
echo "resource_dir=$resource_dir"
echo "synced_volc_voices=$sync_output"
if [[ -n "${GIZCLAW_E2E_OPENAI_API_KEY:-}" ]]; then
  echo "openai_compat_model=${GIZCLAW_E2E_CHAT_MODEL:-e2e-chat}"
fi
echo "doubao_chat_model=${GIZCLAW_E2E_DOUBAO_2_LITE_MODEL:-e2e-doubao-2-lite-chat}"
echo "tts_model=${GIZCLAW_E2E_TTS_MODEL:-e2e-tts}"
echo "asr_model=${GIZCLAW_E2E_ASR_MODEL:-e2e-asr}"
echo "realtime_model=${GIZCLAW_E2E_REALTIME_MODEL:-e2e-realtime}"
echo "acl_view=${GIZCLAW_E2E_ACL_VIEW:-e2e-client}"
echo "voice=${GIZCLAW_E2E_VOICE_RESOURCE}"
