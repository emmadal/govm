# govm - Go Version Manager Windows Installer
# PowerShell script for installing govm on Windows systems

# Enable strict mode
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "Installing govm - Go Version Manager" -ForegroundColor Blue

## Detect architecture
$arch = [System.Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITECTURE")
$goArch = switch ($arch) {
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
    Write-Host "Could not retrieve latest version. Using latest available." -ForegroundColor Yellow
    $latestVersion = "unknown"
}

# Download the pre-compiled binary for the detected platform
$downloadUrl = "https://github.com/emmadal/govm/releases/latest/download/govm_windows_$goArch.exe"

Write-Host "Downloading govm binary for windows_$goArch..." -ForegroundColor Blue
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile "govm.exe"
}
catch {
    Write-Host "Failed to download govm binary." -ForegroundColor Red
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
Copy-Item "govm.exe" -Destination $govmBinDir

# Create VERSION file with the latest version tag
$latestVersion | Out-File -FilePath (Join-Path $govmBinDir "VERSION") -Encoding utf8

# Add to PATH if not already there
$currentPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
if (-not $currentPath.Contains($govmBinDir)) {
    Write-Host "Adding govm to your PATH..." -ForegroundColor Blue
    $newPath = "$govmBinDir;$currentPath"
    [System.Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    $env:Path = "$govmBinDir;$env:Path"
}

# Clean up temporary directory
Set-Location $env:USERPROFILE
Remove-Item -Recurse -Force $tempDir

Write-Host "🎉 govm has been successfully installed!" -ForegroundColor Green
Write-Host ""
Write-Host "To start using govm, you need to close and reopen your PowerShell/Command Prompt, or run:"
Write-Host "    refreshenv" -ForegroundColor Blue
