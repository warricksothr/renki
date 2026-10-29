<#
.SYNOPSIS
Call the live panel API without copying its token into commands or logs.
#>
param(
    [Parameter(Mandatory=$true)][ValidateSet('GET', 'POST')][string]$Method,
    [Parameter(Mandatory=$true)][string]$ApiPath,
    [string]$BodyBase64 = ''
)
$ErrorActionPreference = 'Stop'
if (-not $ApiPath.StartsWith('/api/')) { throw 'Use an /api/ endpoint' }
$cfg = Get-Content (Join-Path $PSScriptRoot 'controller.local.json') -Raw | ConvertFrom-Json
if (-not $cfg.panel.listen) { throw 'The panel is off: run just vm panel-setup' }
$token = [Environment]::GetEnvironmentVariable('RENKI_PANEL_TOKEN', 'User')
if (-not $token) { $token = $cfg.panel.token }
$request = @{
    Method = $Method
    Uri = 'http://' + $cfg.panel.listen + $ApiPath
    Headers = @{ Authorization = 'Bearer ' + $token; 'X-Renki-Act' = 'vm-tooling' }
    TimeoutSec = 20
}
if ($BodyBase64) {
    $request.Body = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($BodyBase64))
    $request.ContentType = 'application/json; charset=utf-8'
}
Invoke-RestMethod @request | ConvertTo-Json -Depth 30 -Compress
