#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Fix Visual Studio Build Tools C++ workload installation.
.DESCRIPTION
    Installs the C++ development workload for Visual Studio 2022 Build Tools.
    Use this if setup-windows.ps1's --quiet step failed silently.
    Run from the repo root:  .\fix-vs-buildtools.ps1
#>

Write-Host "Visual Studio Build Tools - C++ Workload Fix" -ForegroundColor Cyan
Write-Host "==============================================" -ForegroundColor Cyan
Write-Host ""

# ─── Step 1: Detect VS install path ───
Write-Host "[1/4] Detecting VS Build Tools installation..." -ForegroundColor Cyan

$vsPaths = @()

# Method 1: vswhere (standard)
$vswhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
if (Test-Path $vswhere) {
    Write-Host "  Found vswhere.exe - querying..." -ForegroundColor Yellow
    $vsPath = & $vswhere -latest -products "*" -requiresAny -requires "Microsoft.VisualStudio.Workload.VCTools" -property installationPath 2>$null
    if ($vsPath) { $vsPaths += $vsPath }
}

# Method 2: Check well-known BuildTools paths
$knownPaths = @(
    "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools",
    "C:\Program Files\Microsoft Visual Studio\2022\BuildTools"
)
foreach ($p in $knownPaths) {
    if (Test-Path "$p\Common7\Tools") { $vsPaths += $p }
}

if ($vsPaths.Length -eq 0) {
    Write-Host "  ERROR: Could not find VS Build Tools installation." -ForegroundColor Red
    Write-Host "  Install it first: winget install -h Microsoft.VisualStudio.2022.BuildTools" -ForegroundColor Yellow
    exit 1
}

$installPath = $vsPaths[0]
Write-Host "  Found at: $installPath" -ForegroundColor Green

# ─── Step 2: Check if VC tools are already installed ───
Write-Host "[2/4] Checking C++ workload status..." -ForegroundColor Cyan

$vcDir = "$installPath\VC"
if (Test-Path $vcDir) {
    Write-Host "  C++ workload already installed! (VC directory found)" -ForegroundColor Green
    Write-Host ""
    Write-Host "vcvarsall path:" -ForegroundColor Green
    $vcvars = Get-ChildItem -Path "$installPath\VC\Auxiliary\Build" -Filter "vcvarsall.bat" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($vcvars) {
        Write-Host "  $($vcvars.FullName)" -ForegroundColor White
    }
    exit 0
}

Write-Host "  C++ workload not found. Installing..." -ForegroundColor Yellow

# ─── Step 3: Install VC Tools workload ───
Write-Host "[3/4] Installing Microsoft.VisualStudio.Workload.VCTools..." -ForegroundColor Cyan

$vsInstaller = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vs_installer.exe"
if (-not (Test-Path $vsInstaller)) {
    Write-Host "  ERROR: vs_installer.exe not found at $vsInstaller" -ForegroundColor Red
    Write-Host "  Reinstall VS Build Tools first." -ForegroundColor Yellow
    exit 1
}

Write-Host "  This will open a progress window. Please wait for it to complete." -ForegroundColor Yellow
Write-Host "  It may take 10-30 minutes depending on your internet speed." -ForegroundColor Yellow
Write-Host ""

Start-Process -Wait -FilePath $vsInstaller -ArgumentList @(
    "modify",
    "--installPath", "`"$installPath`"",
    "--add", "Microsoft.VisualStudio.Workload.VCTools",
    "--includeRecommended",
    "--passive"
)

Write-Host "  Installer finished." -ForegroundColor Green

# ─── Step 4: Verify ───
Write-Host "[4/4] Verifying installation..." -ForegroundColor Cyan

if (Test-Path "$installPath\VC") {
    Write-Host "  SUCCESS: C++ workload installed!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Now run this to set up the build environment:" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  & `"$installPath\VC\Auxiliary\Build\vcvarsall.bat`" amd64" -ForegroundColor White
    Write-Host ""
    Write-Host "Then you can build C++ and Rust projects." -ForegroundColor Yellow
} else {
    Write-Host "  FAILED: VC directory still not found." -ForegroundColor Red
    Write-Host "  The installation may have failed. Check for errors in the VS Installer window." -ForegroundColor Yellow
    Write-Host "  You can also try installing the C++ workload manually:" -ForegroundColor Yellow
    Write-Host "    1. Search for 'Visual Studio Installer' in Start Menu" -ForegroundColor Gray
    Write-Host "    2. Find 'Visual Studio Build Tools 2022' → Modify" -ForegroundColor Gray
    Write-Host "    3. Select 'Desktop development with C++' → Install" -ForegroundColor Gray
    exit 1
}
