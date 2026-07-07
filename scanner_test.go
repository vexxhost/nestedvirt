package nestedvirt

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestScanFindsObservedQEMUVMs(t *testing.T) {
	debugRoot, procRoot := testHostFilesystems(t)

	writeNestedRun(t, debugRoot, "1111-0", "0\n")
	writeNestedRun(t, debugRoot, "2222-0", "7\n")
	writeNestedRun(t, debugRoot, "2222-1", "5\n")
	writeNestedRun(t, debugRoot, "3333-0", "1\n")

	writeProc(t, procRoot, 2222, procFixture{
		comm: "qemu-system-x86",
		exe:  "/usr/libexec/qemu-kvm",
		cmdline: []string{
			"/usr/libexec/qemu-kvm",
			"-name", "guest=instance-0000002a,debug-threads=on",
			"-uuid", "11112222-3333-4444-5555-666677778888",
		},
	})
	writeSocketFD(t, procRoot, 2222, 12, 123456)
	writeUnixSocket(t, procRoot, 123456, "/var/lib/libvirt/qemu/domain-7-instance/monitor.sock")
	writeProc(t, procRoot, 3333, procFixture{
		comm:    "kvm-test",
		exe:     "/usr/bin/kvm-test",
		cmdline: []string{"/usr/bin/kvm-test"},
	})

	scanner, err := NewScanner(
		WithDebugFSMount(debugRoot),
		WithProcFSMount(procRoot),
		WithClock(func() time.Time { return time.Unix(100, 0) }),
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	wantTime := time.Unix(100, 0).UTC()
	if !got.ScannedAt.Equal(wantTime) {
		t.Fatalf("ScannedAt = %s, want %s", got.ScannedAt, wantTime)
	}

	wantSummary := Summary{
		NestedRunCounters:  4,
		ObservedProcesses:  2,
		QEMUProcesses:      1,
		MonitorSockets:     1,
		NestedVirtObserved: true,
	}
	if !reflect.DeepEqual(got.Summary, wantSummary) {
		t.Fatalf("Summary = %#v, want %#v", got.Summary, wantSummary)
	}

	if len(got.Findings) != 2 {
		t.Fatalf("Findings length = %d, want 2: %#v", len(got.Findings), got.Findings)
	}

	qemu := got.Findings[0]
	if qemu.Process.PID != 2222 {
		t.Fatalf("first PID = %d, want 2222", qemu.Process.PID)
	}
	if qemu.Process.Kind != ProcessKindQEMU {
		t.Fatalf("qemu kind = %s, want %s", qemu.Process.Kind, ProcessKindQEMU)
	}
	if qemu.NestedRunCount != 12 {
		t.Fatalf("qemu NestedRunCount = %d, want 12", qemu.NestedRunCount)
	}
	if qemu.VM == nil {
		t.Fatal("qemu VM identity is nil")
	}
	if qemu.VM.Name != "instance-0000002a" {
		t.Fatalf("qemu VM name = %q, want instance-0000002a", qemu.VM.Name)
	}
	if qemu.VM.UUID != "11112222-3333-4444-5555-666677778888" {
		t.Fatalf("qemu VM UUID = %q, want expected UUID", qemu.VM.UUID)
	}
	wantSocket := MonitorSocket{
		FD:     12,
		Inode:  123456,
		Path:   "/var/lib/libvirt/qemu/domain-7-instance/monitor.sock",
		Source: monitorSocketSourceProc,
	}
	if !reflect.DeepEqual(qemu.MonitorSockets, []MonitorSocket{wantSocket}) {
		t.Fatalf("qemu MonitorSockets = %#v, want %#v", qemu.MonitorSockets, []MonitorSocket{wantSocket})
	}

	other := got.Findings[1]
	if other.Process.PID != 3333 {
		t.Fatalf("second PID = %d, want 3333", other.Process.PID)
	}
	if other.Process.Kind != ProcessKindOther {
		t.Fatalf("other kind = %s, want %s", other.Process.Kind, ProcessKindOther)
	}
	if other.VM != nil {
		t.Fatalf("other VM identity = %#v, want nil", other.VM)
	}
}

func TestScanNoObservation(t *testing.T) {
	debugRoot, procRoot := testHostFilesystems(t)

	writeNestedRun(t, debugRoot, "1111-0", "0\n")
	writeNestedRun(t, debugRoot, "2222-0", "0\n")

	report, err := Scan(
		context.Background(),
		WithDebugFSMount(debugRoot),
		WithProcFSMount(procRoot),
	)
	if err != nil {
		t.Fatal(err)
	}

	if report.NestedVirtObserved() {
		t.Fatalf("NestedVirtObserved() = true, want false: %#v", report)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("Findings length = %d, want 0", len(report.Findings))
	}
	if report.Summary.NestedRunCounters != 2 {
		t.Fatalf("NestedRunCounters = %d, want 2", report.Summary.NestedRunCounters)
	}
}

func TestScanReportsProcessInspectionErrors(t *testing.T) {
	debugRoot, procRoot := testHostFilesystems(t)

	writeNestedRun(t, debugRoot, "4444-0", "1\n")

	report, err := Scan(
		context.Background(),
		WithDebugFSMount(debugRoot),
		WithProcFSMount(procRoot),
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.Findings) != 1 {
		t.Fatalf("Findings length = %d, want 1", len(report.Findings))
	}

	finding := report.Findings[0]
	if finding.Process.PID != 4444 {
		t.Fatalf("PID = %d, want 4444", finding.Process.PID)
	}
	if finding.Process.Kind != ProcessKindUnknown {
		t.Fatalf("Kind = %s, want %s", finding.Process.Kind, ProcessKindUnknown)
	}
	if len(finding.Errors) != 1 {
		t.Fatalf("Errors length = %d, want 1: %#v", len(finding.Errors), finding.Errors)
	}
}

func testHostFilesystems(t *testing.T) (string, string) {
	t.Helper()

	root := t.TempDir()
	debugRoot := filepath.Join(root, "debugfs")
	procRoot := filepath.Join(root, "proc")

	if err := os.MkdirAll(filepath.Join(debugRoot, "kvm"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(procRoot, "net"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeUnixSocketHeader(t, procRoot)

	return debugRoot, procRoot
}

func writeNestedRun(t *testing.T, debugRoot, dir, value string) {
	t.Helper()

	path := filepath.Join(debugRoot, "kvm", dir)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "nested_run"), []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

type procFixture struct {
	comm    string
	exe     string
	cmdline []string
}

func writeProc(t *testing.T, procRoot string, pid int, fixture procFixture) {
	t.Helper()

	path := filepath.Join(procRoot, strconv.Itoa(pid))
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "comm"), []byte(fixture.comm+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "cmdline"), []byte(joinNull(fixture.cmdline)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(fixture.exe, filepath.Join(path, "exe")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(path, "fd"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func joinNull(values []string) string {
	var out []byte
	for _, value := range values {
		out = append(out, value...)
		out = append(out, 0)
	}
	return string(out)
}

func writeSocketFD(t *testing.T, procRoot string, pid, fd int, inode uint64) {
	t.Helper()

	fdPath := filepath.Join(procRoot, strconv.Itoa(pid), "fd")
	if err := os.MkdirAll(fdPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("socket:["+strconv.FormatUint(inode, 10)+"]", filepath.Join(fdPath, strconv.Itoa(fd))); err != nil {
		t.Fatal(err)
	}
}

func writeUnixSocketHeader(t *testing.T, procRoot string) {
	t.Helper()

	path := filepath.Join(procRoot, "net", "unix")
	header := "Num       RefCount Protocol Flags    Type St Inode Path\n"
	if err := os.WriteFile(path, []byte(header), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeUnixSocket(t *testing.T, procRoot string, inode uint64, socketPath string) {
	t.Helper()

	path := filepath.Join(procRoot, "net", "unix")
	line := "0000000000000000: 00000002 00000000 00010000 0001 01 " + strconv.FormatUint(inode, 10) + " " + socketPath + "\n"

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if _, err := file.WriteString(line); err != nil {
		t.Fatal(err)
	}
}
