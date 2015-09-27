shell2telegram
==============

Create Telegram bot from command-line

Usage:

Get token from [BotFather bot](https://telegram.me/BotFather), and set TB_TOKEN var in shell

    export TB_TOKEN=*******
    shell2telegram [options] /chat_command 'shell command' /chat_command2 'shell command2'
    options:
        -add-exit         : add /exit command for terminate bot
        -tb-token=<TOKEN> : set bot token (or set TB_TOKEN variable)
        -timeout=<NN>     : set timeout for bot (default 60 sec)
        -version
        -help

Links
-----

  * [About Telegram bots](https://core.telegram.org/bots)
  * [Golang bindings for the Telegram Bot API](https://github.com/Syfaro/telegram-bot-api)
  * [shell2http - shell commands as http-server](https://github.com/msoap/shell2http)
