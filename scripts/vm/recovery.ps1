# Opt-in native startup recovery. Game restart required; default remains off.
param([ValidateSet('', 'true', 'false')][string]$Value = '')
. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings
$path = Join-Path (Split-Path $s.ValheimExe) 'BepInEx\config\com.warricksothr.renki.cfg'
$raw = [IO.File]::ReadAllText($path)
$pattern = '(?m)^RecoveryGuard\s*=\s*(true|false)\s*$'
$match = [regex]::Match($raw, $pattern)
if (-not $Value) { if ($match.Success) { $match.Value.Trim() } else { 'RecoveryGuard = false (default)' }; return }
if ($match.Success) { $raw = [regex]::Replace($raw, $pattern, ('RecoveryGuard = ' + $Value)) }
else {
    $section = [regex]::Match($raw, '(?ms)^\[Safety\].*?(?=^\[|\z)')
    if (-not $section.Success) { throw 'Missing Safety section' }
    $raw = $raw.Insert($section.Index + $section.Length, "RecoveryGuard = $Value`r`n`r`n")
}
[IO.File]::WriteAllText($path, $raw)
"RecoveryGuard = $Value (applies after restart)"
