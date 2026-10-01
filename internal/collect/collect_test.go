package collect

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/victech-tech/AI_Risk_OpenSource/internal/report"
)

func fixtureEnv(t *testing.T) Env {
	t.Helper()
	home, err := filepath.Abs(filepath.Join("..", "..", "testdata", "home"))
	if err != nil {
		t.Fatal(err)
	}
	return Env{
		Home:         home,
		AppData:      filepath.Join(home, "AppData", "Roaming"),
		LocalAppData: filepath.Join(home, "AppData", "Local"),
		ProgramData:  filepath.Join(home, "ProgramData"),
	}
}

func findExt(exts []report.BrowserExtension, browser, id string) *report.BrowserExtension {
	for i := range exts {
		if exts[i].Browser == browser && exts[i].ID == id {
			return &exts[i]
		}
	}
	return nil
}

func TestCollectExtensions(t *testing.T) {
	r := &report.Report{}
	if err := CollectExtensions(fixtureEnv(t), r); err != nil {
		t.Fatal(err)
	}
	ex := findExt(r.BrowserExtensions, "chrome", "abcdefghijklmnopabcdefghijklmnop")
	if ex == nil {
		t.Fatalf("chrome extension not found: %+v", r.BrowserExtensions)
	}
	if ex.Name != "Example AI Helper" || ex.Version != "1.10.0" || ex.Profile != "Default" {
		t.Errorf("wrong details (want newest version and localised name): %+v", ex)
	}
	want := map[string]bool{"<all_urls>": true, "debugger": true, "tabs": true, "storage": true, "https://example.com/*": true}
	for _, p := range ex.Permissions {
		if !want[p] {
			t.Errorf("unexpected permission %q", p)
		}
		delete(want, p)
	}
	if len(want) != 0 {
		t.Errorf("missing permissions %v", want)
	}
	if ex.Enabled != nil {
		t.Error("chromium enabled state is not read, so it must be left out")
	}

	plain := findExt(r.BrowserExtensions, "chrome", "ponmlkjihgfedcbaponmlkjihgfedcba")
	if plain == nil || plain.Profile != "Profile 1" || plain.Name != "Plain Extension" {
		t.Errorf("second profile extension wrong: %+v", plain)
	}

	ff := findExt(r.BrowserExtensions, "firefox", "helper@example.com")
	if ff == nil || ff.Enabled == nil || !*ff.Enabled || ff.Profile != "default-release" {
		t.Fatalf("firefox extension wrong: %+v", ff)
	}
	off := findExt(r.BrowserExtensions, "firefox", "off@example.com")
	if off == nil || *off.Enabled {
		t.Errorf("disabled add-on should be listed as not enabled: %+v", off)
	}
	if findExt(r.BrowserExtensions, "firefox", "builtin@mozilla.org") != nil ||
		findExt(r.BrowserExtensions, "firefox", "default-theme@mozilla.org") != nil {
		t.Error("built-in add-ons and themes should be skipped")
	}

	cp := findExt(r.BrowserExtensions, "vscode", "github.copilot")
	if cp == nil || cp.Version != "1.250.0" || cp.Name != "GitHub Copilot" {
		t.Errorf("vscode copilot wrong: %+v", cp)
	}
	py := findExt(r.BrowserExtensions, "vscode", "ms-python.python")
	if py == nil || py.Name != "Python" {
		t.Errorf("vscode display name from package.nls.json not resolved: %+v", py)
	}
}

func TestCollectExtensionsNothingFound(t *testing.T) {
	r := &report.Report{}
	dir := t.TempDir()
	_ = CollectExtensions(Env{Home: dir, AppData: dir, LocalAppData: dir}, r)
	if len(r.Errors) != 1 {
		t.Errorf("expected one error, got %v", r.Errors)
	}
}

func servers(cfg report.AIToolConfig) map[string]string {
	m := map[string]string{}
	for _, s := range cfg.MCPServers {
		m[s.Name] = s.Command
	}
	return m
}

func TestCollectAIConfigs(t *testing.T) {
	r := &report.Report{}
	if err := CollectAIConfigs(fixtureEnv(t), r); err != nil {
		t.Fatal(err)
	}
	byTool := map[string]report.AIToolConfig{}
	for _, c := range r.AIToolConfigs {
		byTool[c.Tool+"|"+c.Path] = c
	}
	check := func(key string, want map[string]string) {
		t.Helper()
		c, ok := byTool[key]
		if !ok {
			t.Errorf("missing config %s (have %v)", key, keys(byTool))
			return
		}
		got := servers(c)
		if len(got) != len(want) {
			t.Errorf("%s: got %v, want %v", key, got, want)
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s: server %s command = %q, want %q", key, k, got[k], v)
			}
		}
	}
	check(`claude-desktop|%APPDATA%\Claude\claude_desktop_config.json`,
		map[string]string{"filesystem": "npx", "github": "node.exe", "remote-api": "(url)"})
	check(`claude-code|%USERPROFILE%\.claude.json`, map[string]string{"puppeteer": "npx", "postgres": "uvx"})
	check(`cursor|%USERPROFILE%\.cursor\mcp.json`, map[string]string{"browser-control": "npx", "context7": "(url)"})
	check(`vscode|%APPDATA%\Code\User\mcp.json`, map[string]string{"memory": "npx", "fetch": "(url)"})
	check(`codex-cli|%USERPROFILE%\.codex\config.toml`, map[string]string{"shell": "shell-mcp.exe", "docs-server": "(url)"})

	if _, ok := byTool[`vscode|%APPDATA%\Code\User\settings.json`]; ok {
		t.Error("VS Code settings.json with no servers should not be reported")
	}
	foundGeminiErr := false
	for _, e := range r.Errors {
		if e.Section == "aiToolConfigs" && strings.Contains(e.Message, ".gemini") {
			foundGeminiErr = true
		}
	}
	if !foundGeminiErr {
		t.Errorf("broken gemini settings should be recorded as an error: %v", r.Errors)
	}
}

