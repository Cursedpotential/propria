# Requires: Windows PowerShell / authenticated, owner-managed CF_MCP_CLIENT_TOKEN.
# Purpose: expose all EXISTING n8n virtual-server tools through the EXISTING
#          OAuth-enabled Propria MCP server used by ChatGPT, Claude and Codex.
# No new gateway, n8n instance, OAuth client, or plaintext credentials.
# Inputs: live ContextForge server/tool registry; outputs: counts, rollback receipt.
# Effects: updates only the Propria virtual server's associated tool IDs.
# Byline: GPT-6 · 2026-10-17.
param(
    [string]$ContextForgeBase = "https://mcp.mitechconsult.com",
    [switch]$DryRun
)
$ErrorActionPreference = 'Stop'
if (-not $env:CF_MCP_CLIENT_TOKEN) {
    throw "A ContextForge client credential is required in the current environment."
}
$base = $ContextForgeBase.TrimEnd('/')
if (-not $base.StartsWith('https://')) {
    throw "ContextForge management must use HTTPS."
}
$headers = @{ Authorization = 'Bearer ' + $env:CF_MCP_CLIENT_TOKEN }
$registered = Invoke-RestMethod -Uri "$base/servers?include_pagination=false" -Headers $headers -TimeoutSec 15
$propria = @($registered | Where-Object { $_.name -eq 'propria' })
$n8n = @($registered | Where-Object { $_.name -eq 'n8n' })
if ($propria.Count -ne 1 -or $n8n.Count -ne 1) {
    throw "Expected exactly one Propria server and one n8n server."
}
$targetId = $propria[0].id
$sourceId = $n8n[0].id
$p = Invoke-RestMethod -Uri "$base/servers/$targetId" -Headers $headers -TimeoutSec 15
$n = Invoke-RestMethod -Uri "$base/servers/$sourceId" -Headers $headers -TimeoutSec 15
if (-not $p.enabled -or -not $p.oauthEnabled -or -not $n.enabled -or -not $n.oauthEnabled) {
    throw "Both existing virtual servers must be active and OAuth-enabled."
}
$old = @($p.associatedToolIds)
$incoming = @($n.associatedToolIds)
if ($old.Count -lt 1 -or $incoming.Count -lt 1 -or
    $incoming.Count -ne @($n.associatedTools).Count) {
    throw "Virtual server tool identities are missing or mismatched."
}
$foreign = @($n.associatedTools | Where-Object { [string]$_ -notlike 'n8n-*' })
if ($foreign.Count -gt 0) {
    throw "The source virtual server contains a tool outside the n8n prefix."
}
$missing = @($incoming | Where-Object { $_ -notin $old })
$desired = @($old + $missing | Select-Object -Unique)
if ($desired.Count -ne ($old.Count + $missing.Count)) {
    throw "Unexpected duplicate tool identity; no changes made."
}
Write-Output ("Before: Propria {0}, n8n source {1}, missing {2}" -f $old.Count, $incoming.Count, $missing.Count)
if ($missing.Count -eq 0) {
    Write-Output ("ALREADY FEDERATED: {0} tools; no mutation required." -f $old.Count)
    exit 0
}
if ($DryRun) {
    Write-Output ("DRY RUN: would associate {0} tools with existing Propria server." -f $desired.Count)
    exit 0
}
$backup = Join-Path $HOME ('.secrets/contextforge-propria-tool-membership-' + (Get-Date -Format 'yyyyMMdd-HHmmss') + '.json')
$before = @{ serverId = $targetId; serverName = $p.name; associatedTools = $old; capturedAt = (Get-Date).ToUniversalTime().ToString('o') }
$before | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $backup -Encoding UTF8
$body = @{ associatedTools = $desired } | ConvertTo-Json -Compress -Depth 6
$didUpdate = $false
try {
    Invoke-RestMethod -Method Put -Uri "$base/servers/$targetId" -Headers $headers -Body $body -ContentType 'application/json' -TimeoutSec 20 | Out-Null
    $didUpdate = $true
    $verify = Invoke-RestMethod -Uri "$base/servers/$targetId" -Headers $headers -TimeoutSec 15
    $got = @($verify.associatedToolIds)
    if ($got.Count -ne $desired.Count -or @($desired | Where-Object { $_ -notin $got }).Count -ne 0 -or
        -not $verify.oauthEnabled -or -not $verify.enabled) {
        throw "Tool membership or OAuth changed unexpectedly."
    }
    Write-Output ("SUCCESS: OAuth-enabled Propria MCP now has {0} tools, including all {1} n8n tools." -f $got.Count, $incoming.Count)
    Write-Output ("Rollback membership stored locally: {0}" -f $backup)
} catch {
    Write-Output "Change did not verify; attempting to restore original membership."
    if ($didUpdate) {
        try {
            $restore = @{ associatedTools = $old } | ConvertTo-Json -Compress -Depth 6
            Invoke-RestMethod -Method Put -Uri "$base/servers/$targetId" -Headers $headers -Body $restore -ContentType 'application/json' -TimeoutSec 20 | Out-Null
            Write-Output "Rollback applied."
        } catch { Write-Output "Rollback failed; restore from the local membership backup." }
    }
    throw "n8n federation could not be verified."
}
