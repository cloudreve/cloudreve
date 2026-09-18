<#
.SYNOPSIS
    Build MSIX packages and bundle for Cloudreve Desktop.

.DESCRIPTION
    This script cross-compiles the Tauri application for x64 and/or ARM64,
    creates MSIX packages using makeappx.exe from Windows SDK, and generates
    an MSIX bundle containing all architectures.

.PARAMETER Arch
    Target architecture: "x64", "arm64", or "all" (default: "all")

.PARAMETER Version
    Override the version from tauri.conf.json. Format: "X.Y.Z" (will be converted to "X.Y.Z.0")

.PARAMETER SkipBuild
    Skip the cargo build step (use existing binaries)

.PARAMETER OutputDir
    Output directory for MSIX files (default: "dist")

.EXAMPLE
    .\build-msix.ps1
    # Build MSIX packages for both x64 and ARM64, then create bundle

.EXAMPLE
    .\build-msix.ps1 -Arch x64
    # Build MSIX package for x64 only (no bundle)

.EXAMPLE
    .\build-msix.ps1 -Arch arm64 -Version "1.0.0"
    # Build ARM64 package with custom version (no bundle)
#>

param(
    [ValidateSet("x64", "arm64", "all")]
    [string]$Arch = "all",
    [string]$Version,
    [switch]$SkipBuild,
    [string]$OutputDir = "dist"
)

$ErrorActionPreference = "Stop"

# Paths
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$TauriConfigPath = Join-Path $ScriptDir "src-tauri\tauri.conf.json"
$ManifestPath = Join-Path $ScriptDir "package\AppxManifest.xml"
$PackageDir = Join-Path $ScriptDir "package"
$OutputPath = Join-Path $ScriptDir $OutputDir

# Architecture mapping
$ArchMap = @{
    "x64" = @{
        RustTarget = "x86_64-pc-windows-msvc"
        MsixArch = "x64"
    }
    "arm64" = @{
        RustTarget = "aarch64-pc-windows-msvc"
        MsixArch = "arm64"
    }
}

# Find makeappx.exe from Windows SDK
function Find-MakeAppx {
    $SdkPaths = @(
        "${env:ProgramFiles(x86)}\Windows Kits\10\bin",
        "$env:ProgramFiles\Windows Kits\10\bin"
    )

    foreach ($SdkPath in $SdkPaths) {
        if (Test-Path $SdkPath) {
            # Find all version directories and sort descending
            $Versions = Get-ChildItem $SdkPath -Directory |
                Where-Object { $_.Name -match '^\d+\.\d+\.\d+\.\d+$' } |
                Sort-Object { [Version]$_.Name } -Descending

            foreach ($Ver in $Versions) {
                $MakeAppx = Join-Path $Ver.FullName "x64\makeappx.exe"
                if (Test-Path $MakeAppx) {
                    return $MakeAppx
                }
            }
        }
    }

    return $null
}

# Get version from tauri.conf.json if not provided
if (-not $Version) {
    Write-Host "Reading version from tauri.conf.json..." -ForegroundColor Cyan
    $TauriConfig = Get-Content $TauriConfigPath -Raw | ConvertFrom-Json
    $Version = $TauriConfig.version
}

# Convert to 4-part version for MSIX
$MsixVersion = if ($Version -match '^\d+\.\d+\.\d+$') {
    "$Version.0"
} elseif ($Version -match '^\d+\.\d+\.\d+\.\d+$') {
    $Version
} else {
    Write-Error "Invalid version format: $Version. Expected X.Y.Z or X.Y.Z.W"
    exit 1
}

Write-Host "Version: $MsixVersion" -ForegroundColor Cyan

# Find makeappx.exe
$MakeAppx = Find-MakeAppx
if (-not $MakeAppx) {
    Write-Error "Could not find makeappx.exe. Please install Windows SDK."
    exit 1
}
Write-Host "Found makeappx.exe: $MakeAppx" -ForegroundColor Cyan

# Determine architectures to build
$TargetArchs = if ($Arch -eq "all") { @("x64", "arm64") } else { @($Arch) }

Write-Host "Target architectures: $($TargetArchs -join ', ')" -ForegroundColor Cyan

# Create output directory
if (-not (Test-Path $OutputPath)) {
    New-Item -ItemType Directory -Path $OutputPath -Force | Out-Null
}

# Create temp directory for package staging
$TempBaseDir = Join-Path $env:TEMP "cloudreve-msix-build"
if (Test-Path $TempBaseDir) {
    Remove-Item $TempBaseDir -Recurse -Force
}
New-Item -ItemType Directory -Path $TempBaseDir -Force | Out-Null

