//go:build windows

package collect

import (
	"strings"
	"testing"
	"time"

	"github.com/victech-tech/AI_Risk_OpenSource/internal/report"
)

// TestWindowsCollectorsSmoke runs every collector on the real machine (the
// Windows CI runner). It checks they finish in time, find something, and
// never leak the user's home folder.
func TestWindowsCollectorsSmoke(t *testing.T) {
	env := EnvFromOS()
	r := report.New("test", time.Now(), OSInfo())
	start := time.Now()
	for _, s := range Sections() {
		RunSection(s, env, r)
	}
	if d := time.Since(start); d > 30*time.Second {
		t.Errorf("scan took %s; it must finish in under 30 seconds", d)
	}
	if r.OS.Version == "" {
		t.Error("Windows version not read")
	}
	if len(r.InstalledApps) == 0 {
		t.Error("no installed apps found")
	}
	if len(r.RunningProcesses) == 0 {
		t.Error("no running processes found")
	}
	for _, e := range r.Errors {
		if strings.Contains(e.Message, "stopped after") {
			t.Errorf("section %s timed out", e.Section)
		}
	}
	data, err := r.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if env.Home != "" && strings.Contains(strings.ToLower(string(data)), strings.ToLower(strings.ReplaceAll(env.Home, `\`, `\\`))) {
		t.Error("report contains the home folder path")
	}
}

func TestFiletimeToRFC3339(t *testing.T) {
	// 2026-10-01T12:00:00Z as a Windows FILETIME.
	ft := uint64(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC).UnixNano()/100) + 116444736000000000
	if got := filetimeToRFC3339(ft); got != "2026-10-01T12:00:00Z" {
		t.Errorf("got %q", got)
	}
	if filetimeToRFC3339(1) != "" {
		t.Error("tiny values should give an empty string")
	}
}
