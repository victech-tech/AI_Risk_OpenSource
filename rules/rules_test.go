// Package rules holds the public detection rules (*.json). This test checks
// every rules file. CI also validates the files against rules.schema.json.
package rules

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

type rule struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Vendor   string `json:"vendor"`
	Category string `json:"category"`
	Match    struct {
		InstalledAppName []string `json:"installedAppName"`
		ProcessName      []string `json:"processName"`
		ExtensionID      []string `json:"extensionId"`
		ConfigTool       []string `json:"configTool"`
	} `json:"match"`
	Capabilities []string          `json:"capabilities"`
	BaseRisk     string            `json:"baseRisk"`
	Summary      string            `json:"summary"`
	HowToTurnOff map[string]string `json:"howToTurnOff"`
	Links        []string          `json:"links"`
	LastReviewed string            `json:"lastReviewed"`
}

var (
	categories   = set("ai-assistant", "ai-agent", "ai-browser-extension", "ai-coding-tool", "remote-access", "monitoring", "other")
	capabilities = set("screen", "input-control", "files", "microphone", "camera", "browser-data", "remote-access", "runs-commands", "background")
	risks        = set("low", "medium", "high")
	platforms    = set("windows", "mac", "ios", "android")
	idPattern    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// Chrome Web Store ids are 32 letters a-p; editor ids are publisher.name;
	// Firefox ids are an email-like string or a {GUID}.
	extIDPattern = regexp.MustCompile(`^([a-p]{32}|[a-z0-9-]+\.[a-z0-9-]+|[^\s]+@[^\s]+|\{[0-9a-fA-F-]{36}\})$`)
	// Regex features that work differently (or not at all) in Go and in the
	// browser's JavaScript, which is what the website uses.
	nonPortable = regexp.MustCompile(`\(\?[a-zA-Z]|\(\?<?[=!]|\\[pPzZAG]`)
	jargon      = []string{"exfiltrat", "malware", "spyware", "malicious", "threat actor", "you are safe"}
)

func set(xs ...string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func TestRules(t *testing.T) {
	files, err := filepath.Glob("*.json")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	total := 0
	for _, f := range files {
		if f == "rules.schema.json" {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		var rules []rule
		if err := dec.Decode(&rules); err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		for _, r := range rules {
			total++
			where := f + ": " + r.ID
			if prev, ok := seen[r.ID]; ok {
				t.Errorf("%s: duplicate id (also in %s)", where, prev)
			}
			seen[r.ID] = f
			checkRule(t, where, r)
		}
	}
	if total < 30 {
		t.Errorf("expected at least 30 rules, found %d", total)
	}
}

func checkRule(t *testing.T, where string, r rule) {
	t.Helper()
	if !idPattern.MatchString(r.ID) {
		t.Errorf("%s: id must be lower-case words joined by hyphens", where)
	}
	if r.Name == "" || r.Vendor == "" {
		t.Errorf("%s: name and vendor are required", where)
	}
	if !categories[r.Category] {
		t.Errorf("%s: unknown category %q", where, r.Category)
	}
	if !risks[r.BaseRisk] {
		t.Errorf("%s: unknown baseRisk %q", where, r.BaseRisk)
	}
	if len(r.Capabilities) == 0 {
		t.Errorf("%s: at least one capability is required", where)
	}
	caps := map[string]bool{}
	for _, c := range r.Capabilities {
		if !capabilities[c] {
			t.Errorf("%s: unknown capability %q", where, c)
		}
		if caps[c] {
			t.Errorf("%s: capability %q listed twice", where, c)
		}
		caps[c] = true
	}
	for _, list := range [][]string{r.Match.InstalledAppName, r.Match.ProcessName} {
		for _, p := range list {
			if _, err := regexp.Compile("(?i)" + p); err != nil {
				t.Errorf("%s: pattern %q is not a valid regular expression: %v", where, p, err)
			}
			if nonPortable.MatchString(p) {
				t.Errorf("%s: pattern %q uses a feature that does not work the same in the browser", where, p)
			}
		}
	}
	for _, id := range r.Match.ExtensionID {
		if !extIDPattern.MatchString(id) {
			t.Errorf("%s: extension id %q does not look right (editor ids must be lower case)", where, id)
		}
	}
	if len(r.Summary) < 20 || len(r.Summary) > 400 {
		t.Errorf("%s: summary should be 20 to 400 characters", where)
	}
	if len(r.HowToTurnOff) == 0 {
		t.Errorf("%s: howToTurnOff needs at least one platform", where)
	}
	for p, text := range r.HowToTurnOff {
		if !platforms[p] {
			t.Errorf("%s: unknown platform %q", where, p)
		}
		if len(text) < 10 || text == "..." {
			t.Errorf("%s: howToTurnOff.%s is too short", where, p)
		}
	}
	lower := strings.ToLower(r.Summary)
	for _, j := range jargon {
		if strings.Contains(lower, j) {
			t.Errorf("%s: summary uses %q; describe what the product can do, without accusations or jargon", where, j)
		}
	}
	for _, l := range r.Links {
		if !strings.HasPrefix(l, "https://") {
			t.Errorf("%s: link %q must start with https://", where, l)
		}
	}
	if _, err := time.Parse("2006-01-02", r.LastReviewed); err != nil {
		t.Errorf("%s: lastReviewed must be a date like 2026-10-01", where)
	}
}
