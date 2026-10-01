# Code signing policy

Free code signing provided by [SignPath.io](https://about.signpath.io/), certificate by [SignPath Foundation](https://signpath.org/).

## Status

The application to SignPath Foundation has not been approved yet. Until it is, releases are **unsigned** and come with `checksums.txt` and GitHub build attestations so you can check them (see the README). This page will be updated when signing starts.

## What gets signed

Only `ai-exposure-scanner-windows-x64.exe` and `ai-exposure-scanner-windows-arm64.exe`, built from this repository by `.github/workflows/release.yml` on GitHub Actions. Nothing built anywhere else is ever signed.

## Who is involved

| Role | Who |
|---|---|
| Committers and reviewers | Members of the [victech-tech](https://github.com/victech-tech) organisation with write access to this repository |
| Approvers (approve each signing request in SignPath) | Yan (project owner) |

All changes go through pull requests on a protected `main` branch. Approvers check that each release was built from a tagged commit on `main` by the release workflow.

## Privacy

This program does not send any information to any networked system. See [PRIVACY.md](PRIVACY.md).
