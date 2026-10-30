param([ValidateSet('setup','status','adapter','start','stop','relay','local-start','local-stop','snapshot','game-snapshot','enable','https','codec','beacon','slate','trial-source-loss','trial-watchdog')][string]$Action = 'status')
$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$obsRoot = Join-Path $root 'obs'
$private = Join-Path $root 'private'
$obsExe = Join-Path $obsRoot 'bin/64bit/obs64.exe'

function Save-Json($Path, $Value) {
    [IO.File]::WriteAllText($Path, ($Value | ConvertTo-Json -Depth 20), (New-Object Text.UTF8Encoding $false))
}
function New-Password {
    $bytes = New-Object byte[] 32
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($bytes); [Convert]::ToBase64String($bytes).Replace('/','_').Replace('+','-').TrimEnd('=') } finally { $rng.Dispose() }
}
function Task($Name, $Exe, $Arguments, $Working) {
    $principal = (Get-ScheduledTask -TaskName RenkiStart).Principal
    $action = New-ScheduledTaskAction -Execute $Exe -Argument $Arguments -WorkingDirectory $Working
    $settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    Register-ScheduledTask -TaskName $Name -Principal $principal -Action $action -Settings $settings -Force | Out-Null
}

if ($Action -eq 'setup') {
    if (Get-Process obs64 -ErrorAction SilentlyContinue) { throw 'Stop the dedicated OBS instance before setup' }
    foreach ($item in @(@('obs.zip','0d9436451c2d772b33428e3fdae9776f071895c335de17283176306fb72651d3'), @('mediamtx.zip','b2f5246b07cb72961df03b8d5e8cad43dc657bcf3f49a87bb031bdf22721af74'))) {
        if ((Get-FileHash (Join-Path $root $item[0]) -Algorithm SHA256).Hash.ToLowerInvariant() -ne $item[1]) { throw 'Release checksum mismatch' }
    }
    New-Item -ItemType Directory -Force $private | Out-Null
    $account = [Security.Principal.WindowsIdentity]::GetCurrent().Name
    & icacls $private /inheritance:r /grant:r "${account}:(OI)(CI)F" 'SYSTEM:(OI)(CI)F' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Unable to secure private video configuration' }
    if (-not (Test-Path $obsExe)) { Expand-Archive (Join-Path $root 'obs.zip') $obsRoot }
    if (-not (Test-Path (Join-Path $root 'mediamtx.exe'))) { Expand-Archive (Join-Path $root 'mediamtx.zip') $root -Force }
    New-Item -ItemType File -Force (Join-Path $obsRoot 'portable_mode.txt') | Out-Null
    $wsConfig = Join-Path $obsRoot 'config/obs-studio/plugin_config/obs-websocket'
    New-Item -ItemType Directory -Force $wsConfig | Out-Null
    & icacls (Join-Path $obsRoot 'config') /inheritance:r /grant:r "${account}:(OI)(CI)F" 'SYSTEM:(OI)(CI)F' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Unable to secure OBS configuration' }
    $adapterPath = Join-Path $private 'adapter.json'
    if (Test-Path $adapterPath) { $adapter = Get-Content $adapterPath -Raw | ConvertFrom-Json } else {
        $adapter = @{ obsPassword = (New-Password); controllerConfig = 'C:/renki/controller.local.json'; healthFile = 'C:/renki/streaming/private/health.json'; slate = 'C:/renki/streaming/slate.html' }
        Save-Json $adapterPath $adapter
    }
    Save-Json (Join-Path $wsConfig 'config.json') @{server_enabled=$true; server_port=4455; auth_required=$true; server_password=$adapter.obsPassword; alerts_enabled=$false; first_load=$false}
    $profile = Join-Path $obsRoot 'config/obs-studio/basic/profiles/Renki'
    New-Item -ItemType Directory -Force $profile | Out-Null
    if (-not (Test-Path (Join-Path $profile 'basic.ini'))) {
        @'
[General]
Name=Renki
[Video]
BaseCX=1280
BaseCY=720
OutputCX=1280
OutputCY=720
FPSType=0
FPSCommon=30
[Output]
Mode=Simple
[SimpleOutput]
StreamEncoder=nvenc
VBitrate=3000
UseAdvanced=true
NVENCPreset2=p3
RecQuality=Stream
RecFormat2=mkv
FilePath=C:/renki/streaming
[Audio]
SampleRate=48000
ChannelSetup=Stereo
[Panels]
CookieId=renki-local
'@ | Set-Content -Encoding UTF8 (Join-Path $profile 'basic.ini')
        @'
[General]
FirstRun=true
[Basic]
Profile=Renki
ProfileDir=Renki
SceneCollection=Renki
SceneCollectionFile=Renki
[BasicWindow]
SetupWizard=false
[Audio]
DesktopDevice1=disabled
DesktopDevice2=disabled
AuxDevice1=disabled
AuxDevice2=disabled
AuxDevice3=disabled
AuxDevice4=disabled
'@ | Set-Content -Encoding UTF8 (Join-Path $obsRoot 'config/obs-studio/global.ini')
        Copy-Item (Join-Path $obsRoot 'config/obs-studio/global.ini') (Join-Path $obsRoot 'config/obs-studio/user.ini')
    }
    # Windows loopback remains reachable; deny network clients even though OBS listens on all interfaces.
    if (-not (Get-NetFirewallRule -Name RenkiObsPrivate -ErrorAction SilentlyContinue)) {
        New-NetFirewallRule -Name RenkiObsPrivate -DisplayName 'Renki OBS control stays local' -Direction Inbound -Action Block -Protocol TCP -LocalPort 4455 | Out-Null
    }
    Task 'RenkiOBS' $obsExe '--portable --disable-shutdown-check --disable-missing-files-check --minimize-to-tray --scene "Renki Offline"' (Split-Path $obsExe)
    Write-Host 'Pinned portable OBS prepared; private authenticated control; no output or game started'
}
if ($Action -eq 'adapter') {
    foreach ($name in @('RenkiCapture','RenkiCaptureWatchdog')) { Stop-ScheduledTask -TaskName $name -ErrorAction SilentlyContinue }
    Get-Process obs-adapter -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq (Join-Path $root 'obs-adapter.exe') } | Stop-Process -Force
    for ($attempt = 0; $attempt -lt 40; $attempt++) {
        try { Copy-Item (Join-Path $root 'obs-adapter.next.exe') (Join-Path $root 'obs-adapter.exe') -Force -ErrorAction Stop; break } catch [IO.IOException] { if ($attempt -eq 39) { throw }; Start-Sleep -Milliseconds 250 }
    }
    Remove-Item (Join-Path $root 'obs-adapter.next.exe')
    & (Join-Path $root 'obs-adapter.exe') -config (Join-Path $private 'adapter.json') -action configure
    if ($LASTEXITCODE -ne 0) { throw 'OBS configuration failed' }
    Task 'RenkiCapture' (Join-Path $root 'obs-adapter.exe') ('-config "' + (Join-Path $private 'adapter.json') + '" -action run') $root
    Task 'RenkiCaptureWatchdog' (Join-Path $root 'obs-adapter.exe') ('-config "' + (Join-Path $private 'adapter.json') + '" -action watchdog') $root
    Start-ScheduledTask -TaskName RenkiCapture
    Start-ScheduledTask -TaskName RenkiCaptureWatchdog
}
if ($Action -eq 'start') {
    $existing = Get-Process obs64 -ErrorAction SilentlyContinue
    if ($existing -and $existing.Path -ne $obsExe.Replace('/','\')) { throw 'A different OBS installation is already running' }
    Task 'RenkiOBS' $obsExe '--portable --disable-shutdown-check --disable-missing-files-check --minimize-to-tray --scene "Renki Offline"' (Split-Path $obsExe)
    if (-not $existing) { Start-ScheduledTask -TaskName RenkiOBS }
}
if ($Action -eq 'beacon') {
    if (Get-Process valheim -ErrorAction SilentlyContinue) { throw 'Stop Valheim before changing the render beacon setting' }
    . 'C:/renki/common.ps1'
    $settings = Read-BotSettings
    $path = Join-Path (Split-Path $settings.ValheimExe) 'BepInEx/config/com.warricksothr.renki.cfg'
    $value = [IO.File]::ReadAllText($path)
    if ($value -match '(?m)^CaptureBeacon\s*=') { $value = $value -replace '(?m)^CaptureBeacon\s*=.*$','CaptureBeacon = true' } else { $value += "`n[Streaming]`nCaptureBeacon = true`n" }
    [IO.File]::WriteAllText($path,$value,(New-Object Text.UTF8Encoding $false))
    Write-Host 'Opt-in game capture beacon enabled for the next game launch'
}
if ($Action -eq 'relay' -or $Action -eq 'enable') {
    $controllerPath = 'C:/renki/controller.local.json'
    $c = Get-Content $controllerPath -Raw | ConvertFrom-Json
    $address = $c.panel.listen.Split(':')[0]
    if (-not (Get-NetIPAddress -IPAddress $address -ErrorAction SilentlyContinue)) { throw 'ICE address must belong to BLIGHT' }
    $relayPath = Join-Path $private 'relay.json'
    $credentialPath = Join-Path $private 'reader.json'
    if (Test-Path $credentialPath) { $password = (Get-Content $credentialPath -Raw | ConvertFrom-Json).password }
    if (-not $password -or $password -notmatch '^[A-Za-z0-9_-]+$') { $password = New-Password; Save-Json $credentialPath @{password=$password} }
    Save-Json $relayPath @{
        logLevel='warn'; api=$false;metrics=$false;pprof=$false;playback=$false;hls=$false;srt=$false;moq=$false;
        rtsp=$true;rtspAddress='127.0.0.1:18554';rtspTransports=@('tcp');rtmp=$true;rtmpAddress='127.0.0.1:11935';
        webrtc=$true;webrtcAddress='127.0.0.1:18889';webrtcLocalUDPAddress=($address+':18189');webrtcLocalTCPAddress=($address+':18189');
        webrtcIPsFromInterfaces=$false;webrtcAdditionalHosts=@($address);webrtcICEServers2=@();webrtcAllowOrigins=@();
        authInternalUsers=@(
            @{user='any';ips=@('127.0.0.1');permissions=@(@{action='publish';path='renki'})},
            @{user='panel';pass=$password;ips=@('127.0.0.1');permissions=@(@{action='read';path='renki'})}
        );
        paths=@{renki=@{source='publisher';maxReaders=4;overridePublisher=$false;alwaysAvailable=$true;alwaysAvailableFile='C:/renki/streaming/standby-obs.mp4';record=$false;forward=@()}}
    }
    foreach ($protocol in @('TCP','UDP')) {
        $name = 'RenkiVideoICE'+$protocol
        if (-not (Get-NetFirewallRule -Name $name -ErrorAction SilentlyContinue)) {
            New-NetFirewallRule -Name $name -DisplayName ('Renki private video '+$protocol) -Direction Inbound -Action Allow -Protocol $protocol -LocalPort 18189 -LocalAddress $address -RemoteAddress '100.64.0.0/10' | Out-Null
        }
    }
    if ($Action -eq 'relay') {
        Task 'RenkiRelay' (Join-Path $root 'mediamtx.exe') ('"'+$relayPath+'"') $root
        Start-ScheduledTask -TaskName RenkiRelay
        Write-Host 'Private relay started; control/ingress loopback, ICE tailnet only'
    } else {
        if (-not (Test-Path ($controllerPath+'.before-video'))) { Copy-Item $controllerPath ($controllerPath+'.before-video') }
        $c.panel | Add-Member -Force -NotePropertyName video -NotePropertyValue @{port=18889;password=$password;healthFile='C:/renki/streaming/private/health.json'}
        Save-Json $controllerPath $c
        Write-Host 'Panel video configured; restart controller separately'
    }
}
if ($Action -eq 'local-start' -or $Action -eq 'local-stop' -or $Action -eq 'snapshot' -or $Action -eq 'game-snapshot' -or $Action -eq 'trial-source-loss') {
    & (Join-Path $root 'obs-adapter.exe') -config (Join-Path $private 'adapter.json') -action $Action
    if ($LASTEXITCODE -ne 0) { throw 'Local OBS action failed' }
}
if ($Action -eq 'trial-watchdog') {
    $exe = Join-Path $root 'obs-adapter.exe'
    $config = Join-Path $private 'adapter.json'
    $before = & $exe -config $config -action status | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0 -or $before.scene -ne 'Renki Gameplay' -or -not $before.stream.outputActive) { throw 'Watchdog trial needs healthy local gameplay output' }
    try {
        Stop-ScheduledTask -TaskName RenkiCapture
        Get-CimInstance Win32_Process -Filter "Name='obs-adapter.exe'" | Where-Object { $_.CommandLine -match '-action run' -and $_.ExecutablePath -eq $exe.Replace('/','\') } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force }
        $pass = $false
        for ($i=0; $i -lt 15; $i++) {
            Start-Sleep -Milliseconds 500
            $after = & $exe -config $config -action status | ConvertFrom-Json
            if ($LASTEXITCODE -eq 0 -and $after.scene -eq 'Renki Offline' -and $after.stream.outputActive -and $after.stream.outputDuration -gt $before.stream.outputDuration) { $pass=$true; break }
        }
        if (-not $pass) { throw 'Independent watchdog did not preserve output on standby' }
        Write-Host 'PASS: adapter killed; independent watchdog selected standby; local output remained active'
    } finally { Start-ScheduledTask -TaskName RenkiCapture }
}
if ($Action -eq 'https') {
    $c = Get-Content 'C:/renki/controller.local.json' -Raw | ConvertFrom-Json
    $parts = $c.panel.listen.Split(':')
    if (-not (Get-NetIPAddress -IPAddress $parts[0] -ErrorAction SilentlyContinue)) { throw 'Panel must be on BLIGHT' }
    & netsh interface portproxy add v4tov4 listenaddress=127.0.0.1 listenport=18780 connectaddress=$($parts[0]) connectport=$($parts[1]) | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Unable to create loopback panel forwarding' }
    & 'C:/Program Files/Tailscale/tailscale.exe' serve --bg 'http://127.0.0.1:18780'
    if ($LASTEXITCODE -ne 0) { throw 'Private HTTPS setup failed' }
}
if ($Action -eq 'slate') {
    $path=Join-Path $private 'relay.json'
    $c=Get-Content $path -Raw | ConvertFrom-Json
    $c.paths.renki.alwaysAvailableFile='C:/renki/streaming/standby-obs.mp4'
    $c.logLevel='info'
    $c | Add-Member -Force -NotePropertyName logDestinations -NotePropertyValue @('file')
    $c | Add-Member -Force -NotePropertyName logFile -NotePropertyValue 'C:/renki/streaming/private/relay.log'
    Save-Json $path $c
    Stop-ScheduledTask -TaskName RenkiRelay
    Get-Process mediamtx -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq (Join-Path $root 'mediamtx.exe').Replace('/','\') } | Stop-Process -Force
    Start-ScheduledTask -TaskName RenkiRelay
    Write-Host 'Matching H264 + silent AAC slate installed; no microphone, desktop or game audio captured'
}
if ($Action -eq 'codec') {
    # Only the dedicated portable instance is touched. Outputs must already be stopped.
    $process = Get-Process obs64 -ErrorAction SilentlyContinue
    if ($process) {
        if ($process.Path -ne $obsExe.Replace('/','\')) { throw 'Unexpected OBS installation' }
        $diagnostics = & (Join-Path $root 'obs-adapter.exe') -config (Join-Path $private 'adapter.json') -action status | ConvertFrom-Json
        if ($LASTEXITCODE -ne 0 -or $diagnostics.stream.outputActive) { throw 'Stop local output before changing its encoder' }
        & (Join-Path $root 'obs-adapter.exe') -config (Join-Path $private 'adapter.json') -action standby
        [void]$process.CloseMainWindow()
        if (-not $process.WaitForExit(5000)) { Stop-ScheduledTask -TaskName RenkiOBS; Start-Sleep -Seconds 1 }
        if (Get-Process -Id $process.Id -ErrorAction SilentlyContinue) { Stop-Process -Id $process.Id -Force; $process.WaitForExit(5000) | Out-Null }
    }
    if (Get-Process obs64 -ErrorAction SilentlyContinue) { throw 'OBS must be stopped before encoder settings change' }
    $profile = Join-Path $obsRoot 'config/obs-studio/basic/profiles/Renki'
    $iniPath = Join-Path $profile 'basic.ini'
    $ini = [IO.File]::ReadAllText($iniPath)
    $ini = $ini.Replace('Mode=Simple','Mode=Advanced')
    $advanced = "[AdvOut]`nEncoder=obs_nvenc_h264_tex`nTrackIndex=1`nApplyServiceSettings=false`nRescale=false`nRecType=Standard`nRecEncoder=none`nRecFilePath=C:/renki/streaming`nRecFormat2=mkv`n`n"
    if ($ini -match '(?m)^\[AdvOut\]') { $ini = [regex]::Replace($ini, '(?ms)^\[AdvOut\].*?(?=^\[|\z)', $advanced) } else { $ini += "`n"+$advanced }
    [IO.File]::WriteAllText($iniPath,$ini,(New-Object Text.UTF8Encoding $false))
    Save-Json (Join-Path $profile 'streamEncoder.json') @{rate_control='CBR';bitrate=3000;keyint_sec=2;preset='p3';profile='baseline';bf=0;bframes=0;lookahead=$false;psycho_aq=$false;multipass='disabled'}
    Start-ScheduledTask -TaskName RenkiOBS
    Write-Host 'NVENC baseline H264, no B frames, 720p30 configured; local output remains stopped'
}
if ($Action -eq 'stop') {
    & (Join-Path $root 'obs-adapter.exe') -config (Join-Path $private 'adapter.json') -action local-stop
    if ($LASTEXITCODE -ne 0) { throw 'Refusing to stop an unverified OBS output' }
    & (Join-Path $root 'obs-adapter.exe') -config (Join-Path $private 'adapter.json') -action standby
    Stop-ScheduledTask -TaskName RenkiCapture -ErrorAction SilentlyContinue
    Stop-ScheduledTask -TaskName RenkiCaptureWatchdog -ErrorAction SilentlyContinue
    Stop-ScheduledTask -TaskName RenkiOBS -ErrorAction SilentlyContinue
    Get-Process obs64 -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $obsExe.Replace('/','\') } | Stop-Process -Force
}
if ($Action -eq 'status') {
    Get-ScheduledTask -TaskName RenkiOBS,RenkiCapture,RenkiCaptureWatchdog,RenkiRelay -ErrorAction SilentlyContinue | Select-Object TaskName,State | Format-Table -AutoSize
    Get-Process obs64,obs-adapter,mediamtx -ErrorAction SilentlyContinue | Select-Object Name,Id,CPU,WorkingSet64 | Format-Table -AutoSize
    if (Test-Path (Join-Path $private 'status.json')) { Get-Content (Join-Path $private 'status.json') }
    if (Test-Path (Join-Path $root 'obs-adapter.exe')) { & (Join-Path $root 'obs-adapter.exe') -config (Join-Path $private 'adapter.json') -action status }
}
