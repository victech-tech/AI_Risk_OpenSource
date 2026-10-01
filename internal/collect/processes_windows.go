//go:build windows

package collect

import (
	"sort"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/victech-tech/AI_Risk_OpenSource/internal/redact"
	"github.com/victech-tech/AI_Risk_OpenSource/internal/report"
)

// collectProcesses lists the programs running now: name and file path only.
// It does not read command lines, window titles or memory.
func collectProcesses(env Env, out *report.Report) error {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(snap) }()

	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	seen := map[string]bool{}
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		name := windows.UTF16ToString(e.ExeFile[:])
		if name == "" || e.ProcessID == 0 {
			continue
		}
		path := redact.Path(processPath(e.ProcessID), env.Home)
		key := lowerTrim(name) + "|" + lowerTrim(path)
		if seen[key] {
			continue
		}
		seen[key] = true
		out.RunningProcesses = append(out.RunningProcesses, report.Process{Name: redact.Name(name), Path: path})
	}
	sort.Slice(out.RunningProcesses, func(i, j int) bool {
		return lowerTrim(out.RunningProcesses[i].Name) < lowerTrim(out.RunningProcesses[j].Name)
	})
	return nil
}

func lowerTrim(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// processPath returns the program file of a process, or "" if Windows does
// not allow a normal user to see it (common for system processes).
func processPath(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer func() { _ = windows.CloseHandle(h) }()
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(windows.MAX_LONG_PATH)
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}
