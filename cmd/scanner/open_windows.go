//go:build windows

package main

import (
	"os/exec"

	"golang.org/x/sys/windows"
)

// downloadsDir returns the user's Downloads folder, even if it has been moved.
func downloadsDir() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Downloads, 0)
}

// openInBrowser asks Windows to open an address or a local results page
// in the user's default browser. The scanner itself makes no connection.
func openInBrowser(target string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
}
