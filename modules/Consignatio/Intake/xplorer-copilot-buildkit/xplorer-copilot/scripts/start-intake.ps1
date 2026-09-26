#Requires -Version 7.4
[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'intake-env.ps1')
$intakeRepo = Split-Path $PSScriptRoot -Parent
$intakeVitePath = Join-Path $intakeRepo 'node_modules\vite\bin\vite.js'
$intakeCliPath = Join-Path $intakeRepo 'node_modules\@tauri-apps\cli\tauri.js'
$intakeNode = (Get-Command node -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source
foreach ($intakeRequiredPath in @($intakeVitePath, $intakeCliPath)) {
    if (-not (Test-Path -LiteralPath $intakeRequiredPath -PathType Leaf)) {
        throw "Required local dependency is missing: $intakeRequiredPath. Provision dependencies on E: first."
    }
}
# These files merge in this order on Windows. This launcher owns Vite; a
# future beforeDevCommand must not silently start a second frontend process.
$intakeBeforeDev = $null
foreach ($intakeConfigName in @('tauri.conf.json', 'tauri.windows.conf.json', 'tauri.intake.conf.json')) {
    $intakeConfigPath = Join-Path $intakeRepo ('apps\src-tauri\' + $intakeConfigName)
    if (Test-Path -LiteralPath $intakeConfigPath -PathType Leaf) {
        $intakeConfig = Get-Content -LiteralPath $intakeConfigPath -Raw -Encoding utf8 | ConvertFrom-Json
        if ($intakeConfig.build -and $intakeConfig.build.PSObject.Properties.Name -contains 'beforeDevCommand') {
            $intakeBeforeDev = $intakeConfig.build.beforeDevCommand
        }
    }
}
if ($intakeBeforeDev) { throw 'Tauri beforeDevCommand conflicts with launcher-managed Vite; reconcile the dev configuration before starting.' }
$env:INTAKE_PORTKEY_URL = 'http://100.72.169.40:8787'
$env:INTAKE_PORTKEY_CONFIG_FILE = Join-Path $PSScriptRoot 'intake-portkey-chat.json'
$env:INTAKE_CHAT_MODEL = 'portkey:nemotron-3-super'
$env:VITE_INTAKE_CHAT_MODEL = $env:INTAKE_CHAT_MODEL
# Read only the required existing key. Never copy secrets to source or renderer.
if (-not $env:OLLAMA_API_KEY) {
    $intakeSecretFile = 'C:\Users\matts\.secrets\probata.env'
    if (Test-Path -LiteralPath $intakeSecretFile) {
        $intakeKeyPattern = '^\s*(?:export\s+)?OLLAMA_API_KEY\s*=\s*(?:"([^"]*)"|''([^'']*)''|([^#\s]+))\s*(?:#.*)?$'
        foreach ($intakeKeyLine in [System.IO.File]::ReadLines($intakeSecretFile)) {
            $intakeKeyMatch = [regex]::Match($intakeKeyLine, $intakeKeyPattern)
            if ($intakeKeyMatch.Success) {
                foreach ($intakeGroup in 1..3) {
                    if ($intakeKeyMatch.Groups[$intakeGroup].Success) {
                        $env:OLLAMA_API_KEY = $intakeKeyMatch.Groups[$intakeGroup].Value
                        break
                    }
                }
                break
            }
        }
    }
}
if (-not $env:OLLAMA_API_KEY) { Write-Warning 'Remote chat is unconfigured; file browsing can still start.' }
# Backend remains opt-in until its dedicated Weaviate collection is verified.
# INTAKE_FILESYSTEM_API_URL and token are inherited only by the native host.
$intakeListener = @(Get-NetTCPConnection -State Listen -LocalPort 5176 -ErrorAction SilentlyContinue)
$intakeViteProcess = $null
if (-not $intakeListener) {
    # Null removes only these variables from the child, without mutating this
    # shell's credential-bearing environment needed by the native host.
    $intakeFrontendEnvironment = @{}
    foreach ($intakeVariable in Get-ChildItem Env:) {
        if ($intakeVariable.Name -match '(?i)(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|AUTHORIZATION)|^INTAKE_(PORTKEY|FILESYSTEM)') {
            $intakeFrontendEnvironment[$intakeVariable.Name] = $null
        }
    }
    $intakeStamp = Get-Date -Format 'yyyyMMdd-HHmmss-fff'
    $intakeViteProcess = Start-Process -FilePath $intakeNode -ArgumentList @(('"' + $intakeVitePath + '"'), '--host', '127.0.0.1', '--port', '5176', '--strictPort') -WorkingDirectory $intakeRepo -WindowStyle Hidden -Environment $intakeFrontendEnvironment -PassThru -RedirectStandardOutput (Join-Path $intakeDevRoot "logs\vite-$intakeStamp.stdout.log") -RedirectStandardError (Join-Path $intakeDevRoot "logs\vite-$intakeStamp.stderr.log")
} else {
    foreach ($intakeBinding in $intakeListener) {
        $intakeExisting = Get-CimInstance Win32_Process -Filter "ProcessId=$($intakeBinding.OwningProcess)"
        $intakeCommandLine = [string]$intakeExisting.CommandLine
        if ($intakeBinding.LocalAddress -ne '127.0.0.1' -or $intakeExisting.Name -ne 'node.exe' -or
            $intakeCommandLine.Replace('/', '\').IndexOf($intakeVitePath, [StringComparison]::OrdinalIgnoreCase) -lt 0) {
            throw 'Port 5176 is occupied by an unrecognized or non-loopback process; leaving it untouched.'
        }
    }
}
# Do not start a native compilation against a failed/not-yet-ready frontend.
$intakeFrontendReady = $false
$intakeReadyClock = [Diagnostics.Stopwatch]::StartNew()
while ($intakeReadyClock.Elapsed.TotalSeconds -lt 15) {
    if ($intakeViteProcess -and $intakeViteProcess.HasExited) {
        throw "Intake frontend exited with code $($intakeViteProcess.ExitCode); inspect E: Intake logs."
    }
    try {
        $intakeResponse = Invoke-WebRequest -Uri 'http://127.0.0.1:5176/@vite/client' -TimeoutSec 2 -MaximumRedirection 0 -SkipHttpErrorCheck
        if ($intakeResponse.StatusCode -eq 200) {
            $intakeFrontendReady = $true
            break
        }
    } catch { Write-Verbose 'Waiting for the Intake frontend.' }
    Start-Sleep -Milliseconds 250
}
if (-not $intakeFrontendReady) { throw 'Intake frontend did not become ready within 15 seconds; no native launch attempted.' }
if ($intakeViteProcess) {
    $intakeReadyBindings = @(Get-NetTCPConnection -State Listen -LocalPort 5176 -ErrorAction SilentlyContinue)
    if (-not $intakeReadyBindings -or @($intakeReadyBindings | Where-Object {
        $_.OwningProcess -ne $intakeViteProcess.Id -or $_.LocalAddress -ne '127.0.0.1'
    }).Count -gt 0) {
        throw 'Frontend listener ownership changed during startup; no native launch attempted.'
    }
}
Push-Location (Join-Path $intakeRepo 'apps')
try {
    & $intakeNode $intakeCliPath dev --config src-tauri/tauri.intake.conf.json --no-watch
    if ($LASTEXITCODE -ne 0) { throw "Intake native launcher exited with code $LASTEXITCODE" }
} finally { Pop-Location }
