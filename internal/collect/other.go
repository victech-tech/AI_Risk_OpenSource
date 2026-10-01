//go:build !windows

package collect

import "github.com/victech-tech/AI_Risk_OpenSource/internal/report"

// On other systems only the shared, file-based collectors work. These
// stubs let the program build and its tests run on Linux CI.

func collectApps(Env, *report.Report) error        { return ErrUnsupported }
func collectProcesses(Env, *report.Report) error   { return ErrUnsupported }
func collectStartup(Env, *report.Report) error     { return ErrUnsupported }
func collectPermissions(Env, *report.Report) error { return ErrUnsupported }

// OSInfo describes the operating system.
func OSInfo() report.OS { return osInfoGeneric() }
