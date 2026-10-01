//go:build windows

package collect

import (
	"math"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"

	"github.com/victech-tech/AI_Risk_OpenSource/internal/redact"
	"github.com/victech-tech/AI_Risk_OpenSource/internal/report"
)

const consentStore = `Software\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore`

// Windows capability key -> name written to the report. Screen capture
// keys only exist on newer versions of Windows; missing keys are skipped.
var capabilities = []struct{ key, name string }{
	{"microphone", "microphone"},
	{"webcam", "camera"},
	{"location", "location"},
	{"graphicsCaptureProgrammatic", "screen-capture"},
	{"graphicsCaptureWithoutBorder", "screen-capture-without-border"},
}

// collectPermissions reads which apps Windows has allowed to use the
// microphone, camera, location and screen capture, from the current user's
// privacy settings.
func collectPermissions(env Env, out *report.Report) error {
	for _, c := range capabilities {
		k, err := registry.OpenKey(registry.CURRENT_USER, consentStore+`\`+c.key, registry.READ)
		if err != nil {
			continue
		}
		subs, _ := k.ReadSubKeyNames(-1)
		for _, s := range subs {
			if s == "NonPackaged" {
				out.PrivacyPermissions = append(out.PrivacyPermissions, nonPackaged(k, c.name, env)...)
				continue
			}
			if p, ok := readConsent(k, s, c.name, s); ok {
				out.PrivacyPermissions = append(out.PrivacyPermissions, p)
			}
		}
		_ = k.Close()
	}
	return nil
}

// nonPackaged handles ordinary desktop programs, whose key names are file
// paths with "#" instead of "\". Only the program file name is kept.
func nonPackaged(parent registry.Key, capName string, env Env) []report.PrivacyPermission {
	k, err := registry.OpenKey(parent, "NonPackaged", registry.READ)
	if err != nil {
		return nil
	}
	defer k.Close()
	subs, _ := k.ReadSubKeyNames(-1)
	var out []report.PrivacyPermission
	for _, s := range subs {
		app := redact.Base(strings.ReplaceAll(s, "#", `\`))
		if p, ok := readConsent(k, s, capName, app); ok {
			out = append(out, p)
		}
	}
	return out
}

func readConsent(parent registry.Key, sub, capName, app string) (report.PrivacyPermission, bool) {
	k, err := registry.OpenKey(parent, sub, registry.QUERY_VALUE)
	if err != nil {
		return report.PrivacyPermission{}, false
	}
	defer k.Close()
	v, _, err := k.GetStringValue("Value")
	if err != nil {
		return report.PrivacyPermission{}, false
	}
	p := report.PrivacyPermission{
		Capability: capName,
		App:        redact.Name(app),
		Allowed:    strings.EqualFold(v, "Allow"),
	}
	if ft, _, err := k.GetIntegerValue("LastUsedTimeStop"); err == nil && ft > 0 {
		p.LastUsed = filetimeToRFC3339(ft)
	}
	return p, true
}

// filetimeToRFC3339 converts a Windows FILETIME (100-nanosecond steps since
// 1601) to a UTC timestamp.
func filetimeToRFC3339(ft uint64) string {
	const epochDiff = 116444736000000000
	const maxSteps = math.MaxInt64 / 100
	if ft < epochDiff || ft-epochDiff > maxSteps {
		return ""
	}
	ns := int64(ft-epochDiff) * 100 //nolint:gosec // range checked above
	return time.Unix(0, ns).UTC().Format(time.RFC3339)
}
