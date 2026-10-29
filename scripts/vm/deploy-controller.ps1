# Install an uploaded controller without closing Valheim.
. (Join-Path $PSScriptRoot 'common.ps1')
$s = Read-BotSettings
if (-not $s.ControllerExe) { throw 'The VM does not run a local controller' }
$next = Join-Path $PSScriptRoot 'controller.next.exe'
if (-not (Test-Path $next)) { throw "Missing upload: $next" }
$controllers = @(Get-Controller $s)
try {
    $controllers | Stop-Process
    $controllers | Wait-Process -Timeout 10
    Copy-Item $next $s.ControllerExe -Force
    Remove-Item $next
} finally {
    Start-ScheduledTask -TaskName RenkiStart
}
