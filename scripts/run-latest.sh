#!/usr/bin/env bash
set -euo pipefail

repo="${NESTEDVIRT_REPO:-vexxhost/nestedvirt}"
tag="${NESTEDVIRT_TAG:-}"
use_sudo="${NESTEDVIRT_USE_SUDO:-1}"

for command in curl grep id mktemp sed sha256sum tar uname; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "required command not found: $command" >&2
    exit 1
  fi
done

arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

if [ -z "$tag" ]; then
  tag="$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
fi
if [ -z "$tag" ]; then
  echo "could not determine latest release tag" >&2
  exit 1
fi

version="${tag#v}"
asset="nestedvirt_${version}_linux_${arch}.tar.gz"
base="https://github.com/${repo}/releases/download/${tag}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

if [ "$#" -eq 0 ]; then
  set -- scan
elif [ "$1" = "--version" ] || [ "$1" = "-version" ]; then
  set -- version
elif [ "${1#-}" != "$1" ]; then
  set -- scan "$@"
fi

echo "Downloading nestedvirt ${tag} for linux/${arch}..." >&2
curl -fsSL "${base}/${asset}" -o "${tmp}/${asset}"
curl -fsSL "${base}/checksums.txt" -o "${tmp}/checksums.txt"

(
  cd "$tmp"
  grep -F "  ${asset}" checksums.txt | sha256sum -c -
  tar -xzf "$asset"
)

if [ "$1" = "scan" ] && [ "$(id -u)" -ne 0 ] && [ "$use_sudo" != "0" ]; then
  if ! command -v sudo >/dev/null 2>&1; then
    echo "nestedvirt scan requires root; rerun as root or install sudo" >&2
    exit 1
  fi
  sudo "${tmp}/nestedvirt" "$@"
else
  "${tmp}/nestedvirt" "$@"
fi
