//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// downloadsDir returns the user's Downloads folder, even if it has been moved.
func downloadsDir() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Downloads, 0)
}

// openBrowser asks Windows to open the address in the default browser. The
// scanner itself makes no connection; only the address is passed on, with
// no report data in it.
func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

// showInFolder opens File Explorer with the report selected.
func showInFolder(path string) {
	cmd := exec.Command("explorer")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: fmt.Sprintf(`explorer /select,"%s"`, path)}
	_ = cmd.Start()
}
