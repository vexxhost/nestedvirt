# nestedvirt

Host-local detection for KVM guests that have used nested virtualization.

The library reads KVM `nested_run` counters from debugfs, correlates non-zero
counters with `/proc`, and identifies QEMU guests when their command line
contains common libvirt/QEMU fields such as `-name guest=...` and `-uuid`.
For QEMU processes, it also discovers likely monitor/QMP Unix sockets by
joining `/proc/<pid>/fd` socket inodes with `/proc/net/unix`.

The normal build includes pure-Go libvirt RPC support. The scanner connects to
`qemu:///system` by default and enriches QEMU findings with libvirt domain
identity and OpenStack Nova metadata when the metadata is present. It does not
link against `libvirt.so`.

The command line tool is intended for compute-host triage:

```console
nestedvirt scan
nestedvirt scan --json
```

To download the latest release, verify the checksum, extract it, and run a
scan on a Linux compute host:

```sh
set -euo pipefail

repo="vexxhost/nestedvirt"

arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

tag="$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
if [ -z "$tag" ]; then
  echo "could not determine latest release tag" >&2
  exit 1
fi

version="${tag#v}"
asset="nestedvirt_${version}_linux_${arch}.tar.gz"
base="https://github.com/${repo}/releases/download/${tag}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

curl -fsSL "${base}/${asset}" -o "${tmp}/${asset}"
curl -fsSL "${base}/checksums.txt" -o "${tmp}/checksums.txt"

(
  cd "$tmp"
  grep -F "  ${asset}" checksums.txt | sha256sum -c -
  tar -xzf "$asset"
  sudo ./nestedvirt scan
)
```

Use `sudo ./nestedvirt scan --json` in the last line for machine-readable
output.

Exit codes:

- `0`: scan completed and no nested virtualization usage was observed
- `1`: scan completed and nested virtualization usage was observed
- `2`: scan failed

API documentation lives in the Go package docs:

https://pkg.go.dev/github.com/vexxhost/nestedvirt
