# Setting up the bot VM for remote control

The bot plays on a Windows VM with its own GPU. Once this page is done, the
repository's tooling can deploy, start, stop and inspect the bot over SSH
with `just vm …`, and nobody needs to be at the VM.

A program started straight from an SSH session has no desktop and no GPU, so
the game is never launched that way. SSH starts a scheduled task, and the
task runs the game in the bot account's own desktop session. That session
has to stay logged in: auto-logon and `release-rdp.ps1` below see to that.

## Once, at the VM, over Remote Desktop

1. **Install the game.** Install Valheim with the bot's own Steam account
   (TKT-01M61T36CT). Leave Steam signed in and set to start at logon.
2. **Install BepInEx.** Install
   [BepInExPack_Valheim](https://thunderstore.io/c/valheim/p/denikson/BepInExPack_Valheim/)
   **into the game folder** itself, not into an r2modman profile, because the
   scripts start `valheim.exe` directly. Put any mods the server requires in
   `BepInEx\plugins` beside it. Start the game once so BepInEx makes its
   folders, then quit.
3. **Copy renki over.** On the dev box, run `just package`, then copy
   `dist\renki-vm\` to `C:\renki` on the VM. Copying a folder through Remote
   Desktop works: in Explorer the dev box's copy is at
   `\\wsl.localhost\Ubuntu\<repo>\dist\renki-vm`. The folder carries
   `renki-agent.pub`, the dev box's SSH public key.
4. **Run the setup.** Open Windows PowerShell **as administrator**, logged in
   as the bot's account, and run:

       powershell -ExecutionPolicy Bypass -File C:\renki\setup-remote.ps1

   Installing OpenSSH goes through Windows Update and can take a long time,
   or hang if Windows Update is off. If it does, stop the script, install
   `OpenSSH-Win64-v*.msi` from
   [Win32-OpenSSH releases](https://github.com/PowerShell/Win32-OpenSSH/releases),
   and run the script again: it skips the install when `sshd` already exists.

   Add `-AllowFrom 192.168.1.0/24` (your LAN) to limit who may reach SSH. It
   ends by printing the account and the addresses; keep them.
5. **Fill in the settings.**
   - `C:\renki\bot.local.psd1` (created by the setup): the server, the
     character's file name, the password, and the path to `valheim.exe`.
   - `C:\renki\controller.local.json`, copied from `controller.example.json`:
     put your character name in `owners`, and delete the `llm` block for a
     first run on the command grammar alone.
6. **Turn on auto-logon** for the bot's account with
   [Sysinternals Autologon](https://learn.microsoft.com/sysinternals/downloads/autologon).
   The VM should then come up at the desktop after a reboot.
7. **Give the GPU something to draw on.** If nothing is plugged into the
   passed-through card, use an HDMI or DisplayPort dummy plug. Windows may
   not render a desktop for a GPU with no display attached.
8. **Leave without locking the desktop.** Instead of closing the Remote
   Desktop window, run this from an elevated prompt:

       C:\renki\release-rdp.ps1

   It ends your session but leaves the desktop running on the VM's console.

## Then, on the dev box

    echo 'botuser@192.168.1.50' > vm.local   # the account and address from step 4
    just vm check                            # whoami, hostname, RenkiStart/RenkiStop
    just vm start                            # waits until the bot is in the world
    just vm status
    just vm logs                             # the renki lines of the BepInEx log, and the controller's
    just vm audit 1 -last 10m                # what happened, from the audit log (docs/AUDIT.md)
    just vm stop
    just vm deploy                           # package, stop, copy, install the plugin, start

## When something is off

- **`just vm check` cannot connect.** Run `Get-Service sshd` on the VM. Also
  check the firewall rule `OpenSSH-Server-In-TCP`, and that the key in
  `C:\ProgramData\ssh\administrators_authorized_keys` (administrators) or
  `%USERPROFILE%\.ssh\authorized_keys` (anyone else) is the dev box's.
- **The game starts but shows nothing, or stalls once you disconnect.** The
  session lost its display. Reconnect, then leave with `release-rdp.ps1`.
  TKT-01M61T36D (Spike: does Valheim keep running unattended on the VM)
  records what works.
- **`RenkiStart` does nothing.** Look at the task's History in Task
  Scheduler. It runs only while the bot's account is logged on, which is why
  auto-logon matters.
