package nestedvirt

import "time"

// ProcessKind classifies the userspace process attached to a KVM VM.
type ProcessKind string

const (
	// ProcessKindUnknown means the scanner could not read enough process
	// metadata to classify the process.
	ProcessKindUnknown ProcessKind = "unknown"

	// ProcessKindQEMU means the process looks like QEMU or qemu-kvm.
	ProcessKindQEMU ProcessKind = "qemu"

	// ProcessKindOther means the process was readable but did not look like
	// QEMU.
	ProcessKindOther ProcessKind = "other"
)

// Requirement describes whether a workload should be treated as requiring
// nested virtualization.
type Requirement string

const (
	// RequirementUnknown is used because a nested_run counter only proves prior
	// use, not a durable contractual requirement.
	RequirementUnknown Requirement = "unknown"
)

// Report is the result of a host scan.
type Report struct {
	ScannedAt time.Time `json:"scanned_at"`
	Summary   Summary   `json:"summary"`
	Findings  []Finding `json:"findings"`
}

// Summary contains aggregate scan counts.
type Summary struct {
	NestedRunCounters  int  `json:"nested_run_counters"`
	ObservedProcesses  int  `json:"observed_processes"`
	QEMUProcesses      int  `json:"qemu_processes"`
	UnknownProcesses   int  `json:"unknown_processes"`
	MonitorSockets     int  `json:"monitor_sockets"`
	NestedVirtObserved bool `json:"nested_virt_observed"`
}

// Finding describes one process whose KVM VM has a non-zero nested_run counter.
type Finding struct {
	Process           Process            `json:"process"`
	VM                *VMIdentity        `json:"vm,omitempty"`
	MonitorSockets    []MonitorSocket    `json:"monitor_sockets,omitempty"`
	NestedRunCount    uint64             `json:"nested_run_count"`
	NestedRunCounters []NestedRunCounter `json:"nested_run_counters"`
	Requirement       Requirement        `json:"requirement"`
	Errors            []FindingError     `json:"errors,omitempty"`
}

// Process describes the userspace process attached to a KVM VM.
type Process struct {
	PID        int         `json:"pid"`
	Command    string      `json:"command,omitempty"`
	Executable string      `json:"executable,omitempty"`
	Kind       ProcessKind `json:"kind"`
}

// VMIdentity contains VM identity discovered from a QEMU command line.
type VMIdentity struct {
	Name    string   `json:"name,omitempty"`
	UUID    string   `json:"uuid,omitempty"`
	Sources []string `json:"sources,omitempty"`
}

// MonitorSocket describes a Unix socket that looks like a QEMU monitor or QMP
// endpoint for the process.
type MonitorSocket struct {
	FD     int    `json:"fd"`
	Inode  uint64 `json:"inode"`
	Path   string `json:"path"`
	Source string `json:"source"`
}

// NestedRunCounter identifies one debugfs nested_run counter that contributed
// to a finding.
type NestedRunCounter struct {
	Path  string `json:"path"`
	Count uint64 `json:"count"`
}

// FindingError records process-inspection errors that did not prevent the scan
// from reporting the nested_run evidence.
type FindingError struct {
	PID       int    `json:"pid"`
	Operation string `json:"operation"`
	Error     string `json:"error"`
}

// NestedVirtObserved reports whether any process has a non-zero nested_run
// counter.
func (r Report) NestedVirtObserved() bool {
	return r.Summary.NestedVirtObserved
}
