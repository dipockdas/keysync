#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Run all keysync client library tests.
.DESCRIPTION
    Discovers which tools are available and runs each client's test suite.
    Skips clients whose toolchain is not installed.
#>

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$totalPassed = 0
$totalFailed = 0
$totalSkipped = 0
$results = @()

function Run-Tests {
    param(
        [string]$Name,
        [string]$Dir,
        [scriptblock]$Script
    )

    Write-Host "`n========================================" -ForegroundColor Cyan
    Write-Host "  $Name" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan

    Push-Location (Join-Path $root $Dir)
    try {
        & $Script
        if ($LASTEXITCODE -eq 0 -or $LASTEXITCODE -eq $null) {
            Write-Host "  PASS" -ForegroundColor Green
            $script:totalPassed++
            $results += [PSCustomObject]@{ Client = $Name; Result = "PASS" }
        } else {
            Write-Host "  FAIL (exit code: $LASTEXITCODE)" -ForegroundColor Red
            $script:totalFailed++
            $results += [PSCustomObject]@{ Client = $Name; Result = "FAIL" }
        }
    } catch {
        Write-Host "  ERROR: $_" -ForegroundColor Red
        $script:totalFailed++
        $results += [PSCustomObject]@{ Client = $Name; Result = "FAIL" }
    }
    finally {
        Pop-Location
    }
}

function Skip-Tests {
    param([string]$Name)

    Write-Host "`n----------------------------------------" -ForegroundColor Gray
    Write-Host "  $Name - skipped (tool not found)" -ForegroundColor Gray
    $script:totalSkipped++
    $results += [PSCustomObject]@{ Client = $Name; Result = "SKIP" }
}

Write-Host "======================================================================" -ForegroundColor Cyan
Write-Host "  keysync - Running All Client Library Tests" -ForegroundColor Cyan
Write-Host "======================================================================" -ForegroundColor Cyan
Write-Host ""

# Detect available tools
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

Write-Host "Available tools:" -ForegroundColor Yellow
foreach ($kv in $tools.GetEnumerator()) {
    if ($kv.Value) { Write-Host "  [ok] $($kv.Key)" -ForegroundColor Green }
    else           { Write-Host "  [--] $($kv.Key)" -ForegroundColor DarkGray }
}
Write-Host ""

# --- C# ---
if ($tools.dotnet) {
    Run-Tests -Name "C# (.NET)" -Dir "clients/csharp" -Script {
        dotnet test --no-restore 2>&1
        if ($LASTEXITCODE -ne 0) {
            dotnet restore 2>&1 | Out-Null
            dotnet test 2>&1
        }
    }
} else { Skip-Tests "C# (.NET)" }

# --- Rust ---
if ($tools.cargo) {
    Run-Tests -Name "Rust" -Dir "clients/rust" -Script {
        cargo test 2>&1
    }
} else { Skip-Tests "Rust" }

# --- Go (core) ---
if ($tools.go) {
    Run-Tests -Name "Go (core)" -Dir "." -Script {
        go test ./internal/... -v -count=1 2>&1
    }
} else { Skip-Tests "Go (core)" }

# --- Go (client) ---
if ($tools.go) {
    Run-Tests -Name "Go (client)" -Dir "clients/go" -Script {
        go test ./... -v -count=1 2>&1
    }
} else { Skip-Tests "Go (client)" }

# --- Node ---
if ($tools.node -and $tools.npm) {
    Run-Tests -Name "Node/TypeScript" -Dir "clients/node" -Script {
        npm install 2>&1
        npm test 2>&1
    }
} else { Skip-Tests "Node/TypeScript" }

# --- Python ---
if ($tools.python) {
    Run-Tests -Name "Python" -Dir "clients/python" -Script {
        python -m pytest tests/ -v 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Retrying with pytest install..." -ForegroundColor Yellow
            pip install pytest -q 2>&1 | Out-Null
            python -m pytest tests/ -v 2>&1
        }
    }
} else { Skip-Tests "Python" }

# --- Java ---
if ($tools.mvn) {
    Run-Tests -Name "Java" -Dir "clients/java" -Script {
        mvn test 2>&1
    }
} else { Skip-Tests "Java" }

# --- Swift (macOS only) ---
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

# --- Ruby ---
if ($tools.ruby) {
    Run-Tests -Name "Ruby" -Dir "clients/ruby" -Script {
        ruby -Ilib -Itest test/test_*.rb 2>&1
    }
} else { Skip-Tests "Ruby" }

# --- C++ ---
if ($tools.cmake) {
    Run-Tests -Name "C++" -Dir "clients/cpp" -Script {
        if (-not (Test-Path build)) { New-Item -ItemType Directory -Path build | Out-Null }
        Push-Location build
        cmake .. 2>&1
        cmake --build . 2>&1
        ctest --output-on-failure 2>&1
        Pop-Location
    }
} else { Skip-Tests "C++" }

# --- Summary ---
Write-Host "`n======================================================================" -ForegroundColor Cyan
Write-Host "  Results Summary" -ForegroundColor Cyan
Write-Host "======================================================================" -ForegroundColor Cyan
Write-Host ""
$results | Format-Table -AutoSize

Write-Host ""
$color = if ($totalFailed -gt 0) { "Red" } elseif ($totalSkipped -gt 0) { "Yellow" } else { "Green" }
Write-Host "  Passed: $totalPassed   Failed: $totalFailed   Skipped: $totalSkipped" -ForegroundColor $color
Write-Host ""

if ($totalFailed -gt 0) {
    Write-Host "  Some tests FAILED. Review output above." -ForegroundColor Red
    exit 1
} else {
    Write-Host "  All run tests PASSED." -ForegroundColor Green
    exit 0
}
