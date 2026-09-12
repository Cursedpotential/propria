#Requires -Version 7.0
# Dot-source into a task-local shell. Never changes user/machine environment.
$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'This Intake launcher requires Windows.' }
$intakeRepoRoot = [IO.Path]::GetFullPath((Split-Path $PSScriptRoot -Parent))
if (-not $intakeRepoRoot.StartsWith('E:\', [StringComparison]::OrdinalIgnoreCase)) {
    throw 'The Intake checkout/build output must remain on the approved E: development drive.'
}
$intakeDevRoot = 'E:\AI_Workspace\.intake-dev'
if (-not (Test-Path -LiteralPath 'E:\')) { throw 'E: is required; no storage fallback.' }
foreach ($name in @('temp', 'cargo', 'rustup', 'npm', 'logs', 'runtime')) {
    New-Item -ItemType Directory -Path (Join-Path $intakeDevRoot $name) -Force | Out-Null
}
$env:TEMP = Join-Path $intakeDevRoot 'temp'
$env:TMP = $env:TEMP
$env:CARGO_HOME = Join-Path $intakeDevRoot 'cargo'
$env:RUSTUP_HOME = Join-Path $intakeDevRoot 'rustup'
$env:npm_config_cache = Join-Path $intakeDevRoot 'npm'
$env:CARGO_BUILD_JOBS = '1'
$env:CARGO_PROFILE_DEV_DEBUG = '0'
$env:CARGO_PROFILE_TEST_DEBUG = '0'
$env:CARGO_INCREMENTAL = '0'
$env:XPLORER_INTAKE_MODE = '1'
$env:VITE_INTAKE_MODE = '1'
$env:XPLORER_RUNTIME_DIR = Join-Path $intakeDevRoot 'runtime'
$intakeToolchain = Join-Path $intakeDevRoot 'toolchains\1.91.1\bin'
foreach ($intakeTool in @('rustc.exe', 'cargo.exe', 'rustdoc.exe')) {
    if (-not (Test-Path -LiteralPath (Join-Path $intakeToolchain $intakeTool) -PathType Leaf)) {
        throw "Pinned Rust 1.91.1 tool $intakeTool must be provisioned on E: before building."
    }
}
$env:VSCMD_SKIP_SENDTELEMETRY = '1'
$intakeVsShell = 'C:\Program Files (x86)\Microsoft Visual Studio\18\BuildTools\Common7\Tools\Launch-VsDevShell.ps1'
if (-not (Test-Path -LiteralPath $intakeVsShell -PathType Leaf)) { throw 'Visual Studio Build Tools shell is unavailable.' }
& $intakeVsShell -Arch amd64 -HostArch amd64 -SkipAutomaticLocation -NoLogo
# Developer shell setup may override environment values; reassert child temp paths.
$env:TEMP = Join-Path $intakeDevRoot 'temp'
$env:TMP = $env:TEMP
$env:PATH = $intakeToolchain + ';' + $env:PATH
$env:RUSTC = Join-Path $intakeToolchain 'rustc.exe'
$env:RUSTDOC = Join-Path $intakeToolchain 'rustdoc.exe'
$env:CARGO_TARGET_DIR = Join-Path (Split-Path $PSScriptRoot -Parent) 'apps\src-tauri\target'
if (-not (Get-Command link.exe -ErrorAction SilentlyContinue)) { throw 'MSVC linker is unavailable.' }
$intakeSdk = Join-Path $intakeDevRoot 'windows-sdk\microsoft.windows.sdk.cpp\c'
$intakeSdkLibs = Join-Path $intakeDevRoot 'windows-sdk\microsoft.windows.sdk.cpp.x64\c'
if (-not (Test-Path -LiteralPath (Join-Path $intakeSdkLibs 'um\x64\kernel32.lib'))) {
    throw 'Microsoft Windows SDK NuGet packages must be provisioned on E:.'
}
foreach ($intakeSdkFile in @('Include\10.0.28000.0\um\Windows.h', 'Include\10.0.28000.0\ucrt\stdlib.h', 'bin\10.0.28000.0\x64\rc.exe')) {
    if (-not (Test-Path -LiteralPath (Join-Path $intakeSdk $intakeSdkFile) -PathType Leaf)) {
        throw "Required E: Windows SDK component missing: $intakeSdkFile"
    }
}
$env:WindowsSdkDir = $intakeSdk + '\'
$env:WindowsSDKVersion = '10.0.28000.0\'
$env:LIB = (Join-Path $intakeSdkLibs 'um\x64') + ';' + (Join-Path $intakeSdkLibs 'ucrt\x64') + ';' + $env:LIB
foreach ($include in @('ucrt', 'shared', 'um', 'winrt', 'cppwinrt')) {
    $env:INCLUDE = (Join-Path $intakeSdk ('Include\10.0.28000.0\' + $include)) + ';' + $env:INCLUDE
}
$env:PATH = (Join-Path $intakeSdk 'bin\10.0.28000.0\x64') + ';' + $env:PATH
