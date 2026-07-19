package serve

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ForceTakeover stops other tuile serve processes and clears the listen port.
func ForceTakeover(listenAddr string) error {
	if err := killTuileServeProcesses(os.Getpid()); err != nil {
		return err
	}
	port, err := portFromListen(listenAddr)
	if err != nil {
		return err
	}
	if err := killListenersOnPort(port); err != nil {
		return err
	}
	return waitListenAvailable(listenAddr, 3*time.Second)
}

func killTuileServeProcesses(exceptPID int) error {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil // best-effort on non-linux
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(ent.Name())
		if err != nil || pid == exceptPID || pid <= 1 {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", ent.Name(), "cmdline"))
		if err != nil {
			continue
		}
		parts := strings.Split(string(cmdline), "\x00")
		if len(parts) < 2 {
			continue
		}
		if !strings.HasSuffix(parts[0], "tuile") {
			continue
		}
		if parts[1] != "serve" {
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	time.Sleep(200 * time.Millisecond)
	for _, ent := range entries {
		pid, err := strconv.Atoi(ent.Name())
		if err != nil || pid == exceptPID || pid <= 1 {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", ent.Name(), "cmdline"))
		if err != nil {
			continue
		}
		if strings.Contains(string(cmdline), "tuile\x00serve") {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
	return nil
}

func killListenersOnPort(port int) error {
	pids, err := pidsListeningOnPort(port)
	if err != nil {
		return err
	}
	for _, pid := range pids {
		if pid == os.Getpid() {
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	if len(pids) > 0 {
		time.Sleep(200 * time.Millisecond)
	}
	for _, pid := range pids {
		if pid == os.Getpid() {
			continue
		}
		if processAlive(pid) == nil {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
	return nil
}

func pidsListeningOnPort(port int) ([]int, error) {
	if out, err := exec.Command("ss", "-ltnp", fmt.Sprintf("sport = :%d", port)).CombinedOutput(); err == nil {
		if pids := parsePIDsFromSS(string(out)); len(pids) > 0 {
			return uniquePIDs(pids), nil
		}
	}
	if out, err := exec.Command("fuser", fmt.Sprintf("%d/tcp", port)).CombinedOutput(); err == nil {
		return uniquePIDs(parsePIDsFromFuser(string(out))), nil
	}
	return nil, nil
}

func parsePIDsFromSS(out string) []int {
	var pids []int
	rest := out
	for {
		i := strings.Index(rest, "pid=")
		if i < 0 {
			break
		}
		rest = rest[i+4:]
		j := 0
		for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
			j++
		}
		if j > 0 {
			if pid, err := strconv.Atoi(rest[:j]); err == nil {
				pids = append(pids, pid)
			}
		}
	}
	return pids
}

func parsePIDsFromFuser(out string) []int {
	var pids []int
	for _, tok := range strings.Fields(out) {
		pid, err := strconv.Atoi(strings.TrimSpace(tok))
		if err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}

func uniquePIDs(in []int) []int {
	seen := make(map[int]struct{}, len(in))
	out := make([]int, 0, len(in))
	for _, pid := range in {
		if pid <= 1 {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		out = append(out, pid)
	}
	return out
}

func processAlive(pid int) error {
	return syscall.Kill(pid, 0)
}

func portFromListen(addr string) (int, error) {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, ":") {
			portStr = strings.TrimPrefix(addr, ":")
		} else {
			return 0, fmt.Errorf("listen address: %w", err)
		}
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("listen port: %w", err)
	}
	return port, nil
}

func waitListenAvailable(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			_ = ln.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("listen address %s still in use after force takeover", addr)
}
