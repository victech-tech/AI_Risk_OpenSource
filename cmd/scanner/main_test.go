package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVersionFlag(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"--version"}, strings.NewReader(""), &out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "ai-exposure-scanner") {
		t.Errorf("got %q", out.String())
	}
}

func TestCancelReadsNothing(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	code := run([]string{"--out", dir + "/r.json", "--no-browser"}, strings.NewReader("\n\n"), &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "Nothing was read") {
		t.Errorf("expected cancel message, got %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "r.json")); err == nil {
		t.Error("report should not be written after cancelling")
	}
}

func TestIntroComesBeforePrompt(t *testing.T) {
	var out bytes.Buffer
	run([]string{"--out", t.TempDir(), "--no-browser"}, strings.NewReader("n\n\n"), &out)
	s := out.String()
	for _, want := range []string{"What it reads", "What it will never do", "connect to the internet", "Press Y then Enter"} {
		if !strings.Contains(s, want) {
			t.Errorf("intro missing %q", want)
		}
	}
	if strings.Index(s, "What it will never do") > strings.Index(s, "Press Y then Enter") {
		t.Error("the explanation must come before the prompt")
	}
}

func TestYesWritesReport(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if code := run([]string{"--yes", "--out", dir, "--no-browser"}, strings.NewReader(""), &out); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), "ai-exposure-report-") {
		t.Fatalf("expected one report file, got %v", entries)
	}
}

func TestResolveOutPath(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	got, err := resolveOutPath(dir, now)
	if err != nil || got != filepath.Join(dir, "ai-exposure-report-20261001-1200.json") {
		t.Errorf("folder: got %q, %v", got, err)
	}
	got, _ = resolveOutPath(filepath.Join(dir, "mine.json"), now)
	if filepath.Base(got) != "mine.json" {
		t.Errorf("file: got %q", got)
	}
}
