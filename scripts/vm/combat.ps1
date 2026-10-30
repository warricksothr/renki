# Read or set the plugin's combat switch. A change needs a game restart.
param([ValidateSet('', 'true', 'false')][string]$Value = '')
. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings
$path = Join-Path (Split-Path $s.ValheimExe) 'BepInEx\config\com.warricksothr.renki.cfg'
if (-not (Test-Path $path)) { throw "Missing plugin config: $path" }
$raw = [IO.File]::ReadAllText($path)
$pattern = '(?m)^AllowCombat\s*=\s*(true|false)\s*$'
$match = [regex]::Match($raw, $pattern)
if (-not $match.Success) { throw 'No AllowCombat entry in plugin config' }
if ($Value) {
    $raw = [regex]::Replace($raw, $pattern, ('AllowCombat = ' + $Value))
    [IO.File]::WriteAllText($path, $raw)
    Write-Output "AllowCombat = $Value (applies after restart)"
} else { Write-Output $match.Value.Trim() }
