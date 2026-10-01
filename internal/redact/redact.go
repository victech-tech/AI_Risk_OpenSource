// Package redact holds the helpers that stop secrets and personal paths
// reaching the report. Every value that comes from a config file, the
// registry or a command line must pass through here before it is written.
package redact

import (
	"path/filepath"
	"strings"
)

// UserPlaceholder replaces the user's home folder in every path.
const UserPlaceholder = "<user>"

// Path replaces the user's home folder with <user>. The comparison ignores
// case and accepts either slash style, because Windows paths do.
func Path(p, home string) string {
	if p == "" || home == "" {
		return p
	}
	home = strings.TrimRight(home, `\/`)
	if home == "" {
		return p
	}
	norm := func(s string) string { return strings.ToLower(strings.ReplaceAll(s, "/", `\`)) }
	np, nh := norm(p), norm(home)
	idx := strings.Index(np, nh)
	for idx >= 0 {
		end := idx + len(nh)
		// Only replace whole folder names: C:\Users\Sam must not match C:\Users\Samantha.
		if end == len(np) || np[end] == '\\' {
			p = p[:idx] + homePlaceholder(home, p[idx:end]) + p[end:]
			np = norm(p)
			idx = strings.Index(np, nh)
			continue
		}
		next := strings.Index(np[idx+1:], nh)
		if next < 0 {
			break
		}
		idx += next + 1
	}
	return p
}

// homePlaceholder keeps the parent folder (for example C:\Users\) and swaps
// only the last part, which is the user name.
func homePlaceholder(home, matched string) string {
	cut := strings.LastIndexAny(matched, `\/`)
	if cut < 0 {
		return UserPlaceholder
	}
	return matched[:cut+1] + UserPlaceholder
}

// ExecutableName returns only the program name from a command line, with
// no folder, arguments, flags or values. For example
//
//	"C:\Program Files\Example\example.exe" --token abc  ->  example.exe
//	npx -y @scope/server                              ->  npx
//
// Anything that looks like a URL is reduced to "(url)" so no host, path or
// query string can leak.
func ExecutableName(command string) string {
	c := strings.TrimSpace(command)
	if c == "" {
		return ""
	}
	var first string
	if c[0] == '"' || c[0] == '\'' {
		q := c[0]
		end := strings.IndexByte(c[1:], q)
		if end < 0 {
			first = c[1:]
		} else {
			first = c[1 : end+1]
		}
	} else {
		first = c
		// A command line with spaces but no quotes: Windows tries the longest
		// path ending in .exe first, so look for that before splitting.
		lower := strings.ToLower(c)
		if i := strings.Index(lower, ".exe"); i >= 0 && strings.ContainsAny(c[:i], `\/`) {
			first = c[:i+4]
		} else if i := strings.IndexAny(c, " \t"); i >= 0 {
			first = c[:i]
		}
	}
	if LooksLikeURL(first) {
		return "(url)"
	}
	// filepath.Base on Linux does not split on backslashes, so do it by hand.
	if i := strings.LastIndexAny(first, `\/`); i >= 0 {
		first = first[i+1:]
	}
	first = strings.TrimSpace(first)
	// Strip anything after "?" or "#", and any "key=value" style suffix.
	if i := strings.IndexAny(first, "?#="); i >= 0 {
		first = first[:i]
	}
	return first
}

// LooksLikeURL reports whether s is a web or other URL.
func LooksLikeURL(s string) bool {
	return strings.Contains(s, "://")
}

// StripQuery removes the query string and fragment from a URL or path.
func StripQuery(s string) string {
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		return s[:i]
	}
	return s
}

// Name tidies a display name: it trims spaces, removes control characters
// and limits the length, so odd registry values cannot break the report.
func Name(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	const max = 200
	if len([]rune(s)) > max {
		s = string([]rune(s)[:max])
	}
	return s
}

// Base returns the last element of a Windows or Unix path.
func Base(p string) string {
	p = strings.TrimRight(p, `\/`)
	if i := strings.LastIndexAny(p, `\/`); i >= 0 {
		return p[i+1:]
	}
	return filepath.Base(p)
}
