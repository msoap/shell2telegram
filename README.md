shell2telegram
==============

Create Telegram bot from command-line

Usage:

Get token from [BotFather bot](https://telegram.me/BotFather), and set TB_TOKEN var in shell

    export TB_TOKEN=*******
    shell2telegram [options] /chat_command 'shell command' /chat_command2 'shell command2'
    options:
        -allow-users=<NAMES> : users telegram-names who allow chats with bot ("user1,user2")
        -root-users=<NAMES>  : users telegram-names who confirm new users through of it private chat ("user1,user2")
        -allow-all           : allow all user (DANGEROUS!)
        -add-exit            : add "/shell2telegram exit" command for terminate bot (for root only)
        -log-commands        : logging all commands
        -tb-token=<TOKEN>    : set bot token (or set TB_TOKEN variable)
        -timeout=<NN>        : set timeout for bot (default 60 sec)
        -version
        -help

If not define -allow-users/-root-users options - authorize users via secret code from console or via chat with exists root users.

Predefined commands
-------------------

  * /help - list available commands
  * /auth - begin authorize user
  * /auth CODE - authorize with code from console or from exists root user
  * /authroot - same for root user
  * /authroot CODE - same for root user
  * /shell2telegram stat - show users statistics (for roots only)
  * /shell2telegram exit - terminate bot (for run with -add-exit and roots only)
  * /shell2telegram version - show version

Examples
--------

    # system information
    shell2telegram /top "top -l 1 | head -10" /date "date" /ps "ps aux -m | head -20"
    
    # sound volume control via telegram (Mac OS)
    shell2telegram /get  'osascript -e "output volume of (get volume settings)"' \
                   /up   'osascript -e "set volume output volume (($(osascript -e "output volume of (get volume settings)")+10))"' \
                   /down 'osascript -e "set volume output volume (($(osascript -e "output volume of (get volume settings)")-10))"'

Links
-----

  * [About Telegram bots](https://core.telegram.org/bots)
  * [Golang bindings for the Telegram Bot API](https://github.com/Syfaro/telegram-bot-api)
  * [shell2http - shell commands as http-server](https://github.com/msoap/shell2http)
