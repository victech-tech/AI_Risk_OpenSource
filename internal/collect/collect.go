// Package collect reads facts from the computer. It never judges risk; the
// website does that. Each section is read by one collector, which runs with
// a time limit and records failures instead of stopping the scan.
//
// Only the locations listed in README.md ("What the scanner reads") may be
// read. Adding a new location needs a README update in the same change.
package collect

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/victech-tech/AI_Risk_OpenSource/internal/redact"
	"github.com/victech-tech/AI_Risk_OpenSource/internal/report"
)

// Timeout is the most time any one section may take.
const Timeout = 10 * time.Second

// ErrUnsupported is returned by collectors that only work on Windows.
var ErrUnsupported = errors.New("not supported on this operating system")

// Env holds the folders the collectors look in. Tests point these at
// testdata/ so most tests also run on Linux.
type Env struct {
	Home         string // %USERPROFILE%
	AppData      string // %APPDATA% (Roaming)
	LocalAppData string // %LOCALAPPDATA%
	ProgramData  string // %ProgramData%
}

// EnvFromOS reads the folders from the environment variables Windows sets.
func EnvFromOS() Env {
	home, _ := os.UserHomeDir()
	e := Env{
		Home:         home,
		AppData:      os.Getenv("APPDATA"),
		LocalAppData: os.Getenv("LOCALAPPDATA"),
		ProgramData:  os.Getenv("ProgramData"),
	}
	if e.AppData == "" && home != "" {
		e.AppData = filepath.Join(home, "AppData", "Roaming")
	}
	if e.LocalAppData == "" && home != "" {
		e.LocalAppData = filepath.Join(home, "AppData", "Local")
	}
	return e
}

// Section is one part of the report.
type Section struct {
	Name  string // report key, for example "installedApps"
	Label string // plain English, shown as progress
	Run   func(env Env, out *report.Report) error
}

// Sections lists every collector in the order they run.
func Sections() []Section {
	return []Section{
		{"installedApps", "Installed apps", collectApps},
		{"runningProcesses", "Programs running now", collectProcesses},
		{"startupItems", "Programs that start automatically", collectStartup},
		{"privacyPermissions", "Windows privacy permissions", collectPermissions},
		{"browserExtensions", "Browser and code editor extensions", CollectExtensions},
		{"aiToolConfigs", "AI tool settings", CollectAIConfigs},
	}
}

// RunSection runs one collector with the time limit and merges what it
// found into r. Failures go into r.Errors; the scan always carries on.
func RunSection(s Section, env Env, r *report.Report) {
	runWithTimeout(s, env, r, Timeout)
}

func runWithTimeout(s Section, env Env, r *report.Report, limit time.Duration) {
	part := &report.Report{}
	done := make(chan error, 1)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				done <- fmt.Errorf("unexpected problem: %v", p)
			}
		}()
		done <- s.Run(env, part)
	}()
	select {
	case err := <-done:
		merge(r, part)
		if err != nil {
			r.AddError(s.Name, errors.New(redact.Path(err.Error(), env.Home)))
		}
	case <-time.After(limit):
		// The collector's goroutine is abandoned; its partial results are
		// thrown away so they cannot change the report after this point.
		r.AddError(s.Name, fmt.Errorf("stopped after %s", limit))
	}
}

func merge(dst, src *report.Report) {
	dst.InstalledApps = append(dst.InstalledApps, src.InstalledApps...)
	dst.RunningProcesses = append(dst.RunningProcesses, src.RunningProcesses...)
	dst.StartupItems = append(dst.StartupItems, src.StartupItems...)
	dst.PrivacyPermissions = append(dst.PrivacyPermissions, src.PrivacyPermissions...)
	dst.BrowserExtensions = append(dst.BrowserExtensions, src.BrowserExtensions...)
	dst.AIToolConfigs = append(dst.AIToolConfigs, src.AIToolConfigs...)
	dst.Errors = append(dst.Errors, src.Errors...)
}

// maxFileSize stops the scanner reading unexpectedly large files.
const maxFileSize = 5 << 20

func readSmallFile(path string) ([]byte, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.Size() > maxFileSize {
		return nil, fmt.Errorf("%s is too large to read", filepath.Base(path))
	}
	return os.ReadFile(path)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
