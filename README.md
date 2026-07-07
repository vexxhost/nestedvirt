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

To download the latest release, verify the checksum, extract it, and run a scan
on a Linux compute host:

```sh
curl -fsSL https://raw.githubusercontent.com/vexxhost/nestedvirt/main/scripts/run-latest.sh | bash
```

Pass scanner flags after `bash -s --`; a leading flag implies `scan`:

```sh
curl -fsSL https://raw.githubusercontent.com/vexxhost/nestedvirt/main/scripts/run-latest.sh | bash -s -- --json
```

Set `NESTEDVIRT_TAG=v0.1.0` to pin a specific release.

Exit codes:

- `0`: scan completed and no nested virtualization usage was observed
- `1`: scan completed and nested virtualization usage was observed
- `2`: scan failed

API documentation lives in the Go package docs:

https://pkg.go.dev/github.com/vexxhost/nestedvirt
