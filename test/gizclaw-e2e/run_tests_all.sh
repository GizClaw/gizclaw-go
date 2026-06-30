#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> run giznet e2e"
"$script_dir/run_giznet_tests.sh"

echo "==> run WebRTC e2e"
"$script_dir/run_webrtc_tests.sh"

echo "==> all e2e runs completed"
