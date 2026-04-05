#!/usr/bin/env sh
set -eu

PREFIX="${PREFIX:-/usr/local}"
SOURCE_BIN="${1:-bin/viaduct}"
SOURCE_WEB="${2:-web}"
TARGET_BIN_DIR="${PREFIX}/bin"
TARGET_SHARE_DIR="${PREFIX}/share/viaduct"

if [ ! -f "${SOURCE_BIN}" ]; then
  echo "viaduct install: binary not found at ${SOURCE_BIN}" >&2
  exit 1
fi

mkdir -p "${TARGET_BIN_DIR}" "${TARGET_SHARE_DIR}"
cp "${SOURCE_BIN}" "${TARGET_BIN_DIR}/viaduct"
chmod 0755 "${TARGET_BIN_DIR}/viaduct"

if [ -d "${SOURCE_WEB}" ]; then
  rm -rf "${TARGET_SHARE_DIR}/web"
  mkdir -p "${TARGET_SHARE_DIR}"
  cp -R "${SOURCE_WEB}" "${TARGET_SHARE_DIR}/web"
fi

echo "Installed Viaduct to ${PREFIX}"
