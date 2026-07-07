package nestedvirt

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/procfs"
)

const monitorSocketSourceProc = "proc_fd_net_unix"
const vmIdentitySourceMonitorSocket = "monitor_socket"

func (s *Scanner) discoverMonitorSockets(pid int) ([]MonitorSocket, []FindingError) {
	netUNIX, err := s.procFS.NetUNIX()
	if err != nil {
		return nil, []FindingError{processError(pid, "net_unix", err)}
	}

	fdDir := filepath.Join(s.procFSMount, strconv.Itoa(pid), "fd")
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return nil, []FindingError{processError(pid, "fd", err)}
	}

	byInode := unixSocketsByInode(netUNIX)
	var sockets []MonitorSocket

	for _, entry := range entries {
		fd, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		target, err := os.Readlink(filepath.Join(fdDir, entry.Name()))
		if err != nil {
			continue
		}

		inode, ok := socketInode(target)
		if !ok {
			continue
		}

		for _, line := range byInode[inode] {
			if !looksLikeMonitorSocket(line.Path) {
				continue
			}

			sockets = append(sockets, MonitorSocket{
				FD:     fd,
				Inode:  inode,
				Path:   line.Path,
				Source: monitorSocketSourceProc,
			})
		}
	}

	sort.Slice(sockets, func(i, j int) bool {
		if sockets[i].Path == sockets[j].Path {
			return sockets[i].FD < sockets[j].FD
		}
		return sockets[i].Path < sockets[j].Path
	})

	return deduplicateMonitorSockets(sockets), nil
}

func unixSocketsByInode(netUNIX *procfs.NetUNIX) map[uint64][]*procfs.NetUNIXLine {
	byInode := make(map[uint64][]*procfs.NetUNIXLine)
	if netUNIX == nil {
		return byInode
	}

	for _, line := range netUNIX.Rows {
		if line == nil || line.Inode == 0 || line.Path == "" {
			continue
		}
		byInode[line.Inode] = append(byInode[line.Inode], line)
	}

	return byInode
}

func socketInode(target string) (uint64, bool) {
	const (
		prefix = "socket:["
		suffix = "]"
	)

	if !strings.HasPrefix(target, prefix) || !strings.HasSuffix(target, suffix) {
		return 0, false
	}

	inode, err := strconv.ParseUint(strings.TrimSuffix(strings.TrimPrefix(target, prefix), suffix), 10, 64)
	if err != nil {
		return 0, false
	}

	return inode, true
}

func looksLikeMonitorSocket(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return name == "monitor.sock" ||
		name == "qmp.sock" ||
		strings.Contains(name, "monitor") ||
		strings.Contains(name, "qmp")
}

func deduplicateMonitorSockets(sockets []MonitorSocket) []MonitorSocket {
	type key struct {
		inode uint64
		path  string
	}

	seen := make(map[key]struct{}, len(sockets))
	deduped := sockets[:0]

	for _, socket := range sockets {
		k := key{inode: socket.Inode, path: socket.Path}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		deduped = append(deduped, socket)
	}

	return deduped
}

func libvirtDomainNameFromMonitorSockets(sockets []MonitorSocket) string {
	for _, socket := range sockets {
		if name := libvirtDomainNameFromMonitorSocket(socket.Path); name != "" {
			return name
		}
	}
	return ""
}

func libvirtDomainNameFromMonitorSocket(path string) string {
	const prefix = "domain-"

	dir := filepath.Base(filepath.Dir(path))
	if !strings.HasPrefix(dir, prefix) {
		return ""
	}

	_, name, ok := strings.Cut(strings.TrimPrefix(dir, prefix), "-")
	if !ok {
		return ""
	}

	return strings.TrimSpace(name)
}
