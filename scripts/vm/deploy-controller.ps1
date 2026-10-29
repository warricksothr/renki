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
    # Windows may retain the image lock briefly after the process exits.
    # Retry only that transient I/O failure; keep the uploaded file on error.
    for ($attempt = 0; $attempt -lt 40; $attempt++) {
        try {
            Copy-Item $next $s.ControllerExe -Force -ErrorAction Stop
            break
        } catch [System.IO.IOException] {
            if ($attempt -eq 39) { throw }
            Start-Sleep -Milliseconds 250
        }
    }
    Remove-Item $next
} finally {
    Start-ScheduledTask -TaskName RenkiStart
}
