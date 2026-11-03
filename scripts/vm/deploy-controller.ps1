# Validate the staged controller before stopping anything; retain game state.
param([string]$Root = $PSScriptRoot, [string]$Stage = $PSScriptRoot)
. (Join-Path $Root 'common.ps1')
. (Join-Path $Stage 'deployment.ps1') -Root $Root -Stage $Stage -Action Functions
$s = Read-BotSettings
if (-not $s.ControllerExe) { throw 'The VM does not run a local controller' }
Test-Deployment $s $Root $Stage
$next = Join-Path $Stage 'controller.exe'
$controllers = @(Get-Controller $s)
$gameWasRunning = [bool](Get-Valheim)
try {
    $controllers | Stop-Process
    $controllers | Wait-Process -Timeout 10
    for ($attempt = 0; $attempt -lt 40; $attempt++) {
        try {
            Copy-Item $next $s.ControllerExe -Force -ErrorAction Stop
            break
        } catch [System.IO.IOException] {
            if ($attempt -eq 39) { throw }
            Start-Sleep -Milliseconds 250
        }
    }
    Write-DeploymentReceipt $s $Root $Stage
} finally {
    # A controller that was off stays off. A panel-only runtime stays offline.
    if ($controllers.Count -gt 0) { & (Join-Path $Root 'start-panel.ps1') }
}
