# nestedvirt

Host-local detection for KVM guests that have used nested virtualization.

The library reads KVM `nested_run` counters from debugfs, correlates non-zero
counters with `/proc`, and identifies QEMU guests when their command line
contains common libvirt/QEMU fields such as `-name guest=...` and `-uuid`.

The command line tool is intended for compute-host triage:

```console
nestedvirt scan
nestedvirt scan --json
```

Exit codes:

- `0`: scan completed and no nested virtualization usage was observed
- `1`: scan completed and nested virtualization usage was observed
- `2`: scan failed

API documentation lives in the Go package docs:

https://pkg.go.dev/github.com/vexxhost/nestedvirt
