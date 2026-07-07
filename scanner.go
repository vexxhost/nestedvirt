package nestedvirt

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/prometheus/procfs"
	"github.com/vexxhost/debugfs"
	"github.com/vexxhost/debugfs/kvm"
)

// Scanner correlates KVM debugfs counters with process metadata.
type Scanner struct {
	kvmFS       kvm.FS
	procFS      procfs.FS
	procFSMount string
	now         func() time.Time
}

type scannerConfig struct {
	debugfsMount string
	procfsMount  string
	debugfs      *debugfs.FS
	procfs       *procfs.FS
	now          func() time.Time
}

// Option configures a Scanner.
type Option func(*scannerConfig) error

// NewScanner creates a Scanner. By default it reads /sys/kernel/debug and
// /proc.
func NewScanner(opts ...Option) (*Scanner, error) {
	cfg := scannerConfig{
		debugfsMount: debugfs.DefaultMountPoint,
		procfsMount:  "/proc",
		now:          func() time.Time { return time.Now().UTC() },
	}

	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	dfs, err := configuredDebugFS(cfg)
	if err != nil {
		return nil, err
	}

	pfs, err := configuredProcFS(cfg)
	if err != nil {
		return nil, err
	}

	return &Scanner{
		kvmFS:       kvm.NewFS(dfs),
		procFS:      pfs,
		procFSMount: filepath.Clean(cfg.procfsMount),
		now:         cfg.now,
	}, nil
}

// WithDebugFSMount configures the debugfs mount point.
func WithDebugFSMount(mountPoint string) Option {
	return func(cfg *scannerConfig) error {
		cfg.debugfsMount = mountPoint
		cfg.debugfs = nil
		return nil
	}
}

// WithDebugFS configures the debugfs reader directly.
func WithDebugFS(fs debugfs.FS) Option {
	return func(cfg *scannerConfig) error {
		cfg.debugfs = &fs
		return nil
	}
}

// WithProcFSMount configures the procfs mount point.
func WithProcFSMount(mountPoint string) Option {
	return func(cfg *scannerConfig) error {
		cfg.procfsMount = mountPoint
		cfg.procfs = nil
		return nil
	}
}

// WithProcFS configures the procfs reader directly.
func WithProcFS(fs procfs.FS) Option {
	return func(cfg *scannerConfig) error {
		cfg.procfs = &fs
		return nil
	}
}

// WithClock configures the timestamp source used in reports.
func WithClock(now func() time.Time) Option {
	return func(cfg *scannerConfig) error {
		if now == nil {
			return fmt.Errorf("clock: nil")
		}
		cfg.now = func() time.Time { return now().UTC() }
		return nil
	}
}

// Scan creates a default Scanner with opts and runs it.
func Scan(ctx context.Context, opts ...Option) (Report, error) {
	scanner, err := NewScanner(opts...)
	if err != nil {
		return Report{}, err
	}
	return scanner.Scan(ctx)
}

// Scan reads KVM nested_run counters and returns processes with observed nested
// virtualization use.
func (s *Scanner) Scan(ctx context.Context) (Report, error) {
	runs, err := s.kvmFS.NestedRuns(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("read KVM nested_run counters: %w", err)
	}

	report := Report{
		ScannedAt: s.now(),
		Summary: Summary{
			NestedRunCounters: len(runs),
		},
	}

	for _, observed := range aggregateNestedRuns(runs) {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		if observed.total == 0 {
			continue
		}

		finding := Finding{
			Process: Process{
				PID:  observed.pid,
				Kind: ProcessKindUnknown,
			},
			NestedRunCount:    observed.total,
			NestedRunCounters: observed.counters,
			Requirement:       RequirementUnknown,
		}

		inspected, errors := s.inspectProcess(observed.pid)
		finding.Process = inspected.Process
		finding.Errors = append(finding.Errors, errors...)

		if inspected.Process.Kind == ProcessKindQEMU {
			finding.VM = qemuVMIdentity(inspected.CommandLine)
			sockets, socketErrors := s.discoverMonitorSockets(observed.pid)
			finding.MonitorSockets = sockets
			finding.Errors = append(finding.Errors, socketErrors...)
		}

		report.Findings = append(report.Findings, finding)
	}

	sort.Slice(report.Findings, func(i, j int) bool {
		return report.Findings[i].Process.PID < report.Findings[j].Process.PID
	})

	report.Summary.ObservedProcesses = len(report.Findings)
	report.Summary.NestedVirtObserved = len(report.Findings) > 0
	for _, finding := range report.Findings {
		switch finding.Process.Kind {
		case ProcessKindQEMU:
			report.Summary.QEMUProcesses++
			report.Summary.MonitorSockets += len(finding.MonitorSockets)
		case ProcessKindUnknown:
			report.Summary.UnknownProcesses++
		}
	}

	return report, nil
}

func configuredDebugFS(cfg scannerConfig) (debugfs.FS, error) {
	if cfg.debugfs != nil {
		return *cfg.debugfs, nil
	}

	dfs, err := debugfs.NewFS(cfg.debugfsMount)
	if err != nil {
		return debugfs.FS{}, fmt.Errorf("open debugfs: %w", err)
	}

	return dfs, nil
}

func configuredProcFS(cfg scannerConfig) (procfs.FS, error) {
	if cfg.procfs != nil {
		return *cfg.procfs, nil
	}

	pfs, err := procfs.NewFS(cfg.procfsMount)
	if err != nil {
		return procfs.FS{}, fmt.Errorf("open procfs: %w", err)
	}

	return pfs, nil
}

type observedNestedRuns struct {
	pid      int
	total    uint64
	counters []NestedRunCounter
}

func aggregateNestedRuns(runs []kvm.NestedRun) []observedNestedRuns {
	byPID := make(map[int]*observedNestedRuns)

	for _, run := range runs {
		observed := byPID[run.PID]
		if observed == nil {
			observed = &observedNestedRuns{pid: run.PID}
			byPID[run.PID] = observed
		}

		observed.total += run.Count
		observed.counters = append(observed.counters, NestedRunCounter{
			Path:  run.Path,
			Count: run.Count,
		})
	}

	aggregated := make([]observedNestedRuns, 0, len(byPID))
	for _, observed := range byPID {
		sort.Slice(observed.counters, func(i, j int) bool {
			return observed.counters[i].Path < observed.counters[j].Path
		})
		aggregated = append(aggregated, *observed)
	}

	sort.Slice(aggregated, func(i, j int) bool {
		return aggregated[i].pid < aggregated[j].pid
	})

	return aggregated
}