func keys[M ~map[string]V, V any](m M) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestNoSecretsInReport is the acceptance check from the design: a report
// built from fixture configs full of fake API keys must contain none of them,
// nor the user's name, nor any profile file the scanner must not read.
func TestNoSecretsInReport(t *testing.T) {
	env := fixtureEnv(t)
	r := report.New("test", time.Now(), report.OS{Family: "windows"})
	RunSection(Section{"browserExtensions", "", CollectExtensions}, env, r)
	RunSection(Section{"aiToolConfigs", "", CollectAIConfigs}, env, r)
	data, err := r.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, bad := range []string{"FAKESECRET", "FAKEUSER", "sk-", "ghp_", "Bearer", "api_key", "token", "secret-project", "SECRET-HISTORY", "PREFS", "?", "args", "env", "headers", env.Home} {
		if strings.Contains(s, bad) {
			t.Errorf("report contains %q:\n%s", bad, s)
		}
	}
}

func TestRunSectionRecordsErrorsAndTimeouts(t *testing.T) {
	r := &report.Report{}
	env := Env{Home: `C:\Users\Sam`}
	runWithTimeout(Section{Name: "x", Run: func(Env, *report.Report) error {
		return errors.New(`cannot open C:\Users\Sam\file`)
	}}, env, r, time.Second)
	if len(r.Errors) != 1 || r.Errors[0].Message != `cannot open C:\Users\<user>\file` {
		t.Errorf("error not recorded or not redacted: %v", r.Errors)
	}

	r = &report.Report{}
	runWithTimeout(Section{Name: "slow", Run: func(_ Env, out *report.Report) error {
		time.Sleep(200 * time.Millisecond)
		out.InstalledApps = append(out.InstalledApps, report.InstalledApp{Name: "late"})
		return nil
	}}, env, r, 20*time.Millisecond)
	if len(r.Errors) != 1 || len(r.InstalledApps) != 0 {
		t.Errorf("timeout not handled: %+v", r)
	}

	r = &report.Report{}
	runWithTimeout(Section{Name: "panics", Run: func(Env, *report.Report) error { panic("boom") }}, env, r, time.Second)
	if len(r.Errors) != 1 {
		t.Errorf("panic not recorded: %v", r.Errors)
	}
}

func TestStripJSONC(t *testing.T) {
	in := `{"a": "x // not a comment", /* c */ "b": [1, 2,], // end
"c": "say \"hi\", ok",}`
	got := string(stripJSONC([]byte(in)))
	for _, want := range []string{`"x // not a comment"`, `[1, 2]`, `"say \"hi\", ok"}`} {
		if !strings.Contains(got, want) {
			t.Errorf("stripJSONC result %q missing %q", got, want)
		}
	}
}

func TestVersionLess(t *testing.T) {
	if !versionLess("1.2.0_0", "1.10.0_0") || versionLess("2.0", "1.9") || !versionLess("1.0", "1.0.1") {
		t.Error("versionLess wrong")
	}
}

func TestFirefoxProfileName(t *testing.T) {
	if firefoxProfileName("abc.default-release") != "default-release" || firefoxProfileName("plain") != "plain" {
		t.Error("firefoxProfileName wrong")
	}
}

func TestEnvFromOSFillsDefaults(t *testing.T) {
	t.Setenv("APPDATA", "")
	t.Setenv("LOCALAPPDATA", "")
	e := EnvFromOS()
	if home, _ := os.UserHomeDir(); home != "" && (e.AppData == "" || e.LocalAppData == "") {
		t.Errorf("defaults not filled: %+v", e)
	}
}

func TestSectionsCoverReportKeys(t *testing.T) {
	want := []string{"installedApps", "runningProcesses", "startupItems", "privacyPermissions", "browserExtensions", "aiToolConfigs"}
	got := Sections()
	if len(got) != len(want) {
		t.Fatalf("got %d sections", len(got))
	}
	for i, s := range got {
		if s.Name != want[i] {
			t.Errorf("section %d = %s, want %s", i, s.Name, want[i])
		}
	}
}
