package collect

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/victech-tech/AI_Risk_OpenSource/internal/redact"
	"github.com/victech-tech/AI_Risk_OpenSource/internal/report"
)

// Base folders for config paths. The report shows the path with the
// placeholder (for example %APPDATA%\Claude\...), never the real folder.
const (
	baseHome    = "%USERPROFILE%"
	baseAppData = "%APPDATA%"
)

// configFormat says how to find MCP servers (tool connections) in a file.
type configFormat int

const (
	formatMCPServers     configFormat = iota // {"mcpServers": {...}}
	formatServers                            // {"servers": {...}} (VS Code mcp.json)
	formatVSCodeSettings                     // {"mcp": {"servers": {...}}} (VS Code settings.json)
	formatClaudeCode                         // mcpServers at the top and under each project
	formatCodexTOML                          // [mcp_servers.<name>] tables in TOML
)

// knownConfig is one AI tool config file the scanner looks for. This list
// is the complete set of AI config locations the scanner may read. Add new
// tools here and to the README table in the same change.
type knownConfig struct {
	Tool   string
	Base   string
	Path   []string
	Format configFormat
}

var knownConfigs = []knownConfig{
	{"claude-desktop", baseAppData, []string{"Claude", "claude_desktop_config.json"}, formatMCPServers},
	{"claude-code", baseHome, []string{".claude.json"}, formatClaudeCode},
	{"cursor", baseHome, []string{".cursor", "mcp.json"}, formatMCPServers},
	{"windsurf", baseHome, []string{".codeium", "windsurf", "mcp_config.json"}, formatMCPServers},
	{"vscode", baseAppData, []string{"Code", "User", "mcp.json"}, formatServers},
	{"vscode", baseAppData, []string{"Code", "User", "settings.json"}, formatVSCodeSettings},
	{"cline", baseAppData, []string{"Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"}, formatMCPServers},
	{"gemini-cli", baseHome, []string{".gemini", "settings.json"}, formatMCPServers},
	{"codex-cli", baseHome, []string{".codex", "config.toml"}, formatCodexTOML},
	{"lm-studio", baseHome, []string{".lmstudio", "mcp.json"}, formatMCPServers},
}

// CollectAIConfigs reads the known AI tool config files. Only server names
// and the program name of each server's command are kept. Arguments,
// environment values, headers, tokens and URLs are never copied.
func CollectAIConfigs(env Env, out *report.Report) error {
	for _, kc := range knownConfigs {
		base := env.Home
		if kc.Base == baseAppData {
			base = env.AppData
		}
		if base == "" {
			continue
		}
		real := filepath.Join(append([]string{base}, kc.Path...)...)
		if !exists(real) {
			continue
		}
		shown := kc.Base + `\` + strings.Join(kc.Path, `\`)
		data, err := readSmallFile(real)
		if err != nil {
			out.AddError("aiToolConfigs", fmt.Errorf("%s could not be opened", shown))
			continue
		}
		servers, err := parseMCPServers(data, kc.Format)
		if err != nil {
			out.AddError("aiToolConfigs", fmt.Errorf("%s could not be read", shown))
			continue
		}
		// settings.json is a general VS Code file; only report it when it
		// actually holds tool connections.
		if kc.Format == formatVSCodeSettings && len(servers) == 0 {
			continue
		}
		out.AIToolConfigs = append(out.AIToolConfigs, report.AIToolConfig{
			Tool:       kc.Tool,
			Path:       shown,
			MCPServers: servers,
		})
	}
	return nil
}

// mcpServer holds only the fields the scanner is allowed to look at.
// Everything else in the file (env, headers, args, tokens) is never decoded.
type mcpServer struct {
	Command   *string `json:"command"`
	URL       *string `json:"url"`
	ServerURL *string `json:"serverUrl"`
	HTTPURL   *string `json:"httpUrl"`
}

func parseMCPServers(data []byte, format configFormat) ([]report.MCPServer, error) {
	data = stripBOM(data)
	if format == formatCodexTOML {
		return parseCodexTOML(data), nil
	}
	clean := stripJSONC(data)
	var servers map[string]json.RawMessage
	switch format {
	case formatMCPServers:
		var f struct {
			MCPServers map[string]json.RawMessage `json:"mcpServers"`
		}
		if err := json.Unmarshal(clean, &f); err != nil {
			return nil, err
		}
		servers = f.MCPServers
	case formatServers:
		var f struct {
			Servers map[string]json.RawMessage `json:"servers"`
		}
		if err := json.Unmarshal(clean, &f); err != nil {
			return nil, err
		}
		servers = f.Servers
	case formatVSCodeSettings:
		var f struct {
			MCP struct {
				Servers map[string]json.RawMessage `json:"servers"`
			} `json:"mcp"`
		}
		if err := json.Unmarshal(clean, &f); err != nil {
			return nil, err
		}
		servers = f.MCP.Servers
	case formatClaudeCode:
		var f struct {
			MCPServers map[string]json.RawMessage `json:"mcpServers"`
			Projects   map[string]struct {
				MCPServers map[string]json.RawMessage `json:"mcpServers"`
			} `json:"projects"`
		}
		if err := json.Unmarshal(clean, &f); err != nil {
			return nil, err
		}
		servers = map[string]json.RawMessage{}
		for k, v := range f.MCPServers {
			servers[k] = v
		}
		// Project folder names are not copied; only the server names.
		for _, p := range f.Projects {
			for k, v := range p.MCPServers {
				if _, ok := servers[k]; !ok {
					servers[k] = v
				}
			}
		}
	}
	return toServers(servers), nil
}

func toServers(m map[string]json.RawMessage) []report.MCPServer {
	out := []report.MCPServer{}
	for name, raw := range m {
		var s mcpServer
		_ = json.Unmarshal(raw, &s) // a bad entry still counts as a server
		out = append(out, report.MCPServer{Name: redact.Name(name), Command: serverCommand(s)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func serverCommand(s mcpServer) string {
	if s.Command != nil && *s.Command != "" {
		return redact.ExecutableName(*s.Command)
	}
	for _, u := range []*string{s.URL, s.ServerURL, s.HTTPURL} {
		if u != nil && *u != "" {
			return "(url)"
		}
	}
	return ""
}

// parseCodexTOML reads only [mcp_servers.<name>] headers and their
// "command" keys. It is not a general TOML reader and does not need to be.
func parseCodexTOML(data []byte) []report.MCPServer {
	byName := map[string]*report.MCPServer{}
	var current *report.MCPServer
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			current = nil
			header := strings.Trim(line, "[] ")
			if !strings.HasPrefix(header, "mcp_servers.") {
				continue
			}
			name := strings.TrimPrefix(header, "mcp_servers.")
			// Sub-tables such as [mcp_servers.x.env] belong to server x.
			if strings.HasPrefix(name, `"`) {
				if end := strings.Index(name[1:], `"`); end >= 0 {
					name = name[1 : end+1]
				}
			} else if i := strings.IndexByte(name, '.'); i >= 0 {
				continue
			}
			name = redact.Name(name)
			if byName[name] == nil {
				byName[name] = &report.MCPServer{Name: name}
			}
			current = byName[name]
			continue
		}
		if current == nil {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "command":
			current.Command = redact.ExecutableName(strings.Trim(strings.TrimSpace(val), `"'`))
		case "url":
			if current.Command == "" {
				current.Command = "(url)"
			}
		}
	}
	out := []report.MCPServer{}
	for _, s := range byName {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
