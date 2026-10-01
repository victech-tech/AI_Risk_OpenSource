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

const runKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`

type runSource struct {
	root  registry.Key
	path  string
	flags uint32
	label string
}

var runSources = []runSource{
	{registry.CURRENT_USER, runKey, 0, `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`},
	{registry.LOCAL_MACHINE, runKey, registry.WOW64_64KEY, `HKLM\Software\Microsoft\Windows\CurrentVersion\Run`},
	{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Run`, 0, `HKLM\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Run`},
}

// collectStartup lists programs set to start when Windows starts. Only the
// entry name and the program name are kept; arguments are dropped.
func collectStartup(env Env, out *report.Report) error {
	for _, src := range runSources {
		k, err := registry.OpenKey(src.root, src.path, registry.QUERY_VALUE|src.flags)
		if err != nil {
			continue
		}
		names, _ := k.ReadValueNames(-1)
		for _, n := range names {
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
	folders := []struct{ dir, label string }{
		{filepath.Join(env.AppData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup"), "Startup folder (this user)"},
		{filepath.Join(env.ProgramData, "Microsoft", "Windows", "Start Menu", "Programs", "StartUp"), "Startup folder (all users)"},
	}
	for _, f := range folders {
		out.StartupItems = append(out.StartupItems, startupFolderItems(f.dir, f.label)...)
	}
	return nil
}

// startupFolderItems lists file names in a Startup folder. Shortcut files
// are not opened, so the name stands in for the command.
func startupFolderItems(dir, label string) []report.StartupItem {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var items []report.StartupItem
	for _, e := range entries {
		if e.IsDir() || strings.EqualFold(e.Name(), "desktop.ini") {
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
