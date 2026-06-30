#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export GIZCLAW_E2E_CONFIG_HOME="$script_dir/testdata/config-home-webrtc"

exec "$script_dir/run_tests.sh"
