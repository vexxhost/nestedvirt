package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"

	"github.com/vexxhost/debugfs"
	"github.com/vexxhost/nestedvirt"
)

const (
	// ExitNoObservation means the scan completed and no nested virtualization
	// usage was observed.
	ExitNoObservation = 0

	// ExitObserved means the scan completed and nested virtualization usage was
	// observed.
	ExitObserved = 1

	// ExitError means the scan failed.
	ExitError = 2
)

// Run executes the nestedvirt CLI.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	args = trimScanCommand(args)

	flags := flag.NewFlagSet("nestedvirt scan", flag.ContinueOnError)
	flags.SetOutput(stderr)

	debugfsMount := flags.String("debugfs", debugfs.DefaultMountPoint, "debugfs mount point")
	procfsMount := flags.String("procfs", "/proc", "procfs mount point")
	jsonOutput := flags.Bool("json", false, "write JSON output")

	if err := flags.Parse(args); err != nil {
		return ExitError
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected argument: %s\n", flags.Arg(0))
		flags.Usage()
		return ExitError
	}

	report, err := nestedvirt.Scan(
		ctx,
		nestedvirt.WithDebugFSMount(*debugfsMount),
		nestedvirt.WithProcFSMount(*procfsMount),
	)
	if err != nil {
		fmt.Fprintf(stderr, "scan failed: %v\n", err)
		return ExitError
	}

	if *jsonOutput {
		if err := writeJSON(stdout, report); err != nil {
			fmt.Fprintf(stderr, "write JSON output: %v\n", err)
			return ExitError
		}
	} else {
		if err := writeText(stdout, report); err != nil {
			fmt.Fprintf(stderr, "write text output: %v\n", err)
			return ExitError
		}
	}

	if report.NestedVirtObserved() {
		return ExitObserved
	}

	return ExitNoObservation
}

func trimScanCommand(args []string) []string {
	if len(args) > 0 && args[0] == "scan" {
		return args[1:]
	}
	return args
}

func writeJSON(w io.Writer, report nestedvirt.Report) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeText(w io.Writer, report nestedvirt.Report) error {
	if !report.NestedVirtObserved() {
		if _, err := fmt.Fprintln(w, "No nested virtualization usage observed."); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		_, err := fmt.Fprintln(w, "Final result: requires nested virt: no observed evidence.")
		return err
	}

	if _, err := fmt.Fprintln(w, "Nested virtualization usage observed:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "PID\tKIND\tPROCESS\tVM\tMONITOR\tNESTED_RUNS"); err != nil {
		return err
	}

	for _, finding := range report.Findings {
		if _, err := fmt.Fprintf(
			table,
			"%d\t%s\t%s\t%s\t%s\t%d\n",
			finding.Process.PID,
			finding.Process.Kind,
			processName(finding.Process),
			vmName(finding.VM),
			monitorSocketName(finding.MonitorSockets),
			finding.NestedRunCount,
		); err != nil {
			return err
		}
	}

	if err := table.Flush(); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "Final result: requires nested virt: unknown; usage was observed, so disabling nested virt may break these VMs.")
	return err
}

func processName(process nestedvirt.Process) string {
	if process.Command != "" {
		return process.Command
	}
	if process.Executable != "" {
		return filepath.Base(process.Executable)
	}
	return "-"
}

func monitorSocketName(sockets []nestedvirt.MonitorSocket) string {
	if len(sockets) == 0 {
		return "-"
	}
	return sockets[0].Path
}

func vmName(vm *nestedvirt.VMIdentity) string {
	if vm == nil {
		return "-"
	}
	if vm.Name != "" && vm.UUID != "" {
		return vm.Name + " (" + vm.UUID + ")"
	}
	if vm.Name != "" {
		return vm.Name
	}
	if vm.UUID != "" {
		return vm.UUID
	}
	return "-"
}
