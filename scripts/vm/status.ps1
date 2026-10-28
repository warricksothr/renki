<#
.SYNOPSIS
What the remote tooling sees: whether the game and controller run, what the
controller says about the bot, and the tail of both logs.

.PARAMETER Lines
How many lines of each log to show; 0 shows none.
#>
param([int]$Lines = 0)

. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings

$v = Get-Valheim
$c = Get-Controller $s
Write-Host ("valheim:    " + $(if ($v) { "running (pid $($v.Id -join ','), session $($v.SessionId -join ','))" } else { 'not running' }))
Write-Host ("controller: " + $(if ($c) { "running (pid $($c.Id))" } elseif ($s.ControllerExe) { 'not running' } else { 'elsewhere' }))
$st = Get-BotStatus $s
Write-Host ("bot:        " + $(if ($st) { $st | ConvertTo-Json -Compress } else { 'controller not answering' }))

if ($Lines -gt 0) {
    $bepinex = Join-Path (Split-Path $s.ValheimExe) 'BepInEx\LogOutput.log'
    foreach ($log in @($bepinex, (Join-Path $PSScriptRoot 'logs\controller.log'))) {
        Write-Host "`n--- $log"
        if (Test-Path $log) {
            Get-Content $log -Tail $Lines | ForEach-Object {
                if ($log -eq $bepinex -and $_ -notmatch 'renki|Error|Exception') { return }
                $_
            }
        } else {
            Write-Host '(none)'
        }
    }
}
