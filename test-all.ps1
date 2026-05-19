#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Run all keysync client library tests.
.DESCRIPTION
    Discovers which tools are available and runs each client's test suite.
    Skips clients whose toolchain is not installed. Writes a report to test-results/.
#>

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$totalPassed = 0
$totalFailed = 0
$totalSkipped = 0
$results = @()

# Build log file path: test-results/<platform>-<arch>-<timestamp>.txt
$platform = if ($env:OS) { $env:OS } else { "unknown" }
$arch = if ($env:PROCESSOR_ARCHITECTURE) { $env:PROCESSOR_ARCHITECTURE } else { "unknown" }
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$logDir = Join-Path $root "test-results"
$logFile = Join-Path $logDir "test-$platform-$arch-$timestamp.txt"
$null = New-Item -ItemType Directory -Path $logDir -Force

# Log accumulator (console + file)
$logLines = [System.Collections.ArrayList]@()
function Write-Log {
    param([string]$Message, [string]$ForegroundColor = "White")
    Write-Host $Message -ForegroundColor $ForegroundColor
    $null = $logLines.Add($Message)
}

function Run-Tests {
    param(
        [string]$Name,
        [string]$Dir,
        [scriptblock]$Script
    )

    Write-Log "" -ForegroundColor Cyan
    Write-Log "========================================" -ForegroundColor Cyan
    Write-Log "  $Name" -ForegroundColor Cyan
    Write-Log "========================================" -ForegroundColor Cyan

    Push-Location (Join-Path $root $Dir)
    try {
        # Capture test output
        $output = & $Script 2>&1
        $ec = $LASTEXITCODE
        $output | ForEach-Object { $null = $logLines.Add("$_") }
        $output | Out-Host

        if ($ec -eq 0 -or $ec -eq $null) {
            Write-Log "  PASS" -ForegroundColor Green
            $script:totalPassed++
            $results += [PSCustomObject]@{ Client = $Name; Result = "PASS" }
        } else {
            Write-Log "  FAIL (exit code: $ec)" -ForegroundColor Red
            $script:totalFailed++
            $results += [PSCustomObject]@{ Client = $Name; Result = "FAIL" }
        }
    } catch {
        Write-Log "  ERROR: $_" -ForegroundColor Red
        $null = $logLines.Add("  ERROR: $_")
        $script:totalFailed++
        $results += [PSCustomObject]@{ Client = $Name; Result = "FAIL" }
    }
    finally {
        Pop-Location
    }
}

function Skip-Tests {
    param([string]$Name)

    Write-Log "" -ForegroundColor Gray
    Write-Log "  $Name - skipped (tool not found)" -ForegroundColor Gray
    $script:totalSkipped++
    $results += [PSCustomObject]@{ Client = $Name; Result = "SKIP" }
}

# ====== Header ======
Write-Log "======================================================================" -ForegroundColor Cyan
Write-Log "  keysync - Running All Client Library Tests" -ForegroundColor Cyan
Write-Log "  Platform: $platform / $arch" -ForegroundColor Cyan
Write-Log "  Log file: $logFile" -ForegroundColor Cyan
Write-Log "======================================================================" -ForegroundColor Cyan
Write-Log ""

# ====== Auto-fix PATH for common install locations ======
$null = @(
    # Cargo/Rust
    @("$env:USERPROFILE\.cargo\bin", "cargo.exe"),
    @("$env:LOCALAPPDATA\.cargo\bin", "cargo.exe"),
    # Maven
    @("$env:USERPROFILE\Downloads\apache-maven-*\bin", "mvn.cmd"),
    @("$env:USERPROFILE\Tools\apache-maven-*\bin", "mvn.cmd"),
    @("$env:ProgramFiles\Apache\maven\bin", "mvn.cmd"),
    @("${env:ProgramFiles(x86)}\Apache\maven\bin", "mvn.cmd"),
    @("$env:ProgramFiles\Maven\apache-maven-*\bin", "mvn.cmd"),
    # CMake
    @("${env:ProgramFiles}\CMake\bin", "cmake.exe"),
    @("${env:ProgramFiles(x86)}\CMake\bin", "cmake.exe"),
    # Ruby
    @("$env:ProgramFiles\Ruby\*\bin", "ruby.exe"),
    @("${env:ProgramFiles(x86)}\Ruby\*\bin", "ruby.exe"),
    @("$env:LOCALAPPDATA\Ruby\*\bin", "ruby.exe"),
    @("$env:USERPROFILE\.local\share\Ruby\*\bin", "ruby.exe"),
    @("C:\Ruby*\bin", "ruby.exe"),
    # Go
    @("$env:ProgramFiles\Go\bin", "go.exe"),
    @("${env:ProgramFiles(x86)}\Go\bin", "go.exe")
) | ForEach-Object {
    $dirPattern = $_[0]
    $exe = $_[1]
    $matches = Get-ChildItem -Path $dirPattern -Directory -ErrorAction SilentlyContinue | Where-Object {
        Test-Path (Join-Path $_.FullName $exe)
    }
    if ($matches) {
        $binDir = $matches[0].FullName
        if ($env:Path -notlike "*$binDir*") {
            $env:Path = "$binDir;$env:Path"
        }
    }
}

