package redact

import (
	"strings"
	"testing"
)

func TestPath(t *testing.T) {
	cases := []struct{ in, home, want string }{
		{`C:\Users\Sam\AppData\Local\X\x.exe`, `C:\Users\Sam`, `C:\Users\<user>\AppData\Local\X\x.exe`},
		{`c:\users\sam\x.exe`, `C:\Users\Sam`, `c:\users\<user>\x.exe`},
		{`C:\Users\Samantha\x.exe`, `C:\Users\Sam`, `C:\Users\Samantha\x.exe`},
		{`C:/Users/Sam/x.exe`, `C:\Users\Sam\`, `C:/Users/<user>/x.exe`},
		{`C:\Program Files\X\x.exe`, `C:\Users\Sam`, `C:\Program Files\X\x.exe`},
		{`/home/sam/.config/x`, `/home/sam`, `/home/<user>/.config/x`},
		{`C:\Users\Sam`, `C:\Users\Sam`, `C:\Users\<user>`},
		{``, `C:\Users\Sam`, ``},
	}
	for _, c := range cases {
		if got := Path(c.in, c.home); got != c.want {
			t.Errorf("Path(%q, %q) = %q, want %q", c.in, c.home, got, c.want)
		}
	}
}

func TestExecutableName(t *testing.T) {
	cases := []struct{ in, want string }{
		{`npx`, `npx`},
		{`npx -y @modelcontextprotocol/server-filesystem C:\Users\Sam`, `npx`},
		{`"C:\Program Files\Example AI\exampleai.exe" --background --token sk-123`, `exampleai.exe`},
		{`C:\Program Files\Example AI\exampleai.exe --api-key=sk-ant-secret`, `exampleai.exe`},
		{`C:\Users\Sam\bin\tool.exe`, `tool.exe`},
		{`/usr/local/bin/python3 server.py --key abc`, `python3`},
		{`https://mcp.example.com/sse?token=abc123`, `(url)`},
		{`'uvx' mcp-server-git`, `uvx`},
		{`API_KEY=sk-123`, `API_KEY`},
		{`   `, ``},
	}
	for _, c := range cases {
		if got := ExecutableName(c.in); got != c.want {
			t.Errorf("ExecutableName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExecutableNameNeverLeaksSecrets(t *testing.T) {
	secrets := []string{"sk-ant-api03-SECRET", "ghp_SECRETTOKEN", "Bearer SECRET", "token=SECRET"}
	for _, s := range secrets {
		for _, cmd := range []string{
			"node server.js " + s,
			`"C:\x\node.exe" --header "` + s + `"`,
			"https://example.com/mcp?" + s,
		} {
			if got := ExecutableName(cmd); strings.Contains(got, "SECRET") {
				t.Errorf("ExecutableName(%q) leaked a secret: %q", cmd, got)
			}
		}
	}
}

func TestStripQuery(t *testing.T) {
	if got := StripQuery("https://x.test/a?key=1#f"); got != "https://x.test/a" {
		t.Errorf("got %q", got)
	}
}

func TestName(t *testing.T) {
	if got := Name("  Example\x00 AI\n "); got != "Example AI" {
		t.Errorf("got %q", got)
	}
	if got := Name(strings.Repeat("a", 500)); len(got) != 200 {
		t.Errorf("len = %d", len(got))
	}
}

func TestBase(t *testing.T) {
	for in, want := range map[string]string{`C:\a\b.exe`: "b.exe", "/a/b": "b", `C:\a\`: "a", "b": "b"} {
		if got := Base(in); got != want {
			t.Errorf("Base(%q) = %q, want %q", in, got, want)
		}
	}
}
