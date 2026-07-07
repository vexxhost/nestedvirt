// Package nestedvirt detects KVM guests that have used nested virtualization.
//
// The package is intentionally centered on host-local evidence. It reads KVM
// nested_run counters from debugfs through github.com/vexxhost/debugfs, then
// correlates non-zero counters with process metadata from /proc through
// github.com/prometheus/procfs. When the process looks like QEMU, the scanner
// extracts common libvirt/QEMU identity fields such as the guest name and UUID,
// and discovers likely QMP monitor sockets by joining the process fd table with
// /proc/net/unix. It also connects to libvirt through qemu:///system by default
// and enriches findings with domain identity and Nova metadata when available.
//
// A scan reports observed nested virtualization use. It does not prove that a
// guest permanently requires nested virtualization, but a non-zero counter is a
// strong signal that disabling nested virtualization may break that workload.
package nestedvirt
