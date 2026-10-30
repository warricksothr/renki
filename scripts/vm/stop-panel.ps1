# Release the controller executable before installing a full VM package.
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')
$controllers = @(Get-Controller (Read-BotSettings))
if ($controllers.Count -eq 0) {
    Write-Host 'controller: not running'
    exit 0
}
$controllers | Stop-Process
$controllers | Wait-Process -Timeout 10
# Windows may retain process/image records briefly after Wait-Process returns.
for ($attempt = 0; $attempt -lt 40; $attempt++) {
    if (-not (Get-Controller (Read-BotSettings))) {
        # Process disappearance does not guarantee its executable's image
        # lock has been released. Probe write access without changing bytes
        # before scp tries to replace it during a full package installation.
        try {
            $stream = [IO.File]::Open((Read-BotSettings).ControllerExe,
                [IO.FileMode]::Open, [IO.FileAccess]::Write, [IO.FileShare]::Read)
            $stream.Dispose()
            Write-Host 'controller: stopped and executable released'
            exit 0
        } catch [System.IO.IOException] {
            # Retry the transient image lock, with the same bounded wait.
        }
    }
    Start-Sleep -Milliseconds 250
}
throw 'Controller process or executable lock remained after the exit wait'
