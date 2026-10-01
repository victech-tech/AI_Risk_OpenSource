//go:build windows

package collect

import (
	"fmt"

	"golang.org/x/sys/windows/registry"

	"github.com/victech-tech/AI_Risk_OpenSource/internal/redact"
	"github.com/victech-tech/AI_Risk_OpenSource/internal/report"
)

const uninstallKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`

type uninstallSource struct {
	root  registry.Key
	path  string
	flags uint32
	scope string
}

var uninstallSources = []uninstallSource{
	{registry.LOCAL_MACHINE, uninstallKey, registry.WOW64_64KEY, "machine"},
	{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`, 0, "machine"},
	{registry.CURRENT_USER, uninstallKey, 0, "user"},
}

// collectApps reads the name, publisher and version of each installed app
// from the registry "Uninstall" keys. Nothing else is read.
func collectApps(env Env, out *report.Report) error {
	seen := map[string]bool{}
	opened := 0
	for _, src := range uninstallSources {
		k, err := registry.OpenKey(src.root, src.path, registry.READ|src.flags)
		if err != nil {
			continue
		}
		opened++
		names, _ := k.ReadSubKeyNames(-1)
		for _, n := range names {
			app, ok := readUninstallEntry(k, n, src.scope)
			if !ok {
				continue
			}
			key := app.Name + "|" + app.Version + "|" + app.Scope
			if seen[key] {
				continue
			}
			seen[key] = true
			out.InstalledApps = append(out.InstalledApps, app)
		}
		_ = k.Close()
	}
	if opened == 0 {
		return fmt.Errorf("could not open the installed apps list")
	}
	return nil
}

func readUninstallEntry(parent registry.Key, name, scope string) (report.InstalledApp, bool) {
	k, err := registry.OpenKey(parent, name, registry.QUERY_VALUE)
	if err != nil {
		return report.InstalledApp{}, false
	}
	defer k.Close()
	display, _, err := k.GetStringValue("DisplayName")
	if err != nil || display == "" {
		return report.InstalledApp{}, false
	}
	// Hidden system parts and updates are not apps people recognise.
	if v, _, err := k.GetIntegerValue("SystemComponent"); err == nil && v == 1 {
		return report.InstalledApp{}, false
	}
	if p, _, err := k.GetStringValue("ParentKeyName"); err == nil && p != "" {
		return report.InstalledApp{}, false
	}
	publisher, _, _ := k.GetStringValue("Publisher")
	version, _, _ := k.GetStringValue("DisplayVersion")
	return report.InstalledApp{
		Name:      redact.Name(display),
		Publisher: redact.Name(publisher),
		Version:   redact.Name(version),
		Scope:     scope,
	}, true
}

// OSInfo describes the version of Windows.
func OSInfo() report.OS {
	info := osInfoGeneric()
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return info
	}
	defer k.Close()
	major, _, errMaj := k.GetIntegerValue("CurrentMajorVersionNumber")
	minor, _, _ := k.GetIntegerValue("CurrentMinorVersionNumber")
	build, _, _ := k.GetStringValue("CurrentBuild")
	if errMaj == nil {
		info.Version = fmt.Sprintf("%d.%d.%s", major, minor, build)
	} else if build != "" {
		info.Version = build
	}
	edition, _, _ := k.GetStringValue("EditionID")
	info.Edition = redact.Name(edition)
	return info
}
