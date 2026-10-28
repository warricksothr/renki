# Settings for start-bot.ps1 and stop-bot.ps1. Copy to bot.local.psd1 beside
# the scripts and fill in. bot.local.psd1 is gitignored: it holds the server
# password.
@{
    # The bot account's Valheim install. Steam must be running and signed in
    # as the bot's account.
    ValheimExe       = 'C:\Program Files (x86)\Steam\steamapps\common\Valheim\valheim.exe'

    # The server, as host:port (the game port, usually 2456).
    Server           = 'valheim.example.com:2456'

    # The character's save file name, without extension: usually the
    # character's name in lower case. Look in
    # %USERPROFILE%\AppData\LocalLow\IronGate\Valheim\characters_local.
    Character        = 'bjorn'

    # The server password, or '' for none.
    Password         = ''

    # Extra arguments for the game; a small window keeps the VM's GPU idle.
    ExtraArgs        = @('-screen-fullscreen', '0', '-screen-width', '1280', '-screen-height', '720')

    # The controller, when it runs on the VM. Leave ControllerExe '' when the
    # brain runs elsewhere (then ControlUrl points there).
    ControllerExe    = 'controller.exe'
    ControllerConfig = 'controller.local.json'

    # The controller's control API, and the token when one is configured
    # (or set RENKI_TOKEN in the environment instead).
    ControlUrl       = 'http://127.0.0.1:8765'
    Token            = ''
}
