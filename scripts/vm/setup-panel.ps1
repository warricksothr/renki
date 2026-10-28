<#
.SYNOPSIS
Turns on renki's control panel on this VM's Tailscale address.

.DESCRIPTION
Run once, elevated: `just vm panel-setup` does it over SSH. It is safe to
run again. It:

  1. finds this machine's Tailscale IPv4 address (in 100.64.0.0/10);
  2. adds the firewall rule RenkiPanel, letting controller.exe take the
     panel port from the tailnet and nowhere else;
  3. sets panel.listen in controller.local.json, and panel.token to a random
     value when there is none. The token is not printed; `just vm
     panel-token` shows it.

Restart the controller afterwards: `just vm restart`.

.PARAMETER Port
The panel's port. Default 8780.
#>
param([int]$Port = 8780)

$ErrorActionPreference = 'Stop'
$principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Run this elevated: the firewall rule needs an administrator.'
}

# 1. The Tailscale address: 100.64.0.0/10 is 100.64.x.x to 100.127.x.x.
$ts = Get-NetIPAddress -AddressFamily IPv4 | Where-Object {
    $o = $_.IPAddress.Split('.')
    $o[0] -eq '100' -and [int]$o[1] -ge 64 -and [int]$o[1] -le 127
} | Select-Object -First 1
if (-not $ts) { throw 'No Tailscale address (100.64.0.0/10) on this machine. Is Tailscale running and logged in?' }
$ip = $ts.IPAddress

# 2. The firewall: this program, this port, the tailnet only. A rule for the
# program also stops Windows asking about it on the desktop, in front of
# the game.
$exe = Join-Path $PSScriptRoot 'controller.exe'
Get-NetFirewallRule -Name 'RenkiPanel' -ErrorAction SilentlyContinue | Remove-NetFirewallRule
New-NetFirewallRule -Name 'RenkiPanel' -DisplayName 'renki control panel' -Direction Inbound -Action Allow `
    -Protocol TCP -LocalPort $Port -RemoteAddress '100.64.0.0/10' -Program $exe -Profile Any | Out-Null
Write-Host "firewall: $exe may take TCP $Port from 100.64.0.0/10"

# 3. The configuration.
$file = Join-Path $PSScriptRoot 'controller.local.json'
if (-not (Test-Path $file)) { throw "No $file. Copy controller.example.json to it first (docs/VM.md)." }
$cfg = Get-Content $file -Raw | ConvertFrom-Json
if (-not $cfg.PSObject.Properties['panel']) {
    $cfg | Add-Member -NotePropertyName panel -NotePropertyValue ([pscustomobject]@{ listen = ''; token = '' })
}
if (-not $cfg.panel.PSObject.Properties['token']) { $cfg.panel | Add-Member -NotePropertyName token -NotePropertyValue '' }
if (-not $cfg.panel.PSObject.Properties['listen']) { $cfg.panel | Add-Member -NotePropertyName listen -NotePropertyValue '' }
$cfg.panel.listen = "${ip}:$Port"
if ([string]::IsNullOrEmpty($cfg.panel.token)) {
    $bytes = New-Object byte[] 24
    [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    $cfg.panel.token = -join ($bytes | ForEach-Object { $_.ToString('x2') })
    Write-Host 'token: a new one was made (just vm panel-token shows it)'
} else {
    Write-Host 'token: kept the existing one'
}
$cfg | ConvertTo-Json -Depth 10 | Set-Content $file -Encoding ascii

Write-Host "panel: http://${ip}:$Port/ once the controller restarts (just vm restart)"