# Track created MSIX files for bundling
$CreatedMsixFiles = @()

# Build for each architecture
foreach ($TargetArch in $TargetArchs) {
    Write-Host "`n========================================" -ForegroundColor Green
    Write-Host "Building for $TargetArch" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green

    $Config = $ArchMap[$TargetArch]
    $RustTarget = $Config.RustTarget
    $MsixArch = $Config.MsixArch

    # Build the application
    if (-not $SkipBuild) {
        Write-Host "`nBuilding Tauri application for $RustTarget..." -ForegroundColor Cyan

        Push-Location $ScriptDir
        try {
            cargo tauri build --target $RustTarget
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Build failed for $TargetArch with exit code $LASTEXITCODE"
                exit $LASTEXITCODE
            }
        } finally {
            Pop-Location
        }
    } else {
        Write-Host "`nSkipping build step..." -ForegroundColor Yellow
    }

    # Determine binary path
    $BinaryPath = Join-Path $ScriptDir "target\$RustTarget\release\cloudreve-desktop.exe"
    if (-not (Test-Path $BinaryPath)) {
        Write-Error "Binary not found at: $BinaryPath"
        exit 1
    }

    # Create temp directory for this architecture
    $TempPackageDir = Join-Path $TempBaseDir $MsixArch
    Write-Host "Copying package to temp directory: $TempPackageDir" -ForegroundColor Cyan
    Copy-Item $PackageDir -Destination $TempPackageDir -Recurse -Force

    # Copy binary to temp package directory
    Write-Host "Copying binary..." -ForegroundColor Cyan
    Copy-Item $BinaryPath -Destination $TempPackageDir -Force

    # Update AppxManifest.xml in temp directory
    $TempManifestPath = Join-Path $TempPackageDir "AppxManifest.xml"
    Write-Host "Updating AppxManifest.xml for $MsixArch..." -ForegroundColor Cyan

    $ManifestContent = Get-Content $TempManifestPath -Raw -Encoding UTF8

    # Replace placeholders
    $ManifestContent = $ManifestContent -replace '__ARCH__', $MsixArch
    $ManifestContent = $ManifestContent -replace '__VERSION__', $MsixVersion

    # Write back with UTF-8 BOM
    $Utf8Bom = New-Object System.Text.UTF8Encoding $true
    [System.IO.File]::WriteAllText($TempManifestPath, $ManifestContent, $Utf8Bom)

    # Create MSIX package
    $MsixPath = Join-Path $OutputPath "Cloudreve.$MsixArch.msix"
    Write-Host "Creating MSIX package: $MsixPath" -ForegroundColor Cyan

    # Remove existing package if present
    if (Test-Path $MsixPath) {
        Remove-Item $MsixPath -Force
    }

    & $MakeAppx pack /v /p $MsixPath /d $TempPackageDir
    if ($LASTEXITCODE -ne 0) {
        Write-Error "makeappx.exe pack failed with exit code $LASTEXITCODE"
        exit $LASTEXITCODE
    }

    Write-Host "Created: $MsixPath" -ForegroundColor Green
    $CreatedMsixFiles += $MsixPath
}

# Create bundle if building for all architectures
if ($Arch -eq "all" -and $CreatedMsixFiles.Count -gt 1) {
    Write-Host "`n========================================" -ForegroundColor Green
    Write-Host "Creating MSIX Bundle" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green

    $BundlePath = Join-Path $OutputPath "Cloudreve.msixbundle"

    # Remove existing bundle if present
    if (Test-Path $BundlePath) {
        Remove-Item $BundlePath -Force
    }

    Write-Host "Creating bundle: $BundlePath" -ForegroundColor Cyan
    & $MakeAppx bundle /v /d $OutputPath /p $BundlePath /bv $MsixVersion
    if ($LASTEXITCODE -ne 0) {
        Write-Error "makeappx.exe bundle failed with exit code $LASTEXITCODE"
        exit $LASTEXITCODE
    }

    Write-Host "Created: $BundlePath" -ForegroundColor Green
}

# Cleanup temp directory
Write-Host "`nCleaning up temp directory..." -ForegroundColor Cyan
Remove-Item $TempBaseDir -Recurse -Force

Write-Host "`n========================================" -ForegroundColor Green
Write-Host "Build complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host "Output files:" -ForegroundColor Cyan
Get-ChildItem $OutputPath -Filter "*.msix*" | ForEach-Object {
    Write-Host "  $($_.FullName)" -ForegroundColor White
}
