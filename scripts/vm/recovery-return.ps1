# Arm a one-shot native home return, or clear an unused order. Game must be offline.
param([ValidateSet('arm', 'clear')][string]$Action = 'arm', [string]$WorldBase64 = '')
. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings
$path = Join-Path (Split-Path $s.ValheimExe) 'BepInEx\config\com.warricksothr.renki.cfg'
$raw = [IO.File]::ReadAllText($path)
$value = ''
if ($Action -eq 'arm') {
    if (Get-Valheim) { throw 'Recovery return can only be armed with the game offline' }
    if ($raw -notmatch '(?m)^RecoveryGuard\s*=\s*true\s*$') { throw 'Native RecoveryGuard must stay enabled' }
    $status = (& (Join-Path $PSScriptRoot 'panel-request.ps1') -Method GET -ApiPath '/api/status') | ConvertFrom-Json
    if ($status.connected -or $status.task) { throw 'Bot must be offline and unowned before arming recovery' }
    $world = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($WorldBase64))
    if (-not $world) { throw 'Name the known world for offline recovery' }
    $controllerConfig = Get-Content $s.ControllerConfig -Raw | ConvertFrom-Json
    $dataDir = if ($controllerConfig.dataDir) { [string]$controllerConfig.dataDir } else { 'data' }
    if (-not [IO.Path]::IsPathRooted($dataDir)) { $dataDir = Join-Path $PSScriptRoot $dataDir }
    $memory = Get-Content (Join-Path $dataDir 'memory.json') -Raw | ConvertFrom-Json
    $saved = @($memory.homes.PSObject.Properties | Where-Object { $_.Name -ceq $world })
    if ($saved.Count -ne 1) { throw 'No unique saved home for that world' }
    $home = $saved[0].Value
    if ($home.pos.Count -ne 3) { throw 'No pinned controller home' }
    foreach ($coordinate in $home.pos) {
        $number = [double]$coordinate
        if ([double]::IsNaN($number) -or [double]::IsInfinity($number)) { throw 'Invalid home coordinate' }
    }
    $value = @{ World = $world; Pos = @($home.pos); Expires = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds() + 300 } | ConvertTo-Json -Compress
}
$pattern = '(?m)^RecoveryReturn\s*=.*$'
if ([regex]::IsMatch($raw, $pattern)) {
    $raw = [regex]::Replace($raw, $pattern, [System.Text.RegularExpressions.MatchEvaluator]{ param($match) 'RecoveryReturn = ' + $value })
} else {
    $section = [regex]::Match($raw, '(?ms)^\[Safety\].*?(?=^\[|\z)')
    if (-not $section.Success) { throw 'Missing Safety section' }
    $raw = $raw.Insert($section.Index + $section.Length, "RecoveryReturn = $value`r`n`r`n")
}
[IO.File]::WriteAllText($path, $raw)
if ($Action -eq 'arm') { $value } else { 'RecoveryReturn cleared' }
