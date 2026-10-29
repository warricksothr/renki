# Shared by start-bot.ps1 and stop-bot.ps1. Windows PowerShell 5.1 or later.

$ErrorActionPreference = 'Stop'

# The controller's secrets from secrets.local.json (set-secret.ps1), as a
# hashtable of environment variable names to values.
function Read-Secrets {
    $file = Join-Path $PSScriptRoot 'secrets.local.json'
    $out = @{}
    # Blank, as a set-secret run that failed after creating it leaves it.
    $raw = if (Test-Path $file) { Get-Content $file -Raw }
    if ($raw -and $raw.Trim()) {
        $json = $raw | ConvertFrom-Json
        foreach ($p in $json.PSObject.Properties) { $out[$p.Name] = [string]$p.Value }
    }
    return $out
}

function Read-BotSettings {
    $path = Join-Path $PSScriptRoot 'bot.local.psd1'
    if (-not (Test-Path $path)) {
        throw "No $path. Copy bot.example.psd1 to bot.local.psd1 and fill it in."
    }
    $s = Import-PowerShellDataFile $path
    if ($env:RENKI_TOKEN) { $s.Token = $env:RENKI_TOKEN }
    foreach ($key in 'ControllerExe', 'ControllerConfig') {
        if ($s[$key] -and -not [IO.Path]::IsPathRooted($s[$key])) {
            $s[$key] = Join-Path $PSScriptRoot $s[$key]
        }
    }
    return $s
}

function Invoke-Control($s, [string]$method, [string]$path) {
    $headers = @{}
    if ($s.Token) { $headers.Authorization = "Bearer $($s.Token)" }
    try {
        return Invoke-RestMethod -Method $method -Uri ($s.ControlUrl.TrimEnd('/') + $path) -Headers $headers -TimeoutSec 5
    } catch [System.Net.WebException] {
        $code = [int]$_.Exception.Response.StatusCode
        if ($code -eq 409) { return $null }  # no plugin connected
        throw
    }
}

function Get-BotStatus($s) {
    try { return Invoke-Control $s 'GET' '/control/status' } catch { return $null }
}

function Get-Valheim { Get-Process -Name 'valheim' -ErrorAction SilentlyContinue }

function Get-Controller($s) {
    if (-not $s.ControllerExe) { return $null }
    $name = [IO.Path]::GetFileNameWithoutExtension($s.ControllerExe)
    Get-Process -Name $name -ErrorAction SilentlyContinue |
        Where-Object { $_.Path -eq (Resolve-Path $s.ControllerExe).Path }
}
