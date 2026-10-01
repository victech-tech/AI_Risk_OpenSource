package collect

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/victech-tech/AI_Risk_OpenSource/internal/redact"
	"github.com/victech-tech/AI_Risk_OpenSource/internal/report"
)

// Chromium browsers keep one folder per profile under "User Data". The
// scanner reads only Extensions\<id>\<version>\manifest.json and the
// _locales message files next to it. It never opens history, cookies,
// Preferences or any other profile file.
type chromiumBrowser struct {
	key  string // value written to the report
	path []string
}

var chromiumBrowsers = []chromiumBrowser{
	{"chrome", []string{"Google", "Chrome", "User Data"}},
	{"edge", []string{"Microsoft", "Edge", "User Data"}},
	{"brave", []string{"BraveSoftware", "Brave-Browser", "User Data"}},
}

// Code editors keep extensions in a folder under the user's home folder.
type editor struct {
	key  string
	path []string
}

var editors = []editor{
	{"vscode", []string{".vscode", "extensions"}},
	{"cursor", []string{".cursor", "extensions"}},
	{"windsurf", []string{".windsurf", "extensions"}},
}

// Permissions that give an extension access to every website. They are all
// written to the report as "<all_urls>" so the website has one value to check.
var allSitesPatterns = map[string]bool{
	"<all_urls>": true, "*://*/*": true, "http://*/*": true, "https://*/*": true,
}

// CollectExtensions lists browser and code editor extensions.
func CollectExtensions(env Env, out *report.Report) error {
	found := false
	for _, b := range chromiumBrowsers {
		dir := filepath.Join(append([]string{env.LocalAppData}, b.path...)...)
		if !exists(dir) {
			continue
		}
		found = true
		exts, errs := chromiumExtensions(dir, b.key)
		out.BrowserExtensions = append(out.BrowserExtensions, exts...)
		for _, err := range errs {
			out.AddError("browserExtensions", errors.New(redact.Path(err.Error(), env.Home)))
		}
	}
	ffDir := filepath.Join(env.AppData, "Mozilla", "Firefox", "Profiles")
	if exists(ffDir) {
		found = true
		exts, errs := firefoxExtensions(ffDir)
		out.BrowserExtensions = append(out.BrowserExtensions, exts...)
		for _, err := range errs {
			out.AddError("browserExtensions", errors.New(redact.Path(err.Error(), env.Home)))
		}
	}
	for _, e := range editors {
		dir := filepath.Join(append([]string{env.Home}, e.path...)...)
		if !exists(dir) {
			continue
		}
		found = true
		out.BrowserExtensions = append(out.BrowserExtensions, editorExtensions(dir, e.key)...)
	}
	if !found {
		out.AddError("browserExtensions", errors.New("no supported browser or code editor profiles found"))
	}
	return nil
}

func chromiumExtensions(userData, browser string) ([]report.BrowserExtension, []error) {
	var exts []report.BrowserExtension
	var errs []error
	profiles, err := os.ReadDir(userData)
	if err != nil {
		return nil, []error{err}
	}
	for _, p := range profiles {
		if !p.IsDir() {
			continue
		}
		extDir := filepath.Join(userData, p.Name(), "Extensions")
		ids, err := os.ReadDir(extDir)
		if err != nil {
			continue // not a profile folder
		}
		for _, id := range ids {
			if !id.IsDir() || id.Name() == "Temp" {
				continue
			}
			verDir, err := latestVersionDir(filepath.Join(extDir, id.Name()))
			if err != nil {
				continue
			}
			ext, err := parseChromiumManifest(verDir)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s extension %s: %v", browser, id.Name(), err))
				continue
			}
			ext.Browser = browser
			ext.Profile = p.Name()
			ext.ID = id.Name()
			exts = append(exts, ext)
		}
	}
	return exts, errs
}

func latestVersionDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var vers []string
	for _, e := range entries {
		if e.IsDir() && exists(filepath.Join(dir, e.Name(), "manifest.json")) {
			vers = append(vers, e.Name())
		}
	}
	if len(vers) == 0 {
		return "", errors.New("no manifest")
	}
	sort.Slice(vers, func(i, j int) bool { return versionLess(vers[i], vers[j]) })
	return filepath.Join(dir, vers[len(vers)-1]), nil
}

// versionLess compares folder names like "1.10.2_0" part by part.
func versionLess(a, b string) bool {
	split := func(s string) []string {
		return strings.FieldsFunc(s, func(r rune) bool { return r == '.' || r == '_' })
	}
	pa, pb := split(a), split(b)
	for i := 0; i < len(pa) && i < len(pb); i++ {
		if pa[i] == pb[i] {
			continue
		}
		if len(pa[i]) != len(pb[i]) {
			return len(pa[i]) < len(pb[i])
		}
		return pa[i] < pb[i]
	}
	return len(pa) < len(pb)
}

type chromiumManifest struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	DefaultLocale   string            `json:"default_locale"`
	Permissions     []json.RawMessage `json:"permissions"`
	HostPermissions []string          `json:"host_permissions"`
	ContentScripts  []struct {
		Matches []string `json:"matches"`
	} `json:"content_scripts"`
}

