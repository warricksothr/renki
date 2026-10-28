<#
.SYNOPSIS
Starts the renki bot: the controller (when it runs on this VM), then Valheim,
which joins the server with the bot's character by itself.

.DESCRIPTION
Valheim's own `-joinserverwithcharacter host:port character` argument picks
the character and joins with no clicks, and `-password` answers the server's
password prompt. The renki plugin dials the controller once the game is up.
Settings come from bot.local.psd1 beside this script.

.PARAMETER WaitMinutes
How long to wait for the bot to report itself in the world. 0 returns at once.
#>
param([double]$WaitMinutes = 5)

. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings
$logs = Join-Path $PSScriptRoot 'logs'
New-Item -ItemType Directory -Force -Path $logs | Out-Null

if ($s.ControllerExe) {
    if (Get-Controller $s) {
        Write-Host 'controller: already running'
    } else {
        $ctlArgs = @()
        if ($s.ControllerConfig -and (Test-Path $s.ControllerConfig)) { $ctlArgs += @('-config', $s.ControllerConfig) }
        Start-Process -FilePath $s.ControllerExe -ArgumentList $ctlArgs -WorkingDirectory $PSScriptRoot `
            -WindowStyle Hidden -RedirectStandardError (Join-Path $logs 'controller.log') `
            -RedirectStandardOutput (Join-Path $logs 'controller.out.log') | Out-Null
        Write-Host "controller: started ($($s.ControllerExe))"
        Start-Sleep -Seconds 1
    }
}

if (Get-Valheim) {
    Write-Host 'valheim: already running'
} else {
    # The game only honours -joinserverwithcharacter when another argument
    # follows the character, so there is always one.
    $game = @('-joinserverwithcharacter', $s.Server, $s.Character)
    if ($s.Password) { $game += @('-password', $s.Password) } else { $game += '-renki' }
    $game += $s.ExtraArgs
    Start-Process -FilePath $s.ValheimExe -ArgumentList $game -WorkingDirectory (Split-Path $s.ValheimExe) | Out-Null
    Write-Host "valheim: started, joining $($s.Server) as $($s.Character)"
}

if ($WaitMinutes -le 0) { return }
$deadline = (Get-Date).AddMinutes($WaitMinutes)
while ((Get-Date) -lt $deadline) {
    $st = Get-BotStatus $s
    if ($st -and $st.connected -and $st.phase -eq 'in_world') {
        Write-Host "bot: in the world as $($st.character)"
        return
    }
    Start-Sleep -Seconds 3
}
Write-Warning "bot: not in the world after $WaitMinutes minutes; see $logs and BepInEx\LogOutput.log"
exit 1
