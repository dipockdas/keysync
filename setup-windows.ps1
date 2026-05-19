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
Write-Host "[6/9] Ruby" -ForegroundColor Cyan
Install-Winget -Name "Ruby" -Id "RubyInstallerTeam.Ruby.4.0"

# ─── 7. Maven ───
Write-Host "[7/9] Maven" -ForegroundColor Cyan
Install-Winget -Name "Maven" -Id "Apache.Maven"

# ─── 8. CMake ───
Write-Host "[8/9] CMake" -ForegroundColor Cyan
Install-Winget -Name "CMake" -Id "Kitware.CMake"

# ─── 8. Visual Studio Build Tools ───
Write-Host "[9/9] Visual Studio Build Tools (for Rust/C++)" -ForegroundColor Cyan
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

# ─── Fix PATH for the current session ───
Write-Host ""
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host "  Setting up PATH for this session" -ForegroundColor Cyan
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host ""

# Rust/Cargo
$cargoPaths = @(
    "$env:USERPROFILE\.cargo\bin",
    "$env:LOCALAPPDATA\.cargo\bin"
)
foreach ($p in $cargoPaths) {
    if (Test-Path "$p\cargo.exe" -and $env:Path -notlike "*$p*") {
        $env:Path += ";$p"
        Write-Host "  Added Cargo to PATH: $p" -ForegroundColor Green
        break
    }
}

# Maven — search common locations
$mavenDirs = @(
    "$env:USERPROFILE\Downloads\apache-maven-*\bin",
    "C:\Program Files\Apache\maven\bin",
    "C:\Program Files\Maven\apache-maven-*\bin",
    "${env:ProgramFiles(x86)}\Maven\apache-maven-*\bin",
    "$env:USERPROFILE\apache-maven-*\bin"
)
$found = $false
foreach ($pattern in $mavenDirs) {
    $matches = Get-ChildItem -Path $pattern -Directory -ErrorAction SilentlyContinue
    if ($matches) {
        $mvnBin = $matches[0].FullName
        if (Test-Path "$mvnBin\mvn.cmd" -and $env:Path -notlike "*$mvnBin*") {
            $env:Path += ";$mvnBin"
            Write-Host "  Added Maven to PATH: $mvnBin" -ForegroundColor Green
            $found = $true
            break
        }
    }
}
if (-not $found) {
    Write-Host "  Maven not found. To add it later:" -ForegroundColor Yellow
    Write-Host '    $env:Path += ";C:\path\to\apache-maven-x.x.x\bin"' -ForegroundColor Gray
}

# Ruby
$rubyPaths = @(
    "$env:ProgramFiles\Ruby\*\bin",
    "${env:ProgramFiles(x86)}\Ruby\*\bin",
    "$env:LOCALAPPDATA\Ruby\*\bin",
    "$env:USERPROFILE\.local\share\Ruby\*\bin",
    "C:\Ruby*\bin"
)
$found = $false
foreach ($pattern in $rubyPaths) {
    $matches = Get-ChildItem -Path $pattern -Directory -ErrorAction SilentlyContinue
    if ($matches) {
        $rubyBin = $matches[0].FullName
        if ($env:Path -notlike "*$rubyBin*") {
            $env:Path += ";$rubyBin"
            Write-Host "  Added Ruby to PATH: $rubyBin" -ForegroundColor Green
            $found = $true
            break
        }
    }
}
if (-not $found) {
    Write-Host "  Ruby not auto-detected. Install from https://rubyinstaller.org/ if needed." -ForegroundColor DarkGray
}

# ─── Persist PATH changes to machine scope (admin only) ───
if ($isAdmin) {
    Write-Host ""
    Write-Host "Would you like to save these PATH changes permanently? (y/n)" -ForegroundColor Yellow
    $answer = Read-Host
    if ($answer -eq "y") {
        $oldPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
        $newPath = $oldPath
        foreach ($p in $env:Path.Split(';')) {
            if ($p -and $oldPath -notlike "*$p*") {
                $newPath += ";$p"
            }
        }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "Machine")
        Write-Host "  PATH saved permanently." -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host "  Setup complete! Restart PowerShell, then:" -ForegroundColor Cyan
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "  cd keysync" -ForegroundColor White
Write-Host "  git pull" -ForegroundColor White
Write-Host '  powershell -ExecutionPolicy Bypass -File .\test-all.ps1' -ForegroundColor White
Write-Host "  git add test-results/ && git commit -m "test results: Windows"" -ForegroundColor White
Write-Host "  git push" -ForegroundColor White
Write-Host ""
