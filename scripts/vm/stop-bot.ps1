<#
.SYNOPSIS
Stops the renki bot cleanly: logs the character out (which saves it), closes
Valheim, then stops the controller when it runs on this VM.

.PARAMETER WaitSeconds
How long to wait for the logout before closing the game anyway.
#>
param([int]$WaitSeconds = 60)

. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings

$valheim = Get-Valheim
if ($valheim) {
    $st = Get-BotStatus $s
    if ($st -and $st.connected -and $st.phase -in @('in_world', 'logging_out')) {
        if ($st.phase -eq 'in_world') {
            Invoke-Control $s 'POST' '/control/logout' | Out-Null
        }
        Write-Host 'bot: logging out'
        $deadline = (Get-Date).AddSeconds($WaitSeconds)
        do {
            Start-Sleep -Seconds 2
            $st = Get-BotStatus $s
        } while ((Get-Date) -lt $deadline -and $st -and $st.connected -and $st.phase -ne 'menu')
        if ($st -and $st.connected -and $st.phase -ne 'menu') {
            Write-Warning "bot: still '$($st.phase)' after $WaitSeconds s; closing the game anyway"
        } else {
            Write-Host 'bot: logged out'
        }
    } else {
        Write-Warning 'bot: not reported in the world; closing the game without a logout'
    }
    foreach ($p in $valheim) { $p.CloseMainWindow() | Out-Null }
    $valheim | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue
    if (Get-Valheim) { Get-Valheim | Stop-Process -Force }
    Write-Host 'valheim: closed'
} else {
    Write-Host 'valheim: not running'
}

$controller = Get-Controller $s
if ($controller) {
    $controller | Stop-Process
    Write-Host 'controller: stopped'
}
