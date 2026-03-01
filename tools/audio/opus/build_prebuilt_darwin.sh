#!/usr/bin/env bash

set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
	echo "[build_prebuilt_darwin] error: this script must run on macOS" >&2
	exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SRC_DIR="${ROOT_DIR}/third_party/audio/libopus"

if [[ ! -f "${SRC_DIR}/CMakeLists.txt" ]]; then
	echo "[build_prebuilt_darwin] error: missing libopus source at ${SRC_DIR}" >&2
	echo "[build_prebuilt_darwin] hint: run 'git submodule update --init --recursive third_party/audio/libopus'" >&2
	exit 1
fi

ARCH_INPUT="${TARGET_ARCH:-$(uname -m)}"
case "${ARCH_INPUT}" in
x86_64 | amd64)
	PLATFORM="darwin-amd64"
	CMAKE_ARCH="x86_64"
	;;
aarch64 | arm64)
	PLATFORM="darwin-arm64"
	CMAKE_ARCH="arm64"
	;;
*)
	echo "[build_prebuilt_darwin] error: unsupported arch ${ARCH_INPUT}" >&2
	exit 1
	;;
esac

WORK_ROOT="${ROOT_DIR}/.tmp/opus-build/${PLATFORM}"
BUILD_DIR="${WORK_ROOT}/build"
INSTALL_ROOT="${ROOT_DIR}/.tmp/opus-prebuilt/${PLATFORM}"

need_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "[build_prebuilt_darwin] error: missing required command: $1" >&2
		exit 1
	fi
}

need_cmd cmake
need_cmd git

rm -rf "${BUILD_DIR}" "${INSTALL_ROOT}"
mkdir -p "${BUILD_DIR}" "${INSTALL_ROOT}"

cmake -S "${SRC_DIR}" -B "${BUILD_DIR}" \
	-DCMAKE_BUILD_TYPE=Release \
	-DCMAKE_OSX_ARCHITECTURES="${CMAKE_ARCH}" \
	-DBUILD_SHARED_LIBS=OFF \
	-DOPUS_BUILD_SHARED_LIBRARY=OFF \
	-DOPUS_BUILD_PROGRAMS=OFF \
	-DOPUS_BUILD_TESTING=OFF \
	-DOPUS_INSTALL_PKG_CONFIG_MODULE=OFF \
	-DOPUS_INSTALL_CMAKE_CONFIG_MODULE=OFF \
	-DCMAKE_INSTALL_PREFIX="${INSTALL_ROOT}"

JOBS="$(sysctl -n hw.ncpu)"
if [[ "${JOBS}" -lt 1 ]]; then
	JOBS=4
fi

cmake --build "${BUILD_DIR}" -- -j"${JOBS}"
cmake --install "${BUILD_DIR}"

if [[ ! -f "${INSTALL_ROOT}/lib/libopus.a" ]]; then
	echo "[build_prebuilt_darwin] error: missing ${INSTALL_ROOT}/lib/libopus.a" >&2
	exit 1
fi

if [[ ! -f "${INSTALL_ROOT}/include/opus/opus.h" ]]; then
	echo "[build_prebuilt_darwin] error: missing ${INSTALL_ROOT}/include/opus/opus.h" >&2
	exit 1
fi

OPUS_SOURCE_REV="$(git -C "${SRC_DIR}" rev-parse HEAD)"
OPUS_SOURCE_TAG="$(git -C "${SRC_DIR}" describe --tags --always)"

cat >"${INSTALL_ROOT}/build.env" <<EOF
OPUS_SOURCE_TAG=${OPUS_SOURCE_TAG}
OPUS_SOURCE_REV=${OPUS_SOURCE_REV}
OPUS_BUILD_SHARED_LIBRARY=OFF
OPUS_BUILD_PROGRAMS=OFF
OPUS_BUILD_TESTING=OFF
TARGET_PLATFORM=${PLATFORM}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
EOF

echo "[build_prebuilt_darwin] built opus artifacts at ${INSTALL_ROOT}"
