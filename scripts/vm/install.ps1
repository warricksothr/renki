# Install a compatible, staged package. The recipe checks before stopping;
# repeat the check immediately before replacing files to detect intervening drift.
param([string]$Root = $PSScriptRoot, [string]$Stage = $PSScriptRoot)
. (Join-Path $Root 'common.ps1')
. (Join-Path $Stage 'deployment.ps1') -Root $Root -Stage $Stage -Action Functions
$s = Read-BotSettings
Test-Deployment $s $Root $Stage
if (Get-Valheim) { throw 'Valheim is running; stop the bot first (RenkiStop).' }
if (Get-Controller $s) { throw 'The controller is running; stop the panel before installation.' }
$manifest = Get-Content (Join-Path $Stage 'deployment-manifest.json') -Raw | ConvertFrom-Json
if (@($manifest.files.PSObject.Properties).Count -ne 3) { throw 'Full install requires controller, Core and plugin artifacts.' }
$game = Split-Path $s.ValheimExe
$plugins = Join-Path $game 'BepInEx\plugins'
if (-not (Test-Path $plugins)) { throw 'Install BepInExPack_Valheim into the game folder first.' }
$dest = Join-Path $plugins 'renki'
New-Item -ItemType Directory -Force -Path $dest | Out-Null
foreach ($name in 'Renki.Core.dll','Renki.Plugin.dll') {
    Copy-Item (Join-Path (Join-Path $Stage 'plugin') $name) (Join-Path $dest $name) -Force
}
Copy-Item (Join-Path $Stage 'controller.exe') $s.ControllerExe -Force
# Public package tooling/docs only. Settings, secrets and saves are never staged.
foreach ($source in Get-ChildItem $Stage -File) {
    $publicFile = $source.Extension -in '.ps1','.md' -or $source.Name -in 'controller.example.json','bot.example.psd1','SHA256SUMS','deployment-manifest.json','renki-agent.pub'
    if (-not $publicFile) { continue }
    Copy-Item $source.FullName (Join-Path $Root $source.Name) -Force
}
Write-DeploymentReceipt $s $Root $Stage
