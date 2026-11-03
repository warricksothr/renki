# Exercise both real install entry points with a staged non-descendant manifest.
# The game must already be offline; neither entry point may stop the controller.
param([string]$Root, [string]$Stage)
. (Join-Path $Root 'common.ps1')
. (Join-Path $Stage 'deployment.ps1') -Root $Root -Stage $Stage -Action Functions
$s = Read-BotSettings
if (Get-Valheim) { throw 'Refusal check requires the game already offline.' }
$manifestPath = Join-Path $Stage 'deployment-manifest.json'
$manifest = Get-Content $manifestPath -Raw | ConvertFrom-Json
if ($manifest.version -like '*-dirty') { throw 'Commit and review the candidate before running this check.' }
$before = Get-DeploymentRuntime $s
if (-not $before.controllerRunning) { throw 'Keep the current panel running for the refusal check.' }
if ($before.version -eq $manifest.version) { throw 'Runtime is already this candidate; use a different reviewed descendant for the refusal check.' }
# First exercise the compatible native read-only check against the real config/atlas.
Test-Deployment $s $Root $Stage
# Remove the installed source from candidate ancestry, never edit its real receipt.
$manifest.ancestors = @($manifest.source)
[IO.File]::WriteAllText($manifestPath, ($manifest | ConvertTo-Json -Depth 12), (New-Object Text.UTF8Encoding($false)))
$data = 'data'
if ($s.ControllerConfig -and (Test-Path $s.ControllerConfig)) {
    $cfg = Get-Content $s.ControllerConfig -Raw | ConvertFrom-Json
    if ($cfg.dataDir) { $data = $cfg.dataDir }
}
if (-not [IO.Path]::IsPathRooted($data)) { $data = Join-Path $Root $data }
function Read-ProtectedHashes {
    $hashes = @{}
    $protected = @((Join-Path $Root 'deployment.json'), $s.ControllerConfig)
    $protected += @(Get-ChildItem $data -File -Recurse -ErrorAction SilentlyContinue | Select-Object -ExpandProperty FullName)
    foreach ($file in $protected) { if ($file -and (Test-Path $file)) { $hashes[$file] = (Get-FileHash -Algorithm SHA256 $file).Hash } }
    return $hashes
}
$hashes = Read-ProtectedHashes
$pids = @(Get-Controller $s | Select-Object -ExpandProperty Id)
foreach ($installer in 'install.ps1','deploy-controller.ps1') {
    $refused = $false
    try { & (Join-Path $Stage $installer) -Root $Root -Stage $Stage }
    catch {
        if ($_.Exception.Message -notlike 'Deployment preflight refused*') { throw }
        $refused = $true
    }
    if (-not $refused) { throw "Unsafe admission: $installer did not refuse." }
    $after = Get-DeploymentRuntime $s
    if ($after.gameRunning -or -not $after.controllerRunning -or $after.version -ne $before.version) { throw 'Runtime state changed during refusal.' }
    if ((@((Get-Controller $s).Id) -join ',') -ne ($pids -join ',')) { throw 'Controller process changed during refusal.' }
    foreach ($name in $before.files.Keys) { if ($after.files[$name] -ne $before.files[$name]) { throw "Installed artifact changed: $name" } }
    $afterHashes = Read-ProtectedHashes
    foreach ($file in $hashes.Keys) { if ($afterHashes[$file] -ne $hashes[$file]) { throw 'A protected data/config file changed during refusal.' } }
    if ($hashes.Count -ne $afterHashes.Count) { throw 'Protected data/config file set changed during refusal.' }
    Write-Host "$installer refused: game offline, controller PID retained, installed/data/config hashes unchanged."
}
