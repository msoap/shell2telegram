shell2telegram
==============

Create Telegram bot from command-line

Install
-------

MacOS:

    brew tap msoap/tools
    brew install shell2telegram
    # update:
    brew update; brew upgrade shell2telegram

Or download binaries from: [releases](https://github.com/msoap/shell2telegram/releases) (OS X/Linux/Windows/RaspberryPi)

Or build from source:

    # install Go
    # set $GOPATH if needed
    go get -u github.com/msoap/shell2telegram
    ln -s $GOPATH/bin/shell2telegram ~/bin/shell2telegram # or add $GOPATH/bin to $PATH

Usage
-----

Get token from [BotFather bot](https://telegram.me/BotFather), and set TB_TOKEN var in shell

    export TB_TOKEN=*******
    shell2telegram [options] /chat_command 'shell command' /chat_command2 'shell command2'
    options:
        -allow-users=<NAMES> : telegram users who are allowed to chat with the bot ("user1,user2")
        -root-users=<NAMES>  : telegram users, who confirms new users in their private chat ("user1,user2")
        -allow-all           : allow all users (DANGEROUS!)
        -add-exit            : adding "/shell2telegram exit" command for terminate bot (for roots only)
        -log-commands        : logging all commands
        -tb-token=<TOKEN>    : setting bot token (or set TB_TOKEN variable)
        -timeout=<NN>        : setting timeout for bot (default 60 sec)
        -description=<TITLE> : setting description of bot
        -version
        -help

If not define -allow-users/-root-users options - authorize users via secret code from console or via chat with exists root users.

All text after /chat_command will be sent to STDIN of shell command.

If chat command is /:plain_text - get user message without any /command (for private chats only)

Predefined bot commands
-----------------------

  * /help - list available commands
  * /auth - begin authorize new user
  * /auth CODE - authorize with code from console or from exists root user
  * /authroot - same for new root user
  * /authroot CODE - same for new root user

for root users only:

  * /shell2telegram stat - show users statistics
  * /shell2telegram search query - search users by name/id
  * /shell2telegram ban user_id|username - ban user
  * /shell2telegram exit - terminate bot (for run with -add-exit)
  * /shell2telegram desc "description" - set bot description
  * /shell2telegram version - show version

Examples
--------

    # system information
    shell2telegram /top 'top -l 1 | head -10' /date 'date' /ps 'ps aux -m | head -20'
    
    # sound volume control via telegram (Mac OS)
    shell2telegram /get  'osascript -e "output volume of (get volume settings)"' \
                   /up   'osascript -e "set volume output volume (($(osascript -e "output volume of (get volume settings)")+10))"' \
                   /down 'osascript -e "set volume output volume (($(osascript -e "output volume of (get volume settings)")-10))"'

Links
-----

  * [Telegram channel about shell2telegram](https://telegram.me/shell2telegram)
  * [About Telegram bots](https://core.telegram.org/bots)
  * [Golang bindings for the Telegram Bot API](https://github.com/Syfaro/telegram-bot-api)
  * [shell2http - shell commands as http-server](https://github.com/msoap/shell2http)
