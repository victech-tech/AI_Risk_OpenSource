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
	for _, want := range []string{"Scan finished", "Your results page:", "which AI tools can see or control this PC", "how to switch each one off", "blocked from sending anything", reportURL} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("finish screen missing %q", want)
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatalf("expected the .json report and the .html results page, got %v", entries)
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "ai-exposure-report-") {
			t.Errorf("unexpected file %s", e.Name())
		}
	}
	page, err := os.ReadFile(filepath.Join(dir, strings.TrimSuffix(entries[0].Name(), ".html")+".html"))
	if err == nil && !strings.Contains(string(page), siteURL()+"viewer/viewer.js") {
		t.Error("results page does not load the viewer from the site")
	}
}

func TestSiteURL(t *testing.T) {
	old := reportURL
	defer func() { reportURL = old }()
	for in, want := range map[string]string{
		"https://x.test/report/": "https://x.test/",
		"https://x.test/report":  "https://x.test/",
		"https://x.test/":        "https://x.test/",
	} {
		reportURL = in
		if got := siteURL(); got != want {
			t.Errorf("siteURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFixReportURL(t *testing.T) {
	old := reportURL
	defer func() { reportURL = old }()
	for in, want := range map[string]string{
		"":                       defaultReportURL,
		"/":                      defaultReportURL,
		"http://x.test/report/":  defaultReportURL,
		"https://x.test/report/": "https://x.test/report/",
	} {
		reportURL = in
		fixReportURL()
		if reportURL != want {
			t.Errorf("fixReportURL(%q) = %q, want %q", in, reportURL, want)
		}
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
