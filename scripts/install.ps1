<#
.SYNOPSIS
    DevBoxOS Installer for Windows.
.DESCRIPTION
    Downloads and installs DevBoxOS CLI and Engine, adds to PATH.
    Usage: iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/parv68/DevBoxOS/main/scripts/install.ps1'))
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
        throw "Failed to fetch latest version"
    }
}

Write-Info "Installing DevBoxOS $Version"

$versionStr = $Version -replace '^v', ''
$archiveNames = @(
    "devbox_${versionStr}_windows_${arch}.zip"
    "devboxos_${versionStr}_windows_${arch}.zip"
)

$zipPath = "$env:TEMP\devbox_install.zip"
$downloaded = $false

foreach ($name in $archiveNames) {
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/$name"
    Write-Info "Downloading $name ..."
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -ErrorAction Stop
        $downloaded = $true
        break
    } catch {
        Write-Warn "Failed to download $name, trying next..."
    }
}

if (-not $downloaded) {
    Write-Error "Failed to download from all archive names"
    Write-Error "Check available releases at https://github.com/$Repo/releases"
    throw "Download failed"
}

if (-not (Test-Path $zipPath) -or (Get-Item $zipPath).Length -eq 0) {
    Write-Error "Downloaded file is empty or missing"
    throw "Downloaded file is empty"
}

Write-Info "Installing to $InstallDir ..."
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

Write-Info "Extracting..."
$tmpExtract = "$env:TEMP\devbox_extract"
try {
    New-Item -ItemType Directory -Path $tmpExtract -Force | Out-Null
    Expand-Archive -Path $zipPath -DestinationPath $tmpExtract -Force

    Get-ChildItem -Path $tmpExtract | ForEach-Object {
        $dest = Join-Path $InstallDir $_.Name
        Move-Item -Path $_.FullName -Destination $dest -Force
    }
} catch {
    Write-Error "Failed to extract archive: $_"
    throw "Extraction failed"
} finally {
    if (Test-Path $tmpExtract) { Remove-Item -Path $tmpExtract -Recurse -Force -ErrorAction SilentlyContinue }
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

# Handle old binary name (pre-rename releases had devboxos.exe)
$cliPath = "$InstallDir\devbox.exe"
if (-not (Test-Path $cliPath)) {
    $oldCli = "$InstallDir\devboxos.exe"
    if (Test-Path $oldCli) {
        Rename-Item -Path $oldCli -NewName "devbox.exe" -Force
    }
}

$enginePath = "$InstallDir\devbox-engine.exe"
$oldEngine = "$InstallDir\devboxos-engine.exe"
if (-not (Test-Path $enginePath) -and (Test-Path $oldEngine)) {
    Rename-Item -Path $oldEngine -NewName "devbox-engine.exe" -Force
}

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
    Write-Host "  `$env:Path = `"$InstallDir;`$env:Path`""
    Write-Host ""
} else {
    Write-Error "Installation failed - binaries not found in $InstallDir"
    throw "Installation failed"
}