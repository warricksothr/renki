# Refresh the bridge welcome without closing the game.
. (Join-Path $PSScriptRoot 'common.ps1')
Get-Controller (Read-BotSettings) | Stop-Process
Start-ScheduledTask -TaskName RenkiStart