# ====== Detect available tools ======
$tools = @{
    dotnet   = Get-Command dotnet   -ErrorAction SilentlyContinue
    cargo    = Get-Command cargo    -ErrorAction SilentlyContinue
    go       = Get-Command go       -ErrorAction SilentlyContinue
    node     = Get-Command node     -ErrorAction SilentlyContinue
    npm      = Get-Command npm      -ErrorAction SilentlyContinue
    python   = Get-Command python   -ErrorAction SilentlyContinue
    mvn      = Get-Command mvn      -ErrorAction SilentlyContinue
    ruby     = Get-Command ruby     -ErrorAction SilentlyContinue
    cmake    = Get-Command cmake    -ErrorAction SilentlyContinue
}

Write-Log "Available tools:" -ForegroundColor Yellow
foreach ($kv in $tools.GetEnumerator()) {
    if ($kv.Value) { Write-Log "  [ok] $($kv.Key)" -ForegroundColor Green }
    else           { Write-Log "  [--] $($kv.Key)" -ForegroundColor DarkGray }
}
Write-Log ""

# ====== C# ======
if ($tools.dotnet) {
    Run-Tests -Name "C# (.NET)" -Dir "clients/csharp" -Script {
        dotnet test --no-restore --logger "console;verbosity=detailed" 2>&1
        if ($LASTEXITCODE -ne 0) {
            dotnet restore 2>&1 | Out-Null
            dotnet test --logger "console;verbosity=detailed" 2>&1
        }
    }
} else { Skip-Tests "C# (.NET)" }

# ====== Rust ======
if ($tools.cargo) {
    Run-Tests -Name "Rust" -Dir "clients/rust" -Script {
        cargo test 2>&1
    }
} else { Skip-Tests "Rust" }

# ====== Go (core) ======
if ($tools.go) {
    Run-Tests -Name "Go (core)" -Dir "." -Script {
        go test ./internal/... -v -count=1 2>&1
    }
} else { Skip-Tests "Go (core)" }

# ====== Go (client) ======
if ($tools.go) {
    Run-Tests -Name "Go (client)" -Dir "clients/go" -Script {
        go test ./... -v -count=1 2>&1
    }
} else { Skip-Tests "Go (client)" }

# ====== Node ======
if ($tools.node -and $tools.npm) {
    Run-Tests -Name "Node/TypeScript" -Dir "clients/node" -Script {
        npm install 2>&1
        npm test 2>&1
    }
} else { Skip-Tests "Node/TypeScript" }

# ====== Python ======
if ($tools.python) {
    Run-Tests -Name "Python" -Dir "clients/python" -Script {
        # Install package in dev mode + pytest if needed
        pip install -e . 2>&1 | Out-Null
        pip install pytest -q 2>&1 | Out-Null
        $env:PYTHONPATH = "src"
        python -m pytest tests/ -v 2>&1
    }
} else { Skip-Tests "Python" }

# ====== Java ======
if ($tools.mvn) {
    Run-Tests -Name "Java" -Dir "clients/java" -Script {
        mvn test 2>&1
    }
} else { Skip-Tests "Java" }

# ====== Swift (macOS only) ======
if ($env:OS -eq "Windows_NT") {
    Skip-Tests "Swift (macOS only)"
} else {
    $swift = Get-Command swift -ErrorAction SilentlyContinue
    if ($swift) {
        Run-Tests -Name "Swift" -Dir "clients/swift" -Script {
            swift test 2>&1
        }
    } else { Skip-Tests "Swift" }
}

# ====== Ruby ======
if ($tools.ruby) {
    Run-Tests -Name "Ruby" -Dir "clients/ruby" -Script {
        ruby -Ilib -Itest test/test_*.rb 2>&1
    }
} else { Skip-Tests "Ruby" }

# ====== C++ ======
if ($tools.cmake) {
    Run-Tests -Name "C++" -Dir "clients/cpp" -Script {
        if (-not (Test-Path build)) { New-Item -ItemType Directory -Path build | Out-Null }
        Push-Location build
        cmake .. 2>&1
        cmake --build . --config Debug 2>&1
        # Run test executable directly for detailed output per test function
        if (Test-Path "tests\Debug\keysync_tests.exe") {
            .\tests\Debug\keysync_tests.exe 2>&1
        } elseif (Test-Path "tests\Release\keysync_tests.exe") {
            .\tests\Release\keysync_tests.exe 2>&1
        } else {
            # Fallback: try running from ctest
            ctest --output-on-failure -C Debug 2>&1
        }
        Pop-Location
    }
} else { Skip-Tests "C++" }

# ====== Summary ======
Write-Log ""
Write-Log "======================================================================" -ForegroundColor Cyan
Write-Log "  Results Summary" -ForegroundColor Cyan
Write-Log "======================================================================" -ForegroundColor Cyan
Write-Log ""

$results | ForEach-Object {
    $line = "  $($_.Client.PadRight(20)) $($_.Result)"
    Write-Log $line
}

Write-Log ""
$color = if ($totalFailed -gt 0) { "Red" } elseif ($totalSkipped -gt 0) { "Yellow" } else { "Green" }
Write-Log "  Passed: $totalPassed   Failed: $totalFailed   Skipped: $totalSkipped" -ForegroundColor $color
Write-Log ""

# ====== Write log file ======
$logLines | Out-File -FilePath $logFile -Encoding utf8
Write-Log "  Full log written to: $logFile" -ForegroundColor Cyan
Write-Log ""

if ($totalFailed -gt 0) {
    Write-Log "  Some tests FAILED. Review output above." -ForegroundColor Red
    exit 1
} else {
    Write-Log "  All run tests PASSED." -ForegroundColor Green
    exit 0
}
