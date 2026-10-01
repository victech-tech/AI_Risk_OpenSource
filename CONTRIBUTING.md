# Contributing

Thank you for helping. Most contributions are to the **rules**: the list of known AI tools, remote access tools and monitoring tools.

## Suggesting a tool (no coding needed)

Use the [Suggest a new AI tool](https://github.com/victech-tech/AI_Risk_OpenSource/issues/new?template=suggest-tool.yml) form. Never paste your whole report or anything personal.

## Adding or changing a rule

Rules are in `rules/`, one file per category:

| File | Category |
|---|---|
| `ai-assistants.json` | `ai-assistant` |
| `ai-agents.json` | `ai-agent` |
| `ai-coding-tools.json` | `ai-coding-tool` |
| `browser-extensions.json` | `ai-browser-extension` |
| `remote-access.json` | `remote-access` |
| `monitoring.json` | `monitoring` |

Any file may also hold `other` rules. Each rule looks like this:

```json
{
  "id": "example-ai-desktop",
  "name": "Example AI Desktop",
  "vendor": "Example Ltd",
  "category": "ai-agent",
  "match": {
    "installedAppName": ["^Example AI"],
    "processName": ["^exampleai\\.exe$"],
    "extensionId": [],
    "configTool": []
  },
  "capabilities": ["screen", "input-control", "files"],
  "baseRisk": "high",
  "summary": "An AI assistant that can see your screen and control your mouse and keyboard when you allow it.",
  "howToTurnOff": {
    "windows": "Open Example AI > Settings > Computer control and switch it off, or uninstall it from Settings > Apps."
  },
  "links": ["https://example.com/privacy"],
  "lastReviewed": "2026-10-01"
}
```

### Matching

- `installedAppName` and `processName` are regular expressions, matched without caring about upper or lower case. Keep them simple: use `^`, `$`, `\\.`, `( | )` and `?`. Avoid `(?i)`, look-aheads and `\p{...}`, because the website runs them in the browser.
- `installedAppName` is tested against the app names in Settings > Apps (and against startup entry names).
- `processName` is tested against program names such as `exampleai.exe` (running programs, startup programs and apps with privacy permissions).
- `extensionId` is an exact id: 32 letters for Chrome/Edge/Brave, `publisher.name` in lower case for VS Code/Cursor/Windsurf, or the add-on id for Firefox.
- `configTool` is an exact tool key from `internal/collect/aiconfigs.go`, for example `claude-desktop`, `cursor`, `gemini-cli`.
- A rule with no match fields is allowed. The website uses it in the self-check only.

### Allowed values

- `category`: `ai-assistant`, `ai-agent`, `ai-browser-extension`, `ai-coding-tool`, `remote-access`, `monitoring`, `other`
- `capabilities`: `screen`, `input-control`, `files`, `microphone`, `camera`, `browser-data`, `remote-access`, `runs-commands`, `background`
- `baseRisk`: `low`, `medium`, `high`
- `howToTurnOff` keys: `windows`, `mac`, `ios`, `android` (at least one)

### Writing style

- Plain UK English, short sentences, no jargon. Explain any technical term in brackets.
- Describe **what the product can do**, never accuse anyone. Avoid words like "malware", "spyware" or "malicious"; CI checks for some of these.
- `howToTurnOff` gives real menu paths a non-technical person can follow.
- For monitoring tools, remind people that removing them may alert whoever installed them, and point to the safety page.
- Set `lastReviewed` to the date you checked the rule.

### Checks

Run these before opening a pull request:

```
go test ./rules/
npx -y ajv-cli@5 validate --spec=draft2020 --strict=false -s rules/rules.schema.json -d rules/ai-agents.json
```

CI validates every file against the schema and checks for duplicate ids.

## Changing the scanner

- Read the README section "What the scanner reads". Any new location must be added there in the same pull request.
- Never add network code, telemetry, or dependencies beyond the standard library and `golang.org/x/sys`. Tests in `internal/policy/` enforce the first.
- Anything read from a config file must go through `internal/redact`. Add a fixture with fake secrets to `testdata/` and extend `TestNoSecretsInReport`.
- Changes to `schema/report.schema.json` change the contract with the website. Open an issue first.
