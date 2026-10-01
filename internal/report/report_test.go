package report

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileName(t *testing.T) {
	ts := time.Date(2026, 10, 1, 12, 5, 0, 0, time.UTC)
	if got := FileName(ts); got != "ai-exposure-report-20261001-1205.json" {
		t.Errorf("got %q", got)
	}
}

func TestMarshalShape(t *testing.T) {
	r := New("0.1.0", time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC), OS{Family: "windows"})
	r.AddError("browserExtensions", errors.New("Firefox profile not found"))
	data, err := r.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["schemaVersion"].(float64) != 1 || m["scannedAt"] != "2026-10-01T12:00:00Z" {
		t.Errorf("unexpected header: %v", m)
	}
	if _, ok := m["installedApps"]; ok {
		t.Error("empty sections should be left out")
	}
	if len(m["errors"].([]any)) != 1 {
		t.Error("errors should be kept")
	}
}

func TestWriteDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "r.json")
	r := New("0.1.0", time.Now(), OS{Family: "windows"})
	if err := r.Write(p); err != nil {
		t.Fatal(err)
	}
	if err := r.Write(p); err == nil {
		t.Error("expected an error when the file exists")
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
}
