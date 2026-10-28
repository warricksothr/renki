<#
.SYNOPSIS
One-time setup that lets renki's tooling drive this VM over SSH.

.DESCRIPTION
Run once, in an elevated Windows PowerShell, as the Windows account the bot
plays under. It is safe to run again. It:

  1. installs and starts Windows' built-in OpenSSH Server, with PowerShell as
     its shell and key-only logins;
  2. authorises the public key in renki-agent.pub beside this script (or
     -PublicKey);
  3. registers the RenkiStart and RenkiStop scheduled tasks. They run
     start-bot.ps1 and stop-bot.ps1 in this account's desktop session, so
     the game gets the desktop and the GPU, which a program started straight
     from SSH would not;
  4. stops the VM sleeping and turning its display off;
  5. creates bot.local.psd1 from the example when there is none.

Auto-logon is left to you (Sysinternals Autologon), because it needs this
account's password.

.PARAMETER PublicKey
The SSH public key to authorise. Defaults to renki-agent.pub beside this script.

.PARAMETER AllowFrom
Restrict SSH to these remote addresses, for example 192.168.1.0/24. Default: any.

.PARAMETER KeepPasswordLogins
Leave SSH password logins on. By default only keys are accepted.
#>
param(
    [string]$PublicKey = '',
    [string[]]$AllowFrom = @(),
    [switch]$KeepPasswordLogins
)

$ErrorActionPreference = 'Stop'
$here = $PSScriptRoot

$principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Run this from an elevated PowerShell (Run as administrator).'
}
$user = "$env:USERDOMAIN\$env:USERNAME"
# sshd picks the key file by Administrators membership, which an elevated
# token carries as the S-1-5-32-544 group.
$isAdmin = [Security.Principal.WindowsIdentity]::GetCurrent().Groups.Value -contains 'S-1-5-32-544'

function Step($text) { Write-Host "`n== $text" -ForegroundColor Cyan }

# 1. OpenSSH Server
Step 'OpenSSH Server'
# Any OpenSSH Server will do: the Windows capability, or the MSI from
# github.com/PowerShell/Win32-OpenSSH/releases, which needs no Windows Update.
if (Get-Service sshd -ErrorAction SilentlyContinue) {
    Write-Host 'sshd is already installed'
} else {
    $cap = Get-WindowsCapability -Online -Name 'OpenSSH.Server*' | Select-Object -First 1
    Write-Host "installing $($cap.Name) through Windows Update; this can take a long time."
    Write-Host 'If it hangs, stop it (Ctrl+C), install the OpenSSH MSI instead, and run this again.'
    Add-WindowsCapability -Online -Name $cap.Name | Out-Null
}
Set-Service -Name sshd -StartupType Automatic
Start-Service sshd
New-Item -Path 'HKLM:\SOFTWARE\OpenSSH' -Force -ErrorAction SilentlyContinue | Out-Null
New-ItemProperty -Path 'HKLM:\SOFTWARE\OpenSSH' -Name DefaultShell -Force -PropertyType String `
    -Value "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe" | Out-Null

$rule = Get-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue
if (-not $rule) {
    $rule = New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -DisplayName 'OpenSSH Server (sshd)' `
        -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22
}
if ($AllowFrom.Count -gt 0) {
    Set-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -RemoteAddress $AllowFrom
    Write-Host "ssh allowed from $($AllowFrom -join ', ')"
}

# 2. The key
Step 'SSH key'
if (-not $PublicKey) {
    $keyFile = Join-Path $here 'renki-agent.pub'
    if (-not (Test-Path $keyFile)) { throw "No -PublicKey and no $keyFile." }
    $PublicKey = (Get-Content $keyFile -Raw).Trim()
}
if ($PublicKey -notmatch '^(ssh-ed25519|ssh-rsa|ecdsa-sha2-\S+) ') { throw "That does not look like a public key: $PublicKey" }

# Windows' sshd reads administrators' keys from one shared file, and ignores
# it unless only Administrators and SYSTEM can touch it.
if ($isAdmin) {
    $auth = "$env:ProgramData\ssh\administrators_authorized_keys"
} else {
    $auth = Join-Path $env:USERPROFILE '.ssh\authorized_keys'
    New-Item -ItemType Directory -Force -Path (Split-Path $auth) | Out-Null
}
$existing = if (Test-Path $auth) { Get-Content $auth } else { @() }
if ($existing -notcontains $PublicKey) {
    Add-Content -Path $auth -Value $PublicKey -Encoding ascii
    Write-Host "key added to $auth"
} else {
    Write-Host "key already in $auth"
}
if ($isAdmin) {
    icacls.exe $auth /inheritance:r /grant 'Administrators:F' /grant 'SYSTEM:F' | Out-Null
}

$config = "$env:ProgramData\ssh\sshd_config"
if (-not $KeepPasswordLogins -and (Test-Path $config)) {
    $text = Get-Content $config -Raw
    if ($text -notmatch '(?m)^PasswordAuthentication no') {
        $text = $text -replace '(?m)^#?PasswordAuthentication .*$', 'PasswordAuthentication no'
        if ($text -notmatch '(?m)^PasswordAuthentication no') { $text = "PasswordAuthentication no`r`n" + $text }
        Set-Content -Path $config -Value $text -Encoding ascii
        Write-Host 'password logins off'
    }
}
Restart-Service sshd

# 3. Scheduled tasks that run in this account's desktop session
Step 'Scheduled tasks'
$ps = "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe"
$taskPrincipal = New-ScheduledTaskPrincipal -UserId $user -LogonType Interactive -RunLevel Limited
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries `
    -ExecutionTimeLimit (New-TimeSpan -Minutes 10) -MultipleInstances IgnoreNew
foreach ($t in @(
        @{ Name = 'RenkiStart'; Script = 'start-bot.ps1'; Args = '-WaitMinutes 0' },
        @{ Name = 'RenkiStop'; Script = 'stop-bot.ps1'; Args = '' })) {
    $file = Join-Path $here $t.Script
    $action = New-ScheduledTaskAction -Execute $ps -WorkingDirectory $here `
        -Argument "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$file`" $($t.Args)"
    Register-ScheduledTask -TaskName $t.Name -Action $action -Principal $taskPrincipal -Settings $settings `
        -Description 'renki: run by the remote tooling over SSH' -Force | Out-Null
    Write-Host "$($t.Name) -> $($t.Script)"
}

# 4. Keep the VM awake with its display on
Step 'Power'
powercfg.exe /change standby-timeout-ac 0
powercfg.exe /change monitor-timeout-ac 0
powercfg.exe /change hibernate-timeout-ac 0
Write-Host 'sleep and display timeout off'

# 5. Settings
Step 'Settings'
$settingsFile = Join-Path $here 'bot.local.psd1'
if (-not (Test-Path $settingsFile)) {
    Copy-Item (Join-Path $here 'bot.example.psd1') $settingsFile
    Write-Host "created $settingsFile; fill it in"
} else {
    Write-Host "$settingsFile exists"
}

Step 'Done'
$ips = Get-NetIPAddress -AddressFamily IPv4 |
    Where-Object { $_.IPAddress -notlike '127.*' -and $_.IPAddress -notlike '169.254.*' } |
    Select-Object -ExpandProperty IPAddress
Write-Host "account:  $user (administrator: $isAdmin)"
Write-Host "address:  $($ips -join ', ')"
Write-Host "renki at: $here"
Write-Host "`nTell renki's tooling: $env:USERNAME@<one of the addresses above>, folder $here"
