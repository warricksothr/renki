# Read-only compatibility checks run from staging, before runtime stop/replacement.
param([string]$Root = $PSScriptRoot, [string]$Stage = $PSScriptRoot,
      [ValidateSet('Check','Record','Functions')][string]$Action = 'Check')
$ErrorActionPreference = 'Stop'

function Get-DeploymentRuntime($s) {
    $files = @{}
    if ($s.ControllerExe -and (Test-Path $s.ControllerExe)) {
        $files['controller.exe'] = (Get-FileHash -Algorithm SHA256 $s.ControllerExe).Hash.ToLowerInvariant()
    }
    $pluginDir = Join-Path (Split-Path $s.ValheimExe) 'BepInEx\plugins\renki'
    foreach ($name in 'Renki.Core.dll','Renki.Plugin.dll') {
        $path = Join-Path $pluginDir $name
        if (Test-Path $path) { $files["plugin/$name"] = (Get-FileHash -Algorithm SHA256 $path).Hash.ToLowerInvariant() }
    }
    $version = ''
    if ($files.ContainsKey('controller.exe')) {
        $output = & $s.ControllerExe -version
        if ($LASTEXITCODE -ne 0 -or $output -notmatch '^renki controller (.+)$') { throw 'Cannot identify the installed controller version.' }
        $version = $Matches[1]
    }
    return @{gameRunning=[bool](Get-Valheim); controllerRunning=[bool](Get-Controller $s); version=$version; files=$files}
}

function Test-Deployment($s, [string]$rootDir, [string]$stageDir) {
    $runtime = Get-DeploymentRuntime $s
    $snapshot = Join-Path $stageDir 'runtime-check.json'
    $runtime | ConvertTo-Json -Depth 10 | Set-Content -Encoding ASCII $snapshot
    $candidate = Join-Path $stageDir 'controller.exe'
    $arguments = @('-deployment-preflight', (Join-Path $stageDir 'deployment-manifest.json'),
        '-deployment-installed', (Join-Path $rootDir 'deployment.json'), '-deployment-runtime', $snapshot)
    if ($s.ControllerConfig) { $arguments += @('-config', $s.ControllerConfig) }
    $previousSecrets = @{}
    foreach ($entry in (Read-Secrets).GetEnumerator()) {
        $previousSecrets[$entry.Key] = [Environment]::GetEnvironmentVariable($entry.Key, 'Process')
        [Environment]::SetEnvironmentVariable($entry.Key, $entry.Value, 'Process')
    }
    Push-Location $rootDir
    try {
        & $candidate @arguments
        if ($LASTEXITCODE -ne 0) { throw 'Deployment preflight refused; the installed runtime and durable data have been retained.' }
    } finally {
        Pop-Location
        foreach ($name in $previousSecrets.Keys) { [Environment]::SetEnvironmentVariable($name, $previousSecrets[$name], 'Process') }
    }
    $previous = Join-Path $stageDir 'runtime-before.json'
    if (-not (Test-Path $previous)) { Copy-Item $snapshot $previous }
}

function Write-DeploymentReceipt($s, [string]$rootDir, [string]$stageDir) {
    $manifest = Get-Content (Join-Path $stageDir 'deployment-manifest.json') -Raw | ConvertFrom-Json
    $runtime = Get-DeploymentRuntime $s
    foreach ($entry in $manifest.files.PSObject.Properties) {
        if ($runtime.files[$entry.Name] -ne $entry.Value) { throw "Installed checksum mismatch: $($entry.Name)" }
    }
    $receipt = @{manifest=$manifest; installedFiles=$runtime.files;
        previous=(Get-Content (Join-Path $stageDir 'runtime-before.json') -Raw | ConvertFrom-Json);
        installedAt=[DateTime]::UtcNow.ToString('o')}
    $next = Join-Path $rootDir 'deployment.json.next'
    $receipt | ConvertTo-Json -Depth 12 | Set-Content -Encoding ASCII $next
    Move-Item $next (Join-Path $rootDir 'deployment.json') -Force
    Write-Host 'Installed checksums and deployment receipt verified; durable data retained.'
}

if ($Action -ne 'Functions') {
    . (Join-Path $Root 'common.ps1')
    $s = Read-BotSettings
    if ($Action -eq 'Check') { Test-Deployment $s $Root $Stage }
    else { Write-DeploymentReceipt $s $Root $Stage }
}
