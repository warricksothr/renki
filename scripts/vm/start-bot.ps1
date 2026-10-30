<#
.SYNOPSIS
Starts the renki bot: the controller (when it runs on this VM), then Valheim,
which joins the server with the bot's character by itself.

.DESCRIPTION
Valheim's own `-joinserverwithcharacter host:port character` argument picks
the character and joins with no clicks, and `-password` answers the server's
password prompt. The renki plugin dials the controller once the game is up.
Settings come from bot.local.psd1 beside this script.

.PARAMETER ControllerOnly
Start only the controller/panel, without joining the game.

.PARAMETER WaitMinutes
How long to wait for the bot to report itself in the world. 0 returns at once.
#>
param([double]$WaitMinutes = 5, [switch]$ControllerOnly)

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
        # The controller inherits this environment: hand it its secrets,
        # such as the LLM key its configuration names.
        $secrets = Read-Secrets
        foreach ($k in $secrets.Keys) { Set-Item -Path "env:$k" -Value $secrets[$k] }
        if ($secrets.Count) { Write-Host "controller: secrets $($secrets.Keys -join ', ')" }
        Start-Process -FilePath $s.ControllerExe -ArgumentList $ctlArgs -WorkingDirectory $PSScriptRoot `
            -WindowStyle Hidden -RedirectStandardError (Join-Path $logs 'controller.log') `
            -RedirectStandardOutput (Join-Path $logs 'controller.out.log') | Out-Null
        # Only the controller: the game, started below, need not see them.
        foreach ($k in $secrets.Keys) { Remove-Item -Path "env:$k" -ErrorAction SilentlyContinue }
        Write-Host "controller: started ($($s.ControllerExe))"
        Start-Sleep -Seconds 1
    }
}

if ($ControllerOnly) {
    if (-not $s.ControllerExe) { throw 'The VM does not run a local controller' }
    return
}

if (-not (Test-Path -LiteralPath $s.ValheimExe -PathType Leaf)) {
    throw "ValheimExe in bot.local.psd1 is not a file: [$($s.ValheimExe)]"
}

if (Get-Valheim) {
    Write-Host 'valheim: already running'
} else {
    # The game joins from the command line before its matchmaking (and DNS)
    # is up, so it can only join an IP address: resolve a host name here.
    $server = $s.Server
    if ($server -match '^(?<host>.+):(?<port>\d+)$') {
        $hostName = $Matches.host
        $port = $Matches.port
    } else {
        $hostName = $server
        $port = '2456'
    }
    $ip = $null
    if (-not [Net.IPAddress]::TryParse($hostName, [ref]$ip)) {
        $ip = [Net.Dns]::GetHostAddresses($hostName) |
            Where-Object { $_.AddressFamily -eq 'InterNetwork' } | Select-Object -First 1
        if (-not $ip) { throw "cannot resolve $hostName to an IPv4 address" }
        Write-Host "server: $hostName is $ip"
    }
    # The game only honours -joinserverwithcharacter when another argument
    # follows the character, so there is always one.
    $game = @('-joinserverwithcharacter', "$($ip):$port", $s.Character)
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
