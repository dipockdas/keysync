# Cookbook: Testing Cross-Platform CLI Apps on Windows ARM/AMD64

## Overview

This guide covers the complete workflow for testing CLI applications and their multi-language client libraries on Windows virtual machines — both ARM64 and AMD64 (x86-64). It covers everything from VM setup to automated test execution.

**Tested with:** [keysync](https://github.com/dipockdas/keysync) — a cross-platform secret management CLI with client libraries in 9 languages (Go, Rust, Python, Node/TypeScript, Java, C#, C++, Ruby, Swift).

---

## 1. Virtual Machine Setup

### Host Machine
- macOS (ARM64 / Apple Silicon)

### Virtualization Software
- **VirtualBox** (version 7.x+, with Apple Silicon support)
  - Download from: https://www.virtualbox.org/
  - Requires Extension Pack for ARM64 support
  - Note: VirtualBox for Apple Silicon uses ARM64 virtualization (not emulation), so guest must be ARM64

### Windows Guest Downloads

| Architecture | Source | Notes |
|-------------|--------|-------|
| ARM64 | [Windows 11 ARM64 Insider Preview](https://www.microsoft.com/en-us/software-download/windowsinsiderpreviewARM64) | Requires free Microsoft account + Insider Program enrollment |
| AMD64 (x86-64) | [Windows 11 Dev VM](https://developer.microsoft.com/en-us/windows/downloads/virtual-machines/) | Pre-built, expires after 90 days |

### VM Configuration

Recommended specs:
- **RAM:** 8 GB minimum, 16 GB preferred (especially for AMD64 on Apple Silicon — runs via emulation/translation)
- **Storage:** 128 GB dynamically allocated
- **CPU:** 2-4 cores
- **Network:** NAT (for internet access), optional Host-Only (for SSH/file transfer)

### Performance Notes
- **ARM64 VM:** Runs natively on Apple Silicon. Fast, responsive, near-native performance.
- **AMD64 VM:** Runs via x86-to-ARM translation. Noticeably slower. Builds take 2-3x longer. Patience required.

---

## 2. First Boot & Guest Setup

After installing Windows on the VM:

### Essential First Steps

1. **Set execution policy** (so PowerShell scripts can run):
   ```powershell
   Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser -Force
   ```

2. **Install git** (if not pre-installed):
   ```powershell
   winget install --accept-source-agreements --accept-package-agreements --exact -h "Git.Git"
   ```

3. **Clone your repo:**
   ```powershell
   mkdir C:\Projects
   cd C:\Projects
   git clone https://github.com/your-org/your-repo.git
   cd your-repo
   ```

4. **Install VS Code (optional)** for editing scripts remotely:
   ```powershell
   winget install --accept-source-agreements --accept-package-agreements --exact -h "Microsoft.VisualStudioCode"
   ```

---

## 3. Toolchain Installation

### The Setup Script

The repo should contain `setup-windows.ps1` that automates all toolchain installation. Run it as **Administrator**:

```powershell
# As Administrator:
powershell -ExecutionPolicy Bypass -File .\setup-windows.ps1
```

### What the Script Installs

| # | Tool | Package ID / Source | Purpose |
|---|------|-------------------|---------|
| 1 | **Go** | `GoLang.Go` (winget) | Core CLI + Go client tests |
| 2 | **Rust** | `Rustlang.Rustup` (winget) | Rust client tests |
| 3 | **Java JDK 21** | `EclipseAdoptium.Temurin.21.JDK` (winget) | Java client tests (Maven) |
| 4 | **Python 3.14** | `Python.Python.3.14` (winget) | Python client tests |
| 5 | **Node.js LTS** | `OpenJS.NodeJS.LTS` (winget) | TypeScript client tests |
| 6 | **Ruby** | `RubyInstallerTeam.Ruby.4.0` (winget) | Ruby client tests |
| 7 | **CMake** | `Kitware.CMake` (winget) | C++ client build |
| 8 | **VS Build Tools 2022** | `Microsoft.VisualStudio.2022.BuildTools` (winget) | C++ + Rust compilation |
| 9 | **Maven** | Manual download from apache.org | Java client build/test |

### After Winget Installs

**Rustup init:**
```powershell
rustup default stable
```

**VS C++ workload** (critical — the `VCTools` workload is required for C++ and Rust MSVC compilation):
- The script runs this automatically via `vs_installer.exe`
- Workload ID: `Microsoft.VisualStudio.Workload.VCTools`
- **Do NOT use `NativeDesktop`** — that workload ID only exists in the full VS IDE, NOT in Build Tools

**Maven** (downloaded by the script):
- Downloaded from: `https://dlcdn.apache.org/maven/maven-3/3.9.16/binaries/apache-maven-3.9.16-bin.zip`
- Extracted to: `C:\Tools\apache-maven-3.9.16\`

### Manual Fallbacks

If the setup script fails for individual tools:

| Tool | Fallback Command |
|------|-----------------|
| Rust | `winget install Rustlang.Rustup` then `rustup default stable` |
| Java | `winget install EclipseAdoptium.Temurin.21.JDK` |
| Maven | `Invoke-WebRequest https://dlcdn.apache.org/maven/maven-3/3.9.16/binaries/apache-maven-3.9.16-bin.zip -OutFile maven.zip` then extract to `C:\Tools\` |
| VS C++ | `vs_installer.exe modify --installPath "$vsPath" --add Microsoft.VisualStudio.Workload.VCTools --quiet` |

---

## 4. Environment Setup

### PATH Variables

The setup script auto-detects and adds these to the session PATH:

| Tool | Typical Install Location |
|------|------------------------|
| Go | `C:\Program Files\Go\bin` |
| Rust/Cargo | `%USERPROFILE%\.cargo\bin` |
| Java | `C:\Program Files\Eclipse Adoptium\jdk-21.0.11.10-hotspot\bin` |
| Python | `%LOCALAPPDATA%\Programs\Python\Python314-arm64\` (ARM) or `Python314\` (x86) |
| Node.js | `C:\Program Files\nodejs\` |
| Ruby | `C:\Ruby40-arm\bin` (ARM) or `C:\Ruby40-x64\bin` (x86) |
| CMake | `C:\Program Files\CMake\bin` |
| Maven | `C:\Tools\apache-maven-3.9.16\bin` |

### VS Dev Environment

For Rust and C++ compilation, the MSVC environment must be initialized. The test script (`test-all.ps1`) does NOT set this up automatically — you need to do it manually in your PowerShell session:

```powershell
# ARM64:
& "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvarsall.bat" arm64

# AMD64 (x86):
& "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvarsall.bat" amd64
```

To make this permanent, add it to your PowerShell profile:
```powershell
Add-Content -Path $PROFILE -Value '& "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvarsall.bat" arm64'
```

**Important:** On ARM64, the native linker is at:
```
C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Tools\MSVC\14.x.xxxxx\bin\Hostarm64\arm64\link.exe
```

---

## 5. Running Tests

### The Test Script

The repo contains `test-all.ps1` that discovers installed tools and runs all client tests:

```powershell
powershell -ExecutionPolicy Bypass -File .\test-all.ps1
```

The script:
1. Auto-detects tools on PATH (checks `$env:Path` and common install locations)
2. Runs each client's test suite
3. Saves results to `test-results/test-<Platform>-<Arch>-<Timestamp>.txt`
4. Outputs a summary table

### What Gets Tested

| Client | Test Command | Expect |
|--------|-------------|--------|
| Go (core CLI) | `make test` | ~160 tests across 6 packages |
| Go (client) | `go test ./...` | 30 tests |
| Rust | `cargo test` | 45 unit + 3 doc-tests |
| Python | `pytest` | 28 tests |
| Node/TypeScript | `vitest run` | 36 tests (3 files) |
| Java | `mvn test` | 52 tests (2 skipped on non-macOS) |
| C# (.NET) | `dotnet test` | 34 tests (xUnit, 6 classes) |
| C++ | CMake build + run | 27 test functions (2 files) |
| Ruby | `rake test` | 10 runs, 29 assertions |
| Swift | (macOS only) | Skipped on Windows |

### Architecture-Specific Notes

- **Swift:** macOS-only. Always skipped on Windows.
- **Java PlatformDetectionTests:** 2 tests skip on non-macOS (expected).
- **Go TestF_WithColors:** Skips in no-color terminal (expected).
- **Rust:** On ARM64, `aarch64-pc-windows-msvc` target is used with the MSVC linker.

---

## 6. Test Result Workflow

### Saving Results

After running tests, commit and push the results:

```powershell
git add test-results/
git commit -m "test results: Windows <Arch>"
git push
```

### Result File Naming

```
test-results/test-<OS>-<Arch>-<YYYYMMDD>-<HHmmss>.txt
```

Examples:
- `test-Windows_NT-ARM64-20260519-133501.txt`
- `test-Windows_NT-AMD64-20260519-142554.txt`

### Parsing Results

Each result file contains:
- Platform/architecture header
- Available tools detected
- Per-client test output (with PASS/FAIL/SKIP markers)
- Summary table at the bottom

---

## 7. Common Issues & Troubleshooting

### "C++ toolchain not found" (Rust)

**Error:** `link.exe not found`
**Fix:** Ensure VS Build Tools are installed AND `vcvarsall.bat` has been run in the current terminal:
```powershell
& "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\VC\Auxiliary\Build\vcvarsall.bat" arm64
```

### Rust target missing on ARM64

**Error:** `toolchain 'stable-gnu' is not installed`
**Fix:** Use MSVC target (not GNU) on ARM64:
```powershell
rustup default stable-msvc
```
The `gnu` toolchain requires additional LLVM tools. Stick with `msvc` when VS Build Tools are installed.

### Maven not on PATH

Maven is downloaded manually to `C:\Tools\apache-maven-3.9.16\bin`. If the setup script doesn't find it:
```powershell
$env:Path += ";C:\Tools\apache-maven-3.9.16\bin"
```

### VS Installer hangs or "NativeDesktop not found"

The workload ID `Microsoft.VisualStudio.Workload.NativeDesktop` does NOT exist in Visual Studio Build Tools. It only exists in the full IDE. Always use:
```
Microsoft.VisualStudio.Workload.VCTools
```

### CMake CTest fails on MSVC

MSVC uses multi-config generators. You must specify the config:
```powershell
cmake --build . --config Debug
ctest -C Debug
```

### setenv/unsetenv not found (C++ tests)

MSVC does not have POSIX `setenv`/`unsetenv`. Use `_putenv_s` instead:
```cpp
_putenv_s("KEY", "value");   // set
_putenv_s("KEY", "");        // unset
```

### Long VS Installer operations

On ARM64, the VS Build Tools installer can take 30+ minutes and may appear to hang. Be patient. Check progress via Task Manager (look for `vs_installer.exe`).

---

## 8. Architecture Differences Summary

| Aspect | ARM64 | AMD64 (x86-64) |
|--------|-------|----------------|
| **VM performance** | Fast (native) | Slow (translated) |
| **Ruby path** | `C:\Ruby40-arm\bin` | `C:\Ruby40-x64\bin` |
| **Python path** | `Python314-arm64\` | `Python314\` |
| **VS vcvars arch** | `arm64` | `amd64` |
| **Rust target** | `aarch64-pc-windows-msvc` | `x86_64-pc-windows-msvc` |
| **link.exe** | `Hostarm64\arm64\link.exe` | `Hostx64\x64\link.exe` |

The test script handles most of these automatically via PATH auto-detection patterns like `C:\Ruby*\bin` and `Python314*\bin`.

---

## 9. Tools Reference

### Winget Package IDs (Verified)

| Package | Winget ID | Notes |
|---------|-----------|-------|
| Go | `GoLang.Go` | |
| Rustup | `Rustlang.Rustup` | |
| Java JDK 21 | `EclipseAdoptium.Temurin.21.JDK` | |
| Python 3.14 | `Python.Python.3.14` | |
| Node.js LTS | `OpenJS.NodeJS.LTS` | |
| Ruby | `RubyInstallerTeam.Ruby.4.0` | |
| CMake | `Kitware.CMake` | |
| VS Build Tools 2022 | `Microsoft.VisualStudio.2022.BuildTools` | Current/only version |
| VS C++ workload | `Microsoft.VisualStudio.Workload.VCTools` | Use this in Build Tools |
| Git | `Git.Git` | |

**NOT available on winget:**
- **Maven** — rejected by Microsoft because `.cmd` installers not allowed. Download manually from Apache.
- **.NET SDK** — not needed if testing existing binaries; use `dotnet` SDK which may come with VS Build Tools.

### Visual Studio Workload IDs

| Product | C++ Workload ID |
|---------|----------------|
| **Build Tools** (no IDE) | `Microsoft.VisualStudio.Workload.VCTools` |
| **Full IDE** (Community/Pro/Enterprise) | `Microsoft.VisualStudio.Workload.NativeDesktop` |

⚠️ **Using `NativeDesktop` with Build Tools silently fails. Always use `VCTools`.**

---

## 10. Quick-Start Checklist

- [ ] VirtualBox installed on macOS
- [ ] Windows VM created (ARM64 or AMD64)
- [ ] VM boots, internet works
- [ ] Execution policy set: `Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force`
- [ ] Git installed: `winget install Git.Git`
- [ ] Repo cloned: `git clone https://github.com/your-org/your-repo`
- [ ] Setup script run as Admin: `.\setup-windows.ps1`
- [ ] vcvars initialized for correct arch
- [ ] Rust default set: `rustup default stable`
- [ ] PATH has all tools (restart shell or run PATH setup)
- [ ] Tests run: `.\test-all.ps1`
- [ ] Results committed and pushed