func parseChromiumManifest(dir string) (report.BrowserExtension, error) {
	data, err := readSmallFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return report.BrowserExtension{}, err
	}
	var m chromiumManifest
	if err := json.Unmarshal(stripBOM(data), &m); err != nil {
		return report.BrowserExtension{}, fmt.Errorf("manifest.json could not be read")
	}
	name := m.Name
	if strings.HasPrefix(name, "__MSG_") {
		name = localisedName(dir, m.DefaultLocale, strings.TrimSuffix(strings.TrimPrefix(name, "__MSG_"), "__"))
	}
	var perms []string
	for _, raw := range m.Permissions {
		var s string
		if json.Unmarshal(raw, &s) == nil { // older manifests can hold objects; skip those
			perms = append(perms, s)
		}
	}
	perms = append(perms, m.HostPermissions...)
	for _, cs := range m.ContentScripts {
		perms = append(perms, cs.Matches...)
	}
	return report.BrowserExtension{
		Name:        redact.Name(name),
		Version:     redact.Name(m.Version),
		Permissions: normalisePermissions(perms),
	}, nil
}

// localisedName resolves a "__MSG_key__" name from _locales/<lang>/messages.json.
func localisedName(dir, defaultLocale, key string) string {
	locales := []string{defaultLocale, "en_GB", "en", "en_US"}
	for _, loc := range locales {
		if loc == "" {
			continue
		}
		data, err := readSmallFile(filepath.Join(dir, "_locales", loc, "messages.json"))
		if err != nil {
			continue
		}
		var msgs map[string]struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(stripBOM(data), &msgs) != nil {
			continue
		}
		for k, v := range msgs {
			if strings.EqualFold(k, key) && v.Message != "" {
				return v.Message
			}
		}
	}
	return key
}

func normalisePermissions(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, p := range in {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if allSitesPatterns[p] {
			p = "<all_urls>"
		} else if strings.Contains(p, "://") {
			// Host permissions for single sites are kept without any query string.
			p = redact.StripQuery(p)
		}
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

type firefoxAddons struct {
	Addons []struct {
		ID            string `json:"id"`
		Version       string `json:"version"`
		Type          string `json:"type"`
		Active        bool   `json:"active"`
		Location      string `json:"location"`
		DefaultLocale struct {
			Name string `json:"name"`
		} `json:"defaultLocale"`
		UserPermissions *struct {
			Permissions []string `json:"permissions"`
			Origins     []string `json:"origins"`
		} `json:"userPermissions"`
	} `json:"addons"`
}

func firefoxExtensions(profilesDir string) ([]report.BrowserExtension, []error) {
	var exts []report.BrowserExtension
	var errs []error
	profiles, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, []error{err}
	}
	for _, p := range profiles {
		if !p.IsDir() {
			continue
		}
		path := filepath.Join(profilesDir, p.Name(), "extensions.json")
		data, err := readSmallFile(path)
		if err != nil {
			continue
		}
		var f firefoxAddons
		if err := json.Unmarshal(stripBOM(data), &f); err != nil {
			errs = append(errs, fmt.Errorf("firefox profile %s: extensions.json could not be read", firefoxProfileName(p.Name())))
			continue
		}
		for _, a := range f.Addons {
			// Skip Firefox's own built-in add-ons.
			if a.Type != "extension" || strings.HasPrefix(a.Location, "app-system") || a.Location == "app-builtin" {
				continue
			}
			var perms []string
			if a.UserPermissions != nil {
				perms = append(perms, a.UserPermissions.Permissions...)
				perms = append(perms, a.UserPermissions.Origins...)
			}
			enabled := a.Active
			exts = append(exts, report.BrowserExtension{
				Browser:     "firefox",
				Profile:     firefoxProfileName(p.Name()),
				ID:          redact.Name(a.ID),
				Name:        redact.Name(a.DefaultLocale.Name),
				Version:     redact.Name(a.Version),
				Permissions: normalisePermissions(perms),
				Enabled:     &enabled,
			})
		}
	}
	return exts, errs
}

// firefoxProfileName drops the random prefix: "x1y2z3.default-release" -> "default-release".
func firefoxProfileName(folder string) string {
	if i := strings.IndexByte(folder, '.'); i >= 0 && i+1 < len(folder) {
		return folder[i+1:]
	}
	return folder
}

type editorPackage struct {
	Name        string `json:"name"`
	Publisher   string `json:"publisher"`
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
}

// editorExtensions reads only package.json (and package.nls.json for the
// display name) in each extension folder.
func editorExtensions(dir, editorKey string) []report.BrowserExtension {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	latest := map[string]report.BrowserExtension{}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		extDir := filepath.Join(dir, e.Name())
		data, err := readSmallFile(filepath.Join(extDir, "package.json"))
		if err != nil {
			continue
		}
		var p editorPackage
		if json.Unmarshal(stripBOM(data), &p) != nil || p.Name == "" || p.Publisher == "" {
			continue
		}
		name := p.DisplayName
		if strings.HasPrefix(name, "%") && strings.HasSuffix(name, "%") {
			name = editorNLS(extDir, strings.Trim(name, "%"))
		}
		if name == "" {
			name = p.Name
		}
		id := strings.ToLower(p.Publisher + "." + p.Name)
		ext := report.BrowserExtension{
			Browser:     editorKey,
			ID:          redact.Name(id),
			Name:        redact.Name(name),
			Version:     redact.Name(p.Version),
			Permissions: []string{},
		}
		// Old versions can sit next to new ones; keep the newest.
		if prev, ok := latest[id]; !ok || versionLess(prev.Version, ext.Version) {
			latest[id] = ext
		}
	}
	ids := make([]string, 0, len(latest))
	for id := range latest {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]report.BrowserExtension, 0, len(ids))
	for _, id := range ids {
		out = append(out, latest[id])
	}
	return out
}

func editorNLS(dir, key string) string {
	data, err := readSmallFile(filepath.Join(dir, "package.nls.json"))
	if err != nil {
		return ""
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(stripBOM(data), &m) != nil {
		return ""
	}
	var s string
	if json.Unmarshal(m[key], &s) == nil {
		return s
	}
	return ""
}

func stripBOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return b[3:]
	}
	return b
}
