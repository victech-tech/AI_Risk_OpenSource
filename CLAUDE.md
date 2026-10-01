# CLAUDE.md — AI Exposure Scanner (public, open source)

This is **Repo B** of AI Exposure Check. The full design lives in the private repo (`victech-tech/AI_Risk_Private`, `docs/DESIGN.md`); this file summarises the parts that apply here. The private repo's design doc is the source of truth.

## What this repo is

- A read-only Windows scanner (Go, single static `.exe`, no installer, no admin rights) that writes a JSON report of facts.
- The public detection rules dataset (`rules/*.json`) and its schema.
- The report format (`schema/report.schema.json`).
- The website (private repo) copies `rules/` and `schema/` at build time and does all matching and risk scoring in the user's browser.

## Hard requirements (ask before changing)

- **Data contracts:** `schema/report.schema.json` and `rules/rules.schema.json`. Keep them versioned; changes affect the website.
- **No network connections at all.** No telemetry. `internal/policy` tests enforce this. `golang.org/x/sys/windows` importing `net` is the only documented exception.
- **No secrets in the report.** Only MCP server names and the command's program name; never args, env, headers, tokens or URLs. Home folder replaced with `<user>`. Everything goes through `internal/redact`.
- Read only the locations in the README table; update the table in the same change as any new location.
- Never change, delete or install anything on the user's device.
- Dependencies: standard library and `golang.org/x/sys` only.
- **No proprietary code in this repo** (SignPath Foundation condition).
- The scanner collects facts only; it never judges risk.
- Rule changes never trigger a scanner release. Release rarely.

## Layout

```
cmd/scanner/        main.go (consent flow, flags), open_windows.go
internal/collect/   collectors; *_windows.go behind build tags; extensions.go and aiconfigs.go are cross-platform
internal/report/    report structs, writer
internal/redact/    redaction helpers
internal/policy/    no-network tests
rules/              rules JSON + rules_test.go
schema/             report.schema.json
testdata/           fixtures (fake secrets live here)
```

## Commands

```
go test ./...
GOOS=windows go vet ./...
go test ./rules/            # rules checks
```

## Style

- Plain code, few abstractions; the maintainer is a solo, part-time founder.
- All user-facing text: plain UK English, no jargon, explain technical terms in brackets.
