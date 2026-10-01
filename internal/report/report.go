// Package report defines the report file the scanner writes. The shape must
// match schema/report.schema.json, which the website uses to read it.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SchemaVersion is bumped only when the report format changes in a way
// the website must know about.
const SchemaVersion = 1

type Report struct {
	SchemaVersion      int                 `json:"schemaVersion"`
	Scanner            Scanner             `json:"scanner"`
	ScannedAt          string              `json:"scannedAt"`
	OS                 OS                  `json:"os"`
	InstalledApps      []InstalledApp      `json:"installedApps,omitempty"`
	RunningProcesses   []Process           `json:"runningProcesses,omitempty"`
	StartupItems       []StartupItem       `json:"startupItems,omitempty"`
	PrivacyPermissions []PrivacyPermission `json:"privacyPermissions,omitempty"`
	BrowserExtensions  []BrowserExtension  `json:"browserExtensions,omitempty"`
	AIToolConfigs      []AIToolConfig      `json:"aiToolConfigs,omitempty"`
	Errors             []Error             `json:"errors"`
}

type Scanner struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type OS struct {
	Family  string `json:"family"`
	Version string `json:"version,omitempty"`
	Edition string `json:"edition,omitempty"`
	Arch    string `json:"arch,omitempty"`
}

type InstalledApp struct {
	Name      string `json:"name"`
	Publisher string `json:"publisher,omitempty"`
	Version   string `json:"version,omitempty"`
	Scope     string `json:"scope"` // "user" or "machine"
}

type Process struct {
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
}

type StartupItem struct {
	Name    string `json:"name"`
	Source  string `json:"source"`
	Command string `json:"command,omitempty"` // executable name only, never arguments
}

type PrivacyPermission struct {
	Capability string `json:"capability"`
	App        string `json:"app"`
	Allowed    bool   `json:"allowed"`
	LastUsed   string `json:"lastUsed,omitempty"`
}

type BrowserExtension struct {
	Browser     string   `json:"browser"`
	Profile     string   `json:"profile,omitempty"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	Permissions []string `json:"permissions"`
	// Enabled is left out when the scanner cannot tell. For Chromium
	// browsers that state lives in profile files the scanner does not read.
	Enabled *bool `json:"enabled,omitempty"`
}

type AIToolConfig struct {
	Tool       string      `json:"tool"`
	Path       string      `json:"path"`
	MCPServers []MCPServer `json:"mcpServers"`
}

type MCPServer struct {
	Name    string `json:"name"`
	Command string `json:"command,omitempty"` // executable name only, or "(url)"
}

type Error struct {
	Section string `json:"section"`
	Message string `json:"message"`
}

// New starts an empty report.
func New(version string, now time.Time, os OS) *Report {
	return &Report{
		SchemaVersion: SchemaVersion,
		Scanner:       Scanner{Name: "ai-exposure-scanner", Version: version},
		ScannedAt:     now.UTC().Format(time.RFC3339),
		OS:            os,
		Errors:        []Error{},
	}
}

// AddError records a section failure. The scan always carries on.
func (r *Report) AddError(section string, err error) {
	r.Errors = append(r.Errors, Error{Section: section, Message: err.Error()})
}

// FileName gives the default report file name, for example
// ai-exposure-report-20261001-1200.json.
func FileName(now time.Time) string {
	return fmt.Sprintf("ai-exposure-report-%s.json", now.Format("20060102-1504"))
}

// Marshal returns the report as indented JSON so people can read it.
func (r *Report) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write saves the report. It refuses to overwrite an existing file.
func (r *Report) Write(path string) error {
	data, err := r.Marshal()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
