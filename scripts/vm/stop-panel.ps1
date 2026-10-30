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
        Write-Host 'controller: stopped and exit verified'
        exit 0
    }
    Start-Sleep -Milliseconds 250
}
throw 'Controller is still running after the exit wait' 
