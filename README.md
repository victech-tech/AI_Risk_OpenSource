# AI Exposure Scanner

A free, open-source Windows program that lists which AI tools, remote access tools and monitoring tools on your PC could see or control it. It makes a report file that you can read yourself, then drop onto the AI Exposure Check website, which explains the results in plain English **inside your browser**. Nothing is uploaded.

- **Read-only.** It never changes, deletes or installs anything.
- **No internet.** The scanner makes no network connections at all.
- **No admin rights needed.** It runs as a normal user.
- **No secrets.** It never copies passwords, API keys (secret codes that let programs use a service), tokens, documents, emails, browsing history or cookies.

> This check looks for known tools and settings. It cannot find software designed to hide. If you think someone is monitoring you, read the safety page on the website before you remove anything.

## How to use it

1. Download `ai-exposure-scanner-windows-x64.exe` from the [latest release](https://github.com/victech-tech/AI_Risk_OpenSource/releases/latest). If your PC has an ARM processor (for example some Surface and Copilot+ laptops), download `ai-exposure-scanner-windows-arm64.exe` instead.
2. [Check the download](#check-your-download) (recommended).
3. Double-click the file. A window opens and explains what the scanner will read. Nothing is read until you press **Y** then **Enter**.
4. When it finishes, the report is saved in your **Downloads** folder as `ai-exposure-report-YYYYMMDD-HHMM.json`. You can open it in Notepad to see exactly what it contains.
5. Your browser opens the report page. Drag the file onto the page to see your results.

### If Windows shows a warning

New programs from small publishers often show a blue "Windows protected your PC" message (from Microsoft SmartScreen, a download safety check). Choose **More info**, check the publisher, then **Run anyway** only if you have [checked the download](#check-your-download). Please never switch SmartScreen off.

### Check your download

Each release has a `checksums.txt` file. A checksum (a fingerprint of the file) proves the file has not been changed.

1. Open PowerShell (press the Windows key, type PowerShell, press Enter).
2. Run: `Get-FileHash $HOME\Downloads\ai-exposure-scanner-windows-x64.exe`
3. The long code it shows must match the line for that file in `checksums.txt`.

Releases also have **build attestations**, which prove which source code and which GitHub workflow built each file. If you have the GitHub command-line tool, run:

```
gh attestation verify ai-exposure-scanner-windows-x64.exe --repo victech-tech/AI_Risk_OpenSource
```

Signed releases are signed through SignPath Foundation. See [CODE_SIGNING.md](CODE_SIGNING.md).

## What the scanner reads

This is the complete list. The scanner reads nothing else. Any change to this list must update this table in the same pull request.

| Section | Where it reads | What it keeps |
|---|---|---|
| Installed apps | Registry `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall` (64-bit and `WOW6432Node`) and `HKCU\...\Uninstall` | Name, publisher, version |
| Programs running now | The Windows process list | Program name and file path (your user name replaced with `<user>`) |
| Programs that start automatically | `HKCU` and `HKLM` `...\CurrentVersion\Run` keys (including `WOW6432Node`); your Startup folder and the all-users Startup folder | Entry name and program name only (no arguments) |
| Privacy permissions | `HKCU\Software\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\` for `microphone`, `webcam`, `location`, `graphicsCaptureProgrammatic`, `graphicsCaptureWithoutBorder` | App name, allowed yes/no, last used time |
| Browser extensions | Chrome, Edge and Brave: `%LOCALAPPDATA%\<browser>\User Data\<profile>\Extensions\<id>\<version>\manifest.json` and its `_locales\*\messages.json`. Firefox: `%APPDATA%\Mozilla\Firefox\Profiles\<profile>\extensions.json` | Name, id, version, permissions, enabled (Firefox only) |
| Code editor extensions | `%USERPROFILE%\.vscode\extensions`, `.cursor\extensions`, `.windsurf\extensions`: each extension's `package.json` and `package.nls.json` | Name, id, version |
| AI tool settings | Claude Desktop `%APPDATA%\Claude\claude_desktop_config.json`; Claude Code `%USERPROFILE%\.claude.json`; Cursor `%USERPROFILE%\.cursor\mcp.json`; Windsurf `%USERPROFILE%\.codeium\windsurf\mcp_config.json`; VS Code `%APPDATA%\Code\User\mcp.json` and `settings.json`; Cline `%APPDATA%\Code\User\globalStorage\saoudrizwan.claude-dev\settings\cline_mcp_settings.json`; Gemini CLI `%USERPROFILE%\.gemini\settings.json`; Codex CLI `%USERPROFILE%\.codex\config.toml`; LM Studio `%USERPROFILE%\.lmstudio\mcp.json` | Only the names of tool connections (MCP servers) and the program name each one runs (for example `npx`), or `(url)` for an online one |

It also reads the Windows version and edition from `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`.

### What it never reads or keeps

- Browsing history, cookies, saved passwords, bookmarks, the browser's `Preferences` file or any other profile file.
- File contents, documents, emails, photos or messages.
- Command-line arguments, environment values (`env`), `headers`, tokens, API keys, or web addresses (URLs) from AI tool settings.
- Your Windows user name (replaced with `<user>` in every path).

Each section has a 10-second time limit. If a section fails, the scanner notes it in the report's `errors` list and carries on.

## Advanced options

```
ai-exposure-scanner.exe [--yes] [--out <file or folder>] [--no-browser] [--version]
```

| Option | What it does |
|---|---|
| `--yes` | Start without asking, and close without waiting for Enter |
| `--out` | Save the report somewhere else (a file name or a folder) |
| `--no-browser` | Do not open the report page or File Explorer at the end |
| `--version` | Show the version and exit |

---

## For developers

### What is in this repository

```
cmd/scanner/          entry point, consent prompt, writes the report
internal/collect/     one file per section; Windows code behind build tags
internal/report/      report structs and JSON writer
internal/redact/      secret-stripping and path redaction helpers
internal/policy/      tests that enforce "no network connections"
schema/               report.schema.json (the report format, version 1)
rules/                detection rules (*.json) and rules.schema.json
testdata/             fake profiles, extensions and config files with fake secrets
```

The scanner **collects facts only and never judges risk**. Risk rules live in `rules/` and are applied by the website, so rules can change daily without a new scanner release. Each new scanner release starts Windows SmartScreen trust from zero, so releases are rare.

### Build and test

Needs Go (see `go.mod`). Everything except the Windows collectors is tested on Linux too.

```
go test ./...                                  # all tests
GOOS=windows go vet ./...                      # check the Windows code builds
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o ai-exposure-scanner.exe ./cmd/scanner
```

Dependencies are kept to the standard library and `golang.org/x/sys` only. Every dependency is something users must trust.

### Rules

Rules are JSON files in `rules/`, one file per category. See [CONTRIBUTING.md](CONTRIBUTING.md) for how to add or change one. `rules/rules.schema.json` defines the format, and CI checks every file. When rules change on `main`, a `rules-YYYY.MM.DD` tag is created automatically and the website picks it up. **Rule changes never trigger a scanner release.**

### Releases

Pushing a `v*` tag runs `.github/workflows/release.yml`: GoReleaser builds the `.exe` files, writes `checksums.txt`, creates build attestations and (once set up) sends the files to SignPath for signing, which a maintainer approves.

## Licence

[Apache-2.0](LICENSE). Free code signing provided by SignPath.io, certificate by SignPath Foundation (see [CODE_SIGNING.md](CODE_SIGNING.md)).

See also: [PRIVACY.md](PRIVACY.md) · [SECURITY.md](SECURITY.md) · [CONTRIBUTING.md](CONTRIBUTING.md) · [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
