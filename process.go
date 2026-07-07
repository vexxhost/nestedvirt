package nestedvirt

import (
	"path/filepath"
	"strings"
)

type inspectedProcess struct {
	Process
	CommandLine []string
}

func (s *Scanner) inspectProcess(pid int) (inspectedProcess, []FindingError) {
	result := inspectedProcess{
		Process: Process{
			PID:  pid,
			Kind: ProcessKindUnknown,
		},
	}

	proc, err := s.procFS.Proc(pid)
	if err != nil {
		return result, []FindingError{processError(pid, "open", err)}
	}

	var errors []FindingError

	cmdline, err := proc.CmdLine()
	if err != nil {
		errors = append(errors, processError(pid, "cmdline", err))
	} else {
		result.CommandLine = cmdline
	}

	command, err := proc.Comm()
	if err != nil {
		errors = append(errors, processError(pid, "comm", err))
	} else {
		result.Command = command
	}

	executable, err := proc.Executable()
	if err != nil {
		errors = append(errors, processError(pid, "exe", err))
	} else {
		result.Executable = executable
	}

	result.Kind = classifyProcess(result.Command, result.Executable, result.CommandLine)
	return result, errors
}

func processError(pid int, operation string, err error) FindingError {
	return FindingError{
		PID:       pid,
		Operation: operation,
		Error:     err.Error(),
	}
}

func classifyProcess(command, executable string, cmdline []string) ProcessKind {
	candidates := make([]string, 0, 2+len(cmdline))
	candidates = append(candidates, command, executable)
	if len(cmdline) > 0 {
		candidates = append(candidates, cmdline[0])
	}

	readable := false
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		readable = true
		if looksLikeQEMU(candidate) {
			return ProcessKindQEMU
		}
	}

	if readable {
		return ProcessKindOther
	}

	return ProcessKindUnknown
}

func looksLikeQEMU(value string) bool {
	name := strings.ToLower(filepath.Base(value))
	return name == "qemu" ||
		name == "qemu-kvm" ||
		strings.HasPrefix(name, "qemu-system-")
}
