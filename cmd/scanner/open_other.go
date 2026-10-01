//go:build !windows

package main

import (
	"os"
	"path/filepath"
)

// The scanner is built for Windows. These versions exist so it builds and
// its tests run on Linux CI; they never open anything.

func downloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Downloads"), nil
}

func openBrowser(string)  {}
func showInFolder(string) {}
