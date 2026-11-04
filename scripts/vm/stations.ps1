# Read or set ordinary fireplace permission. Changes apply at next game start.
param([ValidateSet('', 'true', 'false')][string]$Value = '')
. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings
$path = Join-Path (Split-Path $s.ValheimExe) 'BepInEx\config\com.warricksothr.renki.cfg'
if (-not (Test-Path $path)) { throw "Missing plugin config: $path" }
$raw = [IO.File]::ReadAllText($path)
# Restrict edits to the real Safety section; preserve other settings verbatim.
$section = [regex]::Match($raw, '(?ms)^\[Safety\][^\r\n]*(?:\r?\n|$)(?<body>.*?)(?=^\[|\z)')
if (-not $section.Success) { throw 'No Safety section in plugin config' }
$body = $section.Groups['body'].Value
$pattern = '(?m)^AllowStations\s*=[ \t]*(true|false)[ \t]*\r?$'
$entries = [regex]::Matches($body, $pattern)
if ([regex]::Matches($body, '(?m)^[ \t]*AllowStations[ \t]*=').Count -ne $entries.Count) { throw 'Invalid AllowStations entry' }
if ($entries.Count -gt 1) { throw 'Duplicate AllowStations entries' }
if ($Value) {
    if ($entries.Count -eq 1) { $body = [regex]::Replace($body, $pattern, ('AllowStations = ' + $Value)) }
    else { $body = "AllowStations = $Value`r`n" + $body }
    $group = $section.Groups['body']
    $raw = $raw.Substring(0, $group.Index) + $body + $raw.Substring($group.Index + $group.Length)
    [IO.File]::WriteAllText($path, $raw)
    Write-Output "AllowStations = $Value (applies after restart)"
} elseif ($entries.Count -eq 1) { Write-Output $entries[0].Value.Trim() }
else { Write-Output 'AllowStations = false (default; no entry yet)' }
