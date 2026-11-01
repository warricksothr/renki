<#
.SYNOPSIS
Call the live panel API without copying its token into commands or logs.
#>
param(
    [Parameter(Mandatory=$true)][ValidateSet('GET', 'POST')][string]$Method,
    [Parameter(Mandatory=$true)][string]$ApiPath,
    [string]$BodyBase64 = '',
    [switch]$IncludeInspection
)
$ErrorActionPreference = 'Stop'
# Windows PowerShell defaults to the console code page on SSH stdout and can
# decode JSON without a charset as Latin-1. The panel's JSON is always UTF-8.
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
if (-not $ApiPath.StartsWith('/api/')) { throw 'Use an /api/ endpoint' }
if ($IncludeInspection) {
    if ($Method -ne 'GET' -or $ApiPath -ne '/api/status' -or $BodyBase64) {
        throw 'Inspection bundle is read-only and requires GET /api/status without a body'
    }
    # One SSH process and one helper upload, with no concurrent SCP overwrite.
    # These remain two observations; the caller must check scope and freshness.
    $status = & $PSCommandPath -Method GET -ApiPath '/api/status' | ConvertFrom-Json
    $inspection = & $PSCommandPath -Method GET -ApiPath '/api/inspection' | ConvertFrom-Json
    @{ status = $status; inspection = $inspection } | ConvertTo-Json -Depth 30 -Compress
    return
}
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
$response = Invoke-WebRequest -UseBasicParsing @request
$response.RawContentStream.Position = 0
$reader = [IO.StreamReader]::new($response.RawContentStream, [Text.Encoding]::UTF8)
try {
    $reader.ReadToEnd() | ConvertFrom-Json | ConvertTo-Json -Depth 30 -Compress
} finally {
    $reader.Dispose()
}
