# Code signing policy

## Status

Releases are currently **unsigned**. Every release comes with:

- `checksums.txt`, so you can check the file you downloaded is exactly the one we published (see "Check your download" in the README);
- GitHub build attestations, which prove which source code and which GitHub workflow built each file.

Because the files are unsigned, Windows SmartScreen (a download safety check) may warn you the first time you run a new version. The README explains how to check the file before choosing **More info** > **Run anyway**. Please never switch SmartScreen off.

An application to the SignPath Foundation free signing programme was declined in October 2026, because the project is new and not yet widely known. We plan to apply again once it is, and we are considering other signing options. This page will say clearly when signing starts and who signs the files.

## Microsoft Store version

A Microsoft Store version is being prepared. Store packages are checked and signed by Microsoft when they are published, so they install without a SmartScreen warning. The package contains the same `.exe` as the direct download, built by the same workflow (see `packaging/msix/`).

## What gets signed (once signing starts)

Only `ai-exposure-scanner-windows-x64.exe` and `ai-exposure-scanner-windows-arm64.exe`, built from this repository by `.github/workflows/release.yml` on GitHub Actions from a tagged commit on `main`. Nothing built anywhere else will ever be signed.

## Who is involved

| Role | Who |
|---|---|
| Committers and reviewers | Members of the [victech-tech](https://github.com/victech-tech) account with write access to this repository |
| Release approver | Yan (project owner) |

All changes go through pull requests on a protected `main` branch.

## Privacy

This program does not send any information to any networked system. See [PRIVACY.md](PRIVACY.md).
