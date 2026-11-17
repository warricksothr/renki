<#
.SYNOPSIS
Read-only facts for `just vm doctor`, as one JSON object on stdout.

doctor.py sends this script over SSH's stdin with $RenkiDir set in front of
it, so nothing is copied to the VM and a VM that has never been deployed to
can still be examined. It changes nothing and prints no token, key, password
or address: only whether each thing exists, runs or answers.
#>
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
if (-not $RenkiDir) { $RenkiDir = 'C:/renki' }

function Try-Get([scriptblock]$block) { try { & $block } catch { $null } }
function Utc($when) { if ($when) { ([datetime]$when).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ') } }

$facts = [ordered]@{ format = 1; now = Utc (Get-Date) }
$facts.boot = Utc (Try-Get { (Get-CimInstance Win32_OperatingSystem).LastBootUpTime })
# The game needs the bot account's desktop: a shell in a session other than 0.
$facts.desktop = [bool](Get-Process explorer -ErrorAction SilentlyContinue | Where-Object { $_.SessionId -ne 0 })

$tasks = [ordered]@{}
foreach ($t in @(Get-ScheduledTask -TaskName 'Renki*' -ErrorAction SilentlyContinue)) { $tasks[$t.TaskName] = [string]$t.State }
$facts.tasks = $tasks

$running = [ordered]@{}
foreach ($name in 'valheim', 'steam', 'obs64', 'mediamtx') {
    $running[$name] = [bool](Get-Process -Name $name -ErrorAction SilentlyContinue)
}

$files = [ordered]@{}
foreach ($name in 'bot.local.psd1', 'controller.local.json', 'secrets.local.json', 'common.ps1', 'start-bot.ps1') {
    $files[$name] = Test-Path (Join-Path $RenkiDir $name)
}
$facts.files = $files

$settings = $null
if ($files['bot.local.psd1']) { $settings = Try-Get { Import-PowerShellDataFile (Join-Path $RenkiDir 'bot.local.psd1') } }
$game = [ordered]@{ exe = $false; bepinex = $false; plugin = $false }
$drives = @('C')
if ($settings -and $settings.ValheimExe) {
    $root = Split-Path $settings.ValheimExe
    $game.exe = Test-Path $settings.ValheimExe
    $game.bepinex = Test-Path (Join-Path $root 'BepInEx\core')
    $game.plugin = (Test-Path (Join-Path $root 'BepInEx\plugins\renki\Renki.Plugin.dll')) -and
        (Test-Path (Join-Path $root 'BepInEx\plugins\renki\Renki.Core.dll'))
    $drive = (Split-Path $settings.ValheimExe -Qualifier).TrimEnd(':')
    if ($drive -and $drives -notcontains $drive) { $drives += $drive }
}
$facts.game = $game

$exe = if ($settings -and $settings.ControllerExe) {
    if ([IO.Path]::IsPathRooted($settings.ControllerExe)) { $settings.ControllerExe } else { Join-Path $RenkiDir $settings.ControllerExe }
}
$controller = [ordered]@{ local = [bool]$exe; exe = $false; running = $false; answering = $false; bot = $null; logWritten = $null }
if ($exe) {
    $controller.exe = Test-Path $exe
    $full = Try-Get { (Resolve-Path $exe).Path }
    $name = [IO.Path]::GetFileNameWithoutExtension($exe)
    $controller.running = [bool](Get-Process -Name $name -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $full })
}
if ($settings -and $settings.ControlUrl) {
    $headers = @{}
    if ($settings.Token) { $headers.Authorization = "Bearer $($settings.Token)" }
    try {
        $bot = Invoke-RestMethod -Uri ($settings.ControlUrl.TrimEnd('/') + '/control/status') -Headers $headers -TimeoutSec 5
        $controller.answering = $true
        $controller.bot = [ordered]@{ connected = [bool]$bot.connected; phase = [string]$bot.phase }
    } catch [System.Net.WebException] {
        # 409 is the controller saying no plugin is connected.
        if ($_.Exception.Response -and [int]$_.Exception.Response.StatusCode -eq 409) {
            $controller.answering = $true
            $controller.bot = [ordered]@{ connected = $false; phase = '' }
        }
    } catch { }
}
$controller.logWritten = Utc (Try-Get { (Get-Item (Join-Path $RenkiDir 'logs\controller.log')).LastWriteTime })
$facts.controller = $controller
$facts.running = $running

$service = Get-Service Tailscale -ErrorAction SilentlyContinue
$facts.tailscale = [ordered]@{
    installed = [bool]$service
    service   = if ($service) { [string]$service.Status } else { '' }
    backend   = [string](Try-Get { (& tailscale status --json 2>$null | Out-String | ConvertFrom-Json).BackendState })
}

# kind says where the panel listens without printing the address.
$panel = [ordered]@{ configured = $false; kind = ''; addressPresent = $false; http = 0; error = '' }
$config = if ($files['controller.local.json']) { Try-Get { Get-Content (Join-Path $RenkiDir 'controller.local.json') -Raw | ConvertFrom-Json } }
if ($config -and $config.panel -and $config.panel.listen) {
    $panel.configured = $true
    $listen = [string]$config.panel.listen
    $split = $listen.LastIndexOf(':')
    $addr = if ($split -ge 0) { $listen.Substring(0, $split).Trim('[', ']') } else { $listen }
    $port = if ($split -ge 0) { $listen.Substring($split + 1) } else { '' }
    $panel.kind = if ($addr -match '^(127\.|localhost$|::1$)') { 'loopback' }
        elseif ($addr -match '^(100\.(6[4-9]|[7-9]\d|1[01]\d|12[0-7])\.|fd7a:)') { 'tailscale' }
        elseif ($addr -in '', '0.0.0.0', '::') { 'any' } else { 'other' }
    $panel.addressPresent = ($panel.kind -in 'any', 'loopback') -or
        [bool](Get-NetIPAddress -IPAddress $addr -ErrorAction SilentlyContinue)
    $probe = if ($panel.kind -eq 'any') { '127.0.0.1' } elseif ($addr.Contains(':')) { "[$addr]" } else { $addr }
    if (-not $port) { $panel.error = 'no port in panel.listen' }
    elseif ($panel.addressPresent) {
        try {
            $panel.http = [int](Invoke-WebRequest -UseBasicParsing -TimeoutSec 8 -Uri "http://${probe}:$port/").StatusCode
        } catch [System.Net.WebException] {
            if ($_.Exception.Response) { $panel.http = [int]$_.Exception.Response.StatusCode }
            else { $panel.error = [string]$_.Exception.Status }
        } catch { $panel.error = 'failed' }
    }
}
$facts.panel = $panel

$receipt = Try-Get { Get-Content (Join-Path $RenkiDir 'deployment.json') -Raw | ConvertFrom-Json }
$facts.deployment = if ($receipt -and $receipt.manifest) {
    [ordered]@{
        version = [string]$receipt.manifest.version; source = [string]$receipt.manifest.source
        branch = [string]$receipt.manifest.branch; actor = [string]$receipt.manifest.actor
        installedAt = Utc $receipt.installedAt
    }
}

$disks = [ordered]@{}
foreach ($d in $drives) {
    $free = Try-Get { (Get-PSDrive $d).Free }
    if ($null -ne $free) { $disks[$d] = [int][math]::Floor($free / 1GB) }
}
$facts.diskFreeGB = $disks

'RENKI-DOCTOR ' + ($facts | ConvertTo-Json -Depth 5 -Compress)
