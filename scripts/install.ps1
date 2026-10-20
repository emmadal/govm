# govm - Go Version Manager Windows Installer
# PowerShell script for installing govm on Windows systems

# Enable strict mode
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Ensure TLS 1.2 or higher is used for HTTPS connections (required for GitHub API)
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12, [Net.SecurityProtocolType]::Tls13
} catch {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
}

Write-Host "Installing govm - Go Version Manager" -ForegroundColor Blue

## Detect architecture
$arch = [System.Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITECTURE")
# Fallback to PROCESSOR_ARCHITEW6432 if running 32-bit PowerShell on 64-bit Windows
if ($arch -eq "x86" -and [System.Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITEW6432") -ne $null) {
    $arch = [System.Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITEW6432")
}

$goArch = switch ($arch.ToUpper()) {
    "AMD64"   { "amd64" }
    "ARM64"   { "arm64" }
    "X86"     { "386" }
    "x86_64"  { "amd64" }
    default {
        Write-Host "Unsupported architecture: $arch" -ForegroundColor Red
        Write-Host "Please submit an issue at: https://github.com/emmadal/govm/issues"
        exit 1
    }
}

# Define installation directories
$govmDir = Join-Path $env:USERPROFILE ".govm"
$govmVersionsDir = Join-Path $govmDir "versions\go"
$govmCacheDir = Join-Path $govmDir ".cache"
$govmBinDir = Join-Path $env:USERPROFILE ".govm\bin"

# Create govm directories
Write-Host "Creating govm directories..." -ForegroundColor Blue
New-Item -ItemType Directory -Path $govmVersionsDir -Force | Out-Null
New-Item -ItemType Directory -Path $govmCacheDir -Force | Out-Null
New-Item -ItemType Directory -Path $govmBinDir -Force | Out-Null

# Create a temporary directory
$tempDir = Join-Path $env:TEMP "govm_install"
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
Set-Location $tempDir

# Download the pre-compiled binary for the detected platform
$asset = "govm_windows_$goArch.exe"
$baseUrl = "https://github.com/emmadal/govm/releases/latest/download"

Write-Host "Downloading $asset..." -ForegroundColor Blue
try {
    Invoke-WebRequest -Uri "$baseUrl/$asset" -OutFile "govm.exe" -UseBasicParsing
}
catch {
    Write-Host "Failed to download govm binary: $_" -ForegroundColor Red
    Write-Host "Please ensure you have a working internet connection and try again." -ForegroundColor Red
    Write-Host "If the problem persists, please submit an issue at: https://github.com/emmadal/govm/issues" -ForegroundColor Red
    exit 1
}

if (-not (Test-Path "govm.exe") -or (Get-Item "govm.exe").Length -eq 0) {
    Write-Host "Failed to download govm binary." -ForegroundColor Red
    exit 1
}

# Verify the checksum when the release publishes one
$checksums = $null
try {
    $checksums = (Invoke-WebRequest -Uri "$baseUrl/checksums.txt" -UseBasicParsing).Content
    if ($checksums -is [byte[]]) { $checksums = [Text.Encoding]::UTF8.GetString($checksums) }
}
catch {
    Write-Host "Warning: this release publishes no checksums; skipping verification." -ForegroundColor Yellow
}
if ($checksums) {
    $expected = $null
    foreach ($line in ($checksums -split "`n")) {
        $fields = $line.Trim() -split '\s+'
        if ($fields.Count -eq 2 -and $fields[1].TrimStart('*') -eq $asset) { $expected = $fields[0] }
    }
    if (-not $expected) {
        Write-Host "checksums.txt has no entry for $asset." -ForegroundColor Red
        exit 1
    }
    $actual = (Get-FileHash -Algorithm SHA256 "govm.exe").Hash
    if ($actual -ne $expected) {
        Write-Host "Checksum verification failed for $asset." -ForegroundColor Red
        exit 1
    }
    Write-Host "Checksum verified." -ForegroundColor Blue
}

# Install govm binary
Write-Host "Installing govm binary..." -ForegroundColor Blue
try {
    Copy-Item "govm.exe" -Destination $govmBinDir -Force
} catch {
    Write-Host "Failed to copy govm.exe to ${govmBinDir}: $_" -ForegroundColor Red
    exit 1
}

# Put govm and the active Go version (managed by `govm use`) on the user PATH
$currentGoBin = Join-Path $govmDir "current\bin"
$currentPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
if ($null -eq $currentPath) {
    $currentPath = ""
}
$pathEntries = @($currentPath -split ';' | Where-Object { $_ -ne "" })
$missing = @($govmBinDir, $currentGoBin | Where-Object { $pathEntries -notcontains $_ })
if ($missing.Count -gt 0) {
    Write-Host "Adding govm to your PATH..." -ForegroundColor Blue
    try {
        $newPath = (@($missing) + $pathEntries) -join ';'
        [System.Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = ($missing -join ';') + ";$env:Path"
        Write-Host "Successfully updated PATH environment variable." -ForegroundColor Green
    } catch {
        Write-Host "Warning: Failed to update PATH: $_" -ForegroundColor Yellow
        Write-Host "You may need to manually add $govmBinDir and $currentGoBin to your PATH." -ForegroundColor Yellow
    }
} else {
    Write-Host "govm is already in your PATH." -ForegroundColor Green
}

# Clean up temporary directory
Set-Location $env:USERPROFILE
Remove-Item -Recurse -Force $tempDir

Write-Host "🎉 govm has been successfully installed!" -ForegroundColor Green
Write-Host ""
Write-Host "Open a new terminal, then install Go with:"
Write-Host "    govm install latest" -ForegroundColor Blue
