<#
.SYNOPSIS
  Builds Microsoft Store packages (MSIX) for the scanner. Runs on Windows
  (GitHub Actions windows-latest) and needs makeappx.exe from the Windows SDK.

.DESCRIPTION
  Test packages (-Kind test) use a special publisher name that Windows 11
  accepts unsigned, for trying the package on your own PC before submitting.
  Store packages (-Kind store) use the identity from Partner Center and are
  combined into one .msixbundle for upload; Microsoft signs them.

.EXAMPLE
  ./build.ps1 -Kind test -Version 0.1.2 -ExeX64 x64.exe -ExeArm64 arm64.exe -OutDir out
#>
param(
  [Parameter(Mandatory)] [ValidateSet('test', 'store')] [string] $Kind,
  [Parameter(Mandatory)] [string] $Version,       # 0.1.2 or v0.1.2
  [Parameter(Mandatory)] [string] $ExeX64,
  [Parameter(Mandatory)] [string] $ExeArm64,
  [Parameter(Mandatory)] [string] $OutDir,
  [string] $IdentityName = '',                    # Partner Center: Package/Identity/Name
  [string] $Publisher = '',                       # Partner Center: Package/Identity/Publisher (CN=...)
  [string] $PublisherDisplayName = ''             # Partner Center: Package/Properties/PublisherDisplayName
)
$ErrorActionPreference = 'Stop'

# MSIX versions have four parts and the Store needs the last one to be 0.
$v = $Version.TrimStart('v')
if ($v -notmatch '^\d+\.\d+\.\d+$') { throw "Version must look like 0.1.2 (got '$Version')" }
$msixVersion = "$v.0"

if ($Kind -eq 'test') {
  $IdentityName = 'AIExposureCheck.Scanner.Test'
  # The OID marks the package as unsigned, so it can never pass for the real one.
  $Publisher = 'CN=AI Exposure Check Test, OID.2.25.311729368913984317654407730594956997722=1'
  $PublisherDisplayName = 'AI Exposure Check (test build)'
  $displayName = 'AI Exposure Scanner (test)'
} else {
  if (-not $IdentityName -or -not $Publisher -or -not $PublisherDisplayName) {
    throw 'Store packages need -IdentityName, -Publisher and -PublisherDisplayName from Partner Center'
  }
  $displayName = 'AI Exposure Scanner'
}

$sdk = Get-ChildItem "${env:ProgramFiles(x86)}\Windows Kits\10\bin\*\x64\makeappx.exe" -ErrorAction SilentlyContinue |
  Sort-Object FullName | Select-Object -Last 1
if (-not $sdk) { throw 'makeappx.exe not found (install the Windows SDK)' }
$makeappx = $sdk.FullName

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$OutDir = (Resolve-Path $OutDir).Path
$here = $PSScriptRoot
$template = Get-Content -Raw (Join-Path $here 'AppxManifest.template.xml')
$xmlEscape = { param($s) [System.Security.SecurityElement]::Escape($s) }

$packages = @()
foreach ($arch in @(@{ Name = 'x64'; Exe = $ExeX64 }, @{ Name = 'arm64'; Exe = $ExeArm64 })) {
  $stage = Join-Path ([System.IO.Path]::GetTempPath()) "msix-$Kind-$($arch.Name)"
  if (Test-Path $stage) { Remove-Item -Recurse -Force $stage }
  New-Item -ItemType Directory -Path $stage | Out-Null
  Copy-Item $arch.Exe (Join-Path $stage 'ai-exposure-scanner.exe')
  Copy-Item -Recurse (Join-Path $here 'Assets') (Join-Path $stage 'Assets')

  $manifest = $template
  $manifest = $manifest.Replace('{{IDENTITY_NAME}}', (& $xmlEscape $IdentityName))
  $manifest = $manifest.Replace('{{PUBLISHER}}', (& $xmlEscape $Publisher))
  $manifest = $manifest.Replace('{{PUBLISHER_DISPLAY_NAME}}', (& $xmlEscape $PublisherDisplayName))
  $manifest = $manifest.Replace('{{DISPLAY_NAME}}', (& $xmlEscape $displayName))
  $manifest = $manifest.Replace('{{VERSION}}', $msixVersion)
  $manifest = $manifest.Replace('{{ARCH}}', $arch.Name)
  if ($manifest -match '\{\{') { throw 'Unfilled placeholder in AppxManifest' }
  Set-Content -Path (Join-Path $stage 'AppxManifest.xml') -Value $manifest -Encoding UTF8

  $name = if ($Kind -eq 'test') { "ai-exposure-scanner-test-$($arch.Name).msix" } else { "ai-exposure-scanner-$v-$($arch.Name).msix" }
  $out = Join-Path $OutDir $name
  & $makeappx pack /o /d $stage /p $out | Write-Host
  if ($LASTEXITCODE -ne 0) { throw "makeappx pack failed for $($arch.Name)" }
  $packages += $out
}

if ($Kind -eq 'store') {
  # One bundle with both architectures is what Partner Center expects.
  $bundleDir = Join-Path ([System.IO.Path]::GetTempPath()) 'msix-bundle-input'
  if (Test-Path $bundleDir) { Remove-Item -Recurse -Force $bundleDir }
  New-Item -ItemType Directory -Path $bundleDir | Out-Null
  $packages | ForEach-Object { Copy-Item $_ $bundleDir }
  $bundle = Join-Path $OutDir "ai-exposure-scanner-$v.msixbundle"
  & $makeappx bundle /o /d $bundleDir /p $bundle /bv $msixVersion | Write-Host
  if ($LASTEXITCODE -ne 0) { throw 'makeappx bundle failed' }
  # The single-architecture files are inside the bundle; keep only the bundle.
  $packages | Remove-Item
}

Get-ChildItem $OutDir | Format-Table Name, Length
