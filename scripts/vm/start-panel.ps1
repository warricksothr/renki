# The controller needs no GPU, but a scheduled task lets it outlive SSH.
. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings
if (-not $s.ControllerExe) { throw 'The VM does not run a local controller' }
if (-not (Get-Controller $s)) {
    $botTask = Get-ScheduledTask -TaskName RenkiStart
    $action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument (
        '-NoProfile -ExecutionPolicy Bypass -File "' + (Join-Path $PSScriptRoot 'start-bot.ps1') + '" -ControllerOnly -WaitMinutes 0')
    $settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    Register-ScheduledTask -TaskName RenkiController -Action $action -Principal $botTask.Principal -Settings $settings -Force | Out-Null
    Start-ScheduledTask -TaskName RenkiController
}
for ($attempt = 0; $attempt -lt 20; $attempt++) {
    if (Get-BotStatus $s) { Write-Host 'controller and panel ready; game was not started'; exit 0 }
    Start-Sleep -Seconds 1
}
throw 'Controller did not answer within 20 seconds; inspect controller logs'
