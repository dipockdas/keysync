#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Install all toolchains needed to run keysync client tests on Windows.
.DESCRIPTION
    Installs Go, Rust, Java, Python, Node, Ruby, CMake, and Visual Studio
    Build Tools via winget. Run this as Administrator.
#>

Write-Host "========================================================" -ForegroundColor Cyan
Write-Host "  keysync - Installing Windows Test Toolchains" -ForegroundColor Cyan
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host ""

# Ensure running as admin
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "WARNING: Not running as Administrator. Some installers may fail." -ForegroundColor Yellow
    Write-Host "Right-click PowerShell and select 'Run as Administrator'." -ForegroundColor Yellow
    Write-Host ""
}

function Install-Winget {
    param([string]$Name, [string]$Id)
    Write-Host "  Installing $Name..." -ForegroundColor Yellow
    winget install --accept-source-agreements --accept-package-agreements --exact -h "$Id" 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "    $Name installed." -ForegroundColor Green
    } else {
        Write-Host "    $Name may already be installed or failed." -ForegroundColor DarkGray
    }
}

# ─── 1. Go ───
Write-Host "[1/8] Go" -ForegroundColor Cyan
Install-Winget -Name "Go" -Id "GoLang.Go"

# ─── 2. Rust ───
Write-Host "[2/8] Rust" -ForegroundColor Cyan
Install-Winget -Name "Rustup" -Id "Rustlang.Rustup"

# ─── 3. Java (Temurin JDK 21) ───
Write-Host "[3/8] Java (JDK 21)" -ForegroundColor Cyan
Install-Winget -Name "Java Temurin JDK 21" -Id "EclipseAdoptium.Temurin.21.JDK"

# ─── 4. Python ───
Write-Host "[4/8] Python 3.14" -ForegroundColor Cyan
Install-Winget -Name "Python 3.14" -Id "Python.Python.3.14"

# ─── 5. Node.js ───
Write-Host "[5/8] Node.js" -ForegroundColor Cyan
Install-Winget -Name "Node.js" -Id "OpenJS.NodeJS.LTS"

# ─── 6. Ruby ───
Write-Host "[6/8] Ruby" -ForegroundColor Cyan
Install-Winget -Name "Ruby" -Id "RubyInstaller.Ruby.3.2"

# ─── 7. CMake ───
Write-Host "[7/8] CMake" -ForegroundColor Cyan
Install-Winget -Name "CMake" -Id "Kitware.CMake"

# ─── 8. Visual Studio Build Tools ───
Write-Host "[8/8] Visual Studio Build Tools (for Rust/C++)" -ForegroundColor Cyan
Install-Winget -Name "VS Build Tools 2022" -Id "Microsoft.VisualStudio.2022.BuildTools"

Write-Host ""
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host "  All packages installed via winget." -ForegroundColor Cyan
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host ""

# ─── Rustup init ───
Write-Host "Running rustup init..." -ForegroundColor Yellow
$rustup = Get-Command rustup -ErrorAction SilentlyContinue
if (-not $rustup) {
    $rustup = Get-Command "$env:USERPROFILE\.cargo\bin\rustup" -ErrorAction SilentlyContinue
}
if ($rustup) {
    & $rustup default stable 2>&1 | Out-Null
    Write-Host "  Rust is ready." -ForegroundColor Green
} else {
    Write-Host "  rustup not found on PATH. You may need to restart your shell." -ForegroundColor Yellow
}

# ─── VS Build Tools workload ───
Write-Host "Adding VS C++ workload..." -ForegroundColor Yellow
$vsWhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
if (Test-Path $vsWhere) {
    $vsPath = & $vsWhere -latest -property installationPath
    $vsInstaller = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vs_installer.exe"
    if ($vsPath -and (Test-Path $vsInstaller)) {
        $workload = "Microsoft.VisualStudio.Workload.NativeDesktop"
        Start-Process -Wait -FilePath $vsInstaller -ArgumentList "modify --installPath `"$vsPath`" --add $workload --quiet"
        Write-Host "  C++ workload installed." -ForegroundColor Green
    } else {
        Write-Host "  Could not find VS installer. Install 'Desktop development with C++' manually." -ForegroundColor Yellow
    }
} else {
    Write-Host "  vswhere not found. Install C++ workload manually via Visual Studio Installer." -ForegroundColor Yellow
}

# ─── Maven (manual, not on winget) ───
Write-Host ""
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host "  Final Steps" -ForegroundColor Cyan
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Maven: not on winget. To install manually:" -ForegroundColor Yellow
Write-Host "  1. Download from https://maven.apache.org/download.cgi" -ForegroundColor Gray
Write-Host "  2. Extract to C:\Program Files\Apache\maven" -ForegroundColor Gray
Write-Host "  3. Add to PATH (system environment variables)" -ForegroundColor Gray
Write-Host ""

Write-Host "Ruby: if winget install failed, download from https://rubyinstaller.org/" -ForegroundColor Yellow
Write-Host ""

Write-Host "After installing all tools, restart PowerShell, then:" -ForegroundColor Cyan
Write-Host "  cd keysync" -ForegroundColor White
Write-Host "  git pull" -ForegroundColor White
Write-Host '  powershell -ExecutionPolicy Bypass -File .\test-all.ps1' -ForegroundColor White
Write-Host ""
