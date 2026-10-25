//go:build windows

package collect

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"

	"github.com/victech-tech/AI_Risk_OpenSource/internal/redact"
	"github.com/victech-tech/AI_Risk_OpenSource/internal/report"
)

const (
	runKey      = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
	approvedKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\`
)

type runSource struct {
	root     registry.Key
	path     string
	flags    uint32
	label    string
	approved string // StartupApproved subkey that records the Settings > Apps > Startup switch
}

var runSources = []runSource{
	{registry.CURRENT_USER, runKey, 0, `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "Run"},
	{registry.LOCAL_MACHINE, runKey, registry.WOW64_64KEY, `HKLM\Software\Microsoft\Windows\CurrentVersion\Run`, "Run"},
	{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Run`, 0, `HKLM\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Run`, "Run32"},
}

// switchedOff reads the StartupApproved subkey under root and returns the
// entry names that are switched off in Settings > Apps > Startup (or Task
// Manager). Switching an entry off there leaves it in the Run key, so
// without this check a switched-off program would still be listed.
func switchedOff(root registry.Key, sub string) map[string]bool {
	off := map[string]bool{}
	k, err := registry.OpenKey(root, approvedKey+sub, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return off
	}
	defer k.Close()
	names, _ := k.ReadValueNames(-1)
	for _, n := range names {
		b, _, err := k.GetBinaryValue(n)
		if err == nil && StartupSwitchedOff(b) {
			off[strings.ToLower(n)] = true
		}
	}
	return off
}

// collectStartup lists programs set to start when Windows starts. Only the
// entry name and the program name are kept; arguments are dropped. Entries
// switched off in Settings > Apps > Startup are left out, because they no
// longer start.
func collectStartup(env Env, out *report.Report) error {
	for _, src := range runSources {
		k, err := registry.OpenKey(src.root, src.path, registry.QUERY_VALUE|src.flags)
		if err != nil {
			continue
		}
		off := switchedOff(src.root, src.approved)
		names, _ := k.ReadValueNames(-1)
		for _, n := range names {
			if off[strings.ToLower(n)] {
				continue
			}
			cmd, _, err := k.GetStringValue(n)
			if err != nil {
				continue
			}
			out.StartupItems = append(out.StartupItems, report.StartupItem{
				Name:    redact.Name(n),
				Source:  src.label,
				Command: redact.ExecutableName(cmd),
			})
		}
		_ = k.Close()
	}
	folders := []struct {
		dir, label string
		root       registry.Key
	}{
		{filepath.Join(env.AppData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup"), "Startup folder (this user)", registry.CURRENT_USER},
		{filepath.Join(env.ProgramData, "Microsoft", "Windows", "Start Menu", "Programs", "StartUp"), "Startup folder (all users)", registry.LOCAL_MACHINE},
	}
	for _, f := range folders {
		out.StartupItems = append(out.StartupItems, startupFolderItems(f.dir, f.label, switchedOff(f.root, "StartupFolder"))...)
	}
	return nil
}

// startupFolderItems lists file names in a Startup folder. Shortcut files
// are not opened, so the name stands in for the command. Files named in off
// (lower case) are switched off in Settings and left out.
func startupFolderItems(dir, label string, off map[string]bool) []report.StartupItem {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var items []report.StartupItem
	for _, e := range entries {
		if e.IsDir() || strings.EqualFold(e.Name(), "desktop.ini") || off[strings.ToLower(e.Name())] {
			continue
		}
		name := e.Name()
		cmd := ""
		if strings.EqualFold(filepath.Ext(name), ".exe") {
			cmd = name
		}
		items = append(items, report.StartupItem{
			Name:    redact.Name(strings.TrimSuffix(name, filepath.Ext(name))),
			Source:  label,
			Command: cmd,
		})
	}
	return items
}
