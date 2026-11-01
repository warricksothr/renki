# Setting up the bot VM for remote control

For independent OBS capture, standby scenes and private HTTPS video, see
[WINDOWS-VIDEO.md](WINDOWS-VIDEO.md). Streaming tasks are separate from the game
and controller lifecycle; they use this same SSH and interactive console setup.

The bot plays on a Windows VM with its own GPU. Once this page is done, the
repository's tooling can deploy, start, stop and inspect the bot over SSH
with `just vm …`, and nobody needs to be at the VM.

A program started straight from an SSH session has no desktop and no GPU, so
the game is never launched that way. SSH starts a scheduled task, and the
task runs the game in the bot account's own desktop session. That session
has to stay logged in: auto-logon and `release-rdp.ps1` below see to that.

## Monitor while the game is offline

`just vm panel-start` starts only the controller and web panel using a separate
scheduled task, without launching Valheim. The panel remains available to view
retained audit history while Tas is offline. Combat and harvest trial cleanup
logs Tas out, stops the game, restores combat off, then runs this recipe.
`just vm stop` still stops both the game and controller when explicitly used.

## Back up before changing server mods

Run `just vm migration-inspect` to list the sources and total size. After
`just vm stop`, run `just check vm migration-backup` to create a backup under
`C:\renki\backups\vplus-<UTC timestamp>`. This does not install mods or start
the game. Both Valheim and the local controller must stay stopped.

The backup includes BepInEx (plugins, configuration and loader assemblies),
the root Doorstop loader files, the controller executable, local settings and
secrets, the configured controller data directory (memory, goals and atlas),
local character/world saves, and any Valheim Steam Cloud save folders found
under the current account's Steam installation. The base game and server's
world files on another machine are outside this backup.

Every copied file is checked against the source using SHA256, with another
source check to detect changes during the copy. `manifest.json` records each
source path, backup path and checksum. Only a finished, verified copy receives
`complete.json`; a folder without it is incomplete. Backup permissions allow
only the current account and SYSTEM because the copy contains secrets. Keep
the contents on the VM; the command prints paths and counts, never settings
or save contents.

When rolling back, stop the game and controller, verify the manifest hashes,
and restore files to the source paths recorded there. Remove only files
introduced by the migration, such as a newly installed V+ DLL: copying the
old plugins over the new directory does not remove added mods. Restoring a
character save rewinds its inventory, so coordinate that with any server
world rollback rather than restoring it after ordinary play.

## Grantapher Valheim Plus migration

Confirm the host's exact release and settings first. For release 10.4.0.0,
run `just check vm vplus-stage 10.4.0.0` to download the official GitHub DLL
and verify its published SHA256 and size. Stop the client/controller and make
the migration backup above, then run `just check vm vplus-install 10.4.0.0`.
The installer rechecks the backup and DLL hashes and assembly version. It
adds only the mod DLL, verifies Renki's DLLs stayed unchanged, and refuses
existing V+ assemblies or configuration that would need reconciliation.

The existing game and BepInEx runtime are retained. The upstream release
documents Valheim 1.0.20 with BepInExPack 5.4.2351; Blight had Valheim 1.0.21
with BepInExPack 5.4.2360 at migration time. Successful local checks do not
extend upstream's supported version matrix. Check the chosen combination
instead of downgrading the runtime merely to match the bundled archive.

After startup, `just check vm vplus-inspect` reports DLL versions/hashes,
selected loader diagnostics, and local non-secret V+ configuration flags.
It does not report the effective server configuration: V+ synchronizes that
in memory and restores client values on disconnect. Check sync diagnostics
and the host's settings as well. With fresh 10.4.0.0 defaults, the optional
inventory, crafting-from-chests and automatic storage sections are disabled;
the Server section enables mod enforcement and configuration sync.

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
   The script also stops if the capability command returns without creating
   `sshd`, reporting the capability state and `RestartNeeded`. Complete a
   required restart before retrying. If the service is still absent, use the
   official 64-bit MSI instead of repeating the capability command:

       msiexec.exe /i C:\path\OpenSSH-Win64-vVERSION.msi ADDLOCAL=Server /qn /norestart /L*v C:\renki\openssh-install.log

   Wait for the installer to finish and check its exit code: 0 succeeds, 3010
   succeeds with a required restart, and other codes require the installer
   log. After any required restart, rerun `setup-remote.ps1` to restore the
   existing key, firewall scope and scheduled tasks. Existing bot settings
   are retained.

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
    just vm panel GET /api/status             # JSON; token stays on the VM
    just vm panel POST /api/act/stop           # stop through the panel API
    just vm logs                             # the renki lines of the BepInEx log, and the controller's
    just vm audit 1 -last 10m                # what happened, from the audit log (docs/AUDIT.md)
    just vm panel-setup                      # once: the web control panel on Tailscale (docs/PANEL.md)
    just vm stop
    just vm deploy                           # package, stop, copy, install the plugin, start

For controller-only changes, `just vm deploy-controller` builds/uploads the
Windows controller, stops it, installs the upload and restarts the controller
through its scheduled task. Valheim stays running. This refreshes the bridge
welcome and stops active body tasks; the plugin still needs a full
`just vm deploy` when its assemblies change.

## The LLM voice

Lines the command grammar does not know go to an LLM through terva, if
`controller.local.json` has an `llm` block. The VM uses the self-hosted
OpenAI-compatible backend `cpa`:

```json
"llm": {
  "cooldownS": 5,
  "dailyTokens": 200000,
  "terva": {
    "provider": "openai-compatible",
    "baseURL": "https://cpa-api.local.sothr.com/v1",
    "model": "gemma-4-12B-heretic-v3-mtp",
    "apiKeyEnv": "CPA_API_KEY",
    "tervaHome": "terva-home"
  }
}
```

The file names the key's variable, never the key. Store the key with

    just vm secret CPA_API_KEY      # prompts here, not shown, sent over SSH
    just vm restart

`just vm secret` keeps it in `C:\renki\secrets.local.json`, which only the bot's
account and SYSTEM can read, and `start-bot.ps1` hands it to the controller
as an environment variable. Without the key the controller logs `llm voice
off` and the bot answers the command grammar only.

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

## Install while keeping Tas offline

`just vm install` builds the package, stops the game if needed, copies it and
verifies the controller/plugin package SHA-256 manifest and installed DLLs.
It leaves the game stopped, for example while a recovery route is being
cleared. `just vm deploy` performs that same install and then starts Tas;
`just vm start` starts an already installed build. Use `just check vm install`
or `just check vm deploy` to retain deployment output.
