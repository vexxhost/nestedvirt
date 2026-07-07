package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRunTextObserved(t *testing.T) {
	debugRoot, procRoot := testHostFilesystems(t)
	writeNestedRun(t, debugRoot, "2222-0", "3\n")
	writeProc(t, procRoot, 2222, procFixture{
		comm:    "qemu-system-x86",
		exe:     "/usr/libexec/qemu-kvm",
		cmdline: []string{"/usr/libexec/qemu-kvm", "-name", "guest=instance-0000002a", "-uuid", "11112222-3333-4444-5555-666677778888"},
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"scan", "--debugfs", debugRoot, "--procfs", procRoot}, &stdout, &stderr)
	if code != ExitObserved {
		t.Fatalf("Run() code = %d, want %d; stderr=%s", code, ExitObserved, stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{
		"Nested virtualization usage observed",
		"instance-0000002a",
		"requires nested virt: unknown",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
}

func TestRunJSONObserved(t *testing.T) {
	debugRoot, procRoot := testHostFilesystems(t)
	writeNestedRun(t, debugRoot, "2222-0", "3\n")
	writeProc(t, procRoot, 2222, procFixture{
		comm:    "qemu-system-x86",
		exe:     "/usr/libexec/qemu-kvm",
		cmdline: []string{"/usr/libexec/qemu-kvm", "-name", "guest=instance-0000002a"},
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--debugfs", debugRoot, "--procfs", procRoot, "--json"}, &stdout, &stderr)
	if code != ExitObserved {
		t.Fatalf("Run() code = %d, want %d; stderr=%s", code, ExitObserved, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"nested_virt_observed": true`) {
		t.Fatalf("JSON output missing observed summary:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"name": "instance-0000002a"`) {
		t.Fatalf("JSON output missing VM name:\n%s", stdout.String())
	}
}

func TestRunNoObservation(t *testing.T) {
	debugRoot, procRoot := testHostFilesystems(t)
	writeNestedRun(t, debugRoot, "2222-0", "0\n")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--debugfs", debugRoot, "--procfs", procRoot}, &stdout, &stderr)
	if code != ExitNoObservation {
		t.Fatalf("Run() code = %d, want %d; stderr=%s", code, ExitNoObservation, stderr.String())
	}
	if !strings.Contains(stdout.String(), "requires nested virt: no observed evidence") {
		t.Fatalf("stdout missing final result:\n%s", stdout.String())
	}
}

type procFixture struct {
	comm    string
	exe     string
	cmdline []string
}

func testHostFilesystems(t *testing.T) (string, string) {
	t.Helper()

	root := t.TempDir()
	debugRoot := filepath.Join(root, "debugfs")
	procRoot := filepath.Join(root, "proc")

	if err := os.MkdirAll(filepath.Join(debugRoot, "kvm"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(procRoot, 0o755); err != nil {
		t.Fatal(err)
	}

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
}

func joinNull(values []string) string {
	var out []byte
	for _, value := range values {
		out = append(out, value...)
		out = append(out, 0)
	}
	return string(out)
}
