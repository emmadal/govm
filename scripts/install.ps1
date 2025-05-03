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

# Get the latest version tag
Write-Host "Retrieving latest version information..." -ForegroundColor Blue
try {
    $latestVersion = (Invoke-RestMethod -Uri "https://api.github.com/repos/emmadal/govm/releases/latest").tag_name
}
catch {
    Write-Host "Could not retrieve latest version information: $_" -ForegroundColor Yellow
    Write-Host "Using latest available release." -ForegroundColor Yellow
    $latestVersion = "unknown"
}

# Download the pre-compiled binary for the detected platform
$downloadUrl = "https://github.com/emmadal/govm/releases/latest/download/govm_windows_$goArch.exe"

Write-Host "Downloading govm binary for windows_$goArch..." -ForegroundColor Blue
try {
    Write-Host "Downloading from $downloadUrl..." -ForegroundColor Blue
    Invoke-WebRequest -Uri $downloadUrl -OutFile "govm.exe" -UseBasicParsing
}
catch {
    Write-Host "Failed to download govm binary: $_" -ForegroundColor Red
    Write-Host "Please ensure you have a working internet connection and try again." -ForegroundColor Red
    Write-Host "If the problem persists, please submit an issue at: https://github.com/emmadal/govm/issues" -ForegroundColor Red
    exit 1
}

# Check if download was successful
if (-not (Test-Path "govm.exe") -or (Get-Item "govm.exe").Length -eq 0) {
    Write-Host "Failed to download govm binary." -ForegroundColor Red
    Write-Host "To build govm from source, you need Go installed on your machine." -ForegroundColor Red
    Write-Host "Please install Go and then run:" -ForegroundColor Blue
    Write-Host "git clone https://github.com/emmadal/govm.git"
    Write-Host "cd govm"
    Write-Host "go build --ldflags '-s -w' -o govm.exe"
    exit 1
}

# Install govm binary
Write-Host "Installing govm binary..." -ForegroundColor Blue
try {
    Copy-Item "govm.exe" -Destination $govmBinDir -Force
} catch {
    Write-Host "Failed to copy govm.exe to $govmBinDir: $_" -ForegroundColor Red
    Write-Host "You may need administrator privileges to write to this location." -ForegroundColor Yellow
    exit 1
}

# Create VERSION file with the latest version tag and installation timestamp
try {
    "Version: $latestVersion" | Out-File -FilePath (Join-Path $govmBinDir "VERSION") -Encoding utf8
    "Installed: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" | Add-Content -Path (Join-Path $govmBinDir "VERSION") -Encoding utf8
} catch {
    Write-Host "Warning: Could not write version information: $_" -ForegroundColor Yellow
}

# Check if we need to add govm to PATH
$currentPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
if ($null -eq $currentPath) {
    $currentPath = ""
}

# More precise PATH checking by splitting and comparing
$pathEntries = $currentPath -split ';' | Where-Object { $_ -ne "" }
if ($pathEntries -notcontains $govmBinDir) {
    Write-Host "Adding govm to your PATH..." -ForegroundColor Blue
    try {
        $newPath = "$govmBinDir;$currentPath".TrimEnd(';')
        [System.Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = "$govmBinDir;$env:Path"
        Write-Host "Successfully updated PATH environment variable." -ForegroundColor Green
    } catch {
        Write-Host "Warning: Failed to update PATH: $_" -ForegroundColor Yellow
        Write-Host "You may need to manually add $govmBinDir to your PATH." -ForegroundColor Yellow
    }
} else {
    Write-Host "govm bin directory is already in your PATH." -ForegroundColor Green
}

# Clean up temporary directory
Set-Location $env:USERPROFILE
Remove-Item -Recurse -Force $tempDir

Write-Host "🎉 govm has been successfully installed!" -ForegroundColor Green
Write-Host ""
Write-Host "To start using govm, you need to close and reopen your PowerShell/Command Prompt, or run:"
Write-Host "    refreshenv" -ForegroundColor Blue
