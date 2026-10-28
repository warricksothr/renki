<#
.SYNOPSIS
Leave Remote Desktop without taking the game's desktop with you.

.DESCRIPTION
Closing a Remote Desktop window leaves the session logged on but with no
display, and Windows may stop rendering the game. Run this instead, elevated,
from inside the Remote Desktop session. It hands the session back to the
VM's own console (the passed-through GPU), which ends your Remote Desktop
connection and keeps the desktop alive and unlocked.
#>
$ErrorActionPreference = 'Stop'
$session = (Get-Process -Id $PID).SessionId
tscon.exe $session /dest:console
