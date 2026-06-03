<#
.SYNOPSIS
    DevBoxOS Installer for Windows.
.DESCRIPTION
    Downloads and installs DevBoxOS CLI and Engine, adds to PATH.
    Usage: irm https://raw.githubusercontent.com/parv68/DevBoxOS/main/scripts/install.ps1 | iex
.PARAMETER Version
    Version to install (default: latest).
    Set via $env:DEVBOX_VERSION or pass as parameter.
#>

param(
    [string]$Version = $env:DEVBOX_VERSION
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

function Write-Info  { Write-Host "[INFO] $args" -ForegroundColor Green }
function Write-Warn  { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function Write-Error { Write-Host "[ERR]  $args" -ForegroundColor Red }

$RepoOwner = "parv68"
$RepoName = "DevBoxOS"
$Repo = "$RepoOwner/$RepoName"
$InstallDir = if ($env:DEVBOX_INSTALL_DIR) { $env:DEVBOX_INSTALL_DIR } else { "$env:LOCALAPPDATA\DevBoxOS" }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64"  { "amd64" }
    "ARM64"  { "arm64" }
    default  { "amd64" }
}

Write-Info "Detected: windows/$arch"

if (-not $Version) {
    Write-Info "Fetching latest version..."
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -ErrorAction Stop
        $Version = $release.tag_name
    } catch {
        Write-Error "Failed to fetch latest version. Check your internet connection."
        Write-Error "You can set a specific version: `$env:DEVBOX_VERSION = 'v1.0.0'"
        exit 1
    }
}

Write-Info "Installing DevBoxOS $Version"

$archiveName = "devbox_$($Version -replace '^v', '')_windows_$arch.zip"
$downloadUrl = "https://github.com/$Repo/releases/download/$Version/$archiveName"
$zipPath = "$env:TEMP\devbox_install.zip"

Write-Info "Downloading $archiveName ..."
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -ErrorAction Stop
} catch {
    Write-Error "Failed to download $downloadUrl"
    Write-Error "Check available releases at https://github.com/$Repo/releases"
    exit 1
}

if (-not (Test-Path $zipPath) -or (Get-Item $zipPath).Length -eq 0) {
    Write-Error "Downloaded file is empty or missing"
    exit 1
}

Write-Info "Installing to $InstallDir ..."
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

Write-Info "Extracting..."
try {
    Expand-Archive -Path $zipPath -DestinationPath $InstallDir -Force
} catch {
    Write-Error "Failed to extract archive: $_"
    exit 1
}

Remove-Item -Path $zipPath -Force -ErrorAction SilentlyContinue

$currentPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
if ($currentPath -notlike "*$InstallDir*") {
    Write-Info "Adding $InstallDir to PATH..."
    [Environment]::SetEnvironmentVariable("Path", "$InstallDir;$currentPath", [EnvironmentVariableTarget]::User)
    $env:Path = "$InstallDir;$env:Path"
} else {
    Write-Info "$InstallDir is already in PATH"
}

$cliPath = "$InstallDir\devbox.exe"
$enginePath = "$InstallDir\devbox-engine.exe"

if ((Test-Path $cliPath) -and (Test-Path $enginePath)) {
    Write-Info "DevBoxOS installed successfully!"
    try {
        $versionOutput = & $cliPath version 2>$null
        Write-Host ""
        Write-Host "  $versionOutput"
        Write-Host "  Location: $InstallDir"
        Write-Host ""
    } catch {
        Write-Warn "Could not verify version (close and reopen your terminal)"
    }
    Write-Host "  Get started:"
    Write-Host "    devbox init      # Initialize a new project"
    Write-Host "    devbox start     # Start your environment"
    Write-Host "    devbox doctor    # Run diagnostics"
    Write-Host ""
    Write-Warn "Close and reopen your terminal, or run this to update PATH now:"
    Write-Host '  $env:Path = "'"$InstallDir"';$env:Path"'
    Write-Host ""
} else {
    Write-Error "Installation failed - binaries not found in $InstallDir"
    exit 1
}