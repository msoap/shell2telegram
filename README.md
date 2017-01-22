#<img src="https://raw.githubusercontent.com/msoap/shell2telegram/misc/img/shell2telegram_icon.png" width="32" height="32"> shell2telegram

[![Build Status](https://travis-ci.org/msoap/shell2telegram.svg?branch=master)](https://travis-ci.org/msoap/shell2telegram)
[![Coverage Status](https://coveralls.io/repos/github/msoap/shell2telegram/badge.svg?branch=master)](https://coveralls.io/github/msoap/shell2telegram?branch=master)
[![Docker Pulls](https://img.shields.io/docker/pulls/msoap/shell2telegram.svg?maxAge=3600)](https://hub.docker.com/r/msoap/shell2telegram/)
[![Homebrew formula exists](https://img.shields.io/badge/homebrew-🍺-d7af72.svg)](https://github.com/msoap/shell2telegram#install)
[![Report Card](https://goreportcard.com/badge/github.com/msoap/shell2telegram)](https://goreportcard.com/report/github.com/msoap/shell2telegram)

Create Telegram bot from command-line

Install
-------

MacOS:

    brew tap msoap/tools
    brew install shell2telegram
    # update:
    brew upgrade shell2telegram

Or download binaries from: [releases](https://github.com/msoap/shell2telegram/releases) (OS X/Linux/Windows/RaspberryPi)

Or build from source:

    # install Go
    # set $GOPATH if needed
    go get -u github.com/msoap/shell2telegram
    ln -s $GOPATH/bin/shell2telegram ~/bin/shell2telegram # or add $GOPATH/bin to $PATH

Or build image and run with Docker.
Example of `test-bot.Dockerfile` for bot who say current date:
```
FROM msoap/shell2telegram
# may be install some alpine packages:
# RUN apk add --no-cache ...
ENV TB_TOKEN=*******
CMD ["/date", "date"]
```

And build and run:

    docker build -f test-bot.Dockerfile -t test-bot .
    docker run --rm test-bot
    # or run with set token from command line
    docker run -e TB_TOKEN=******* --rm test-bot

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
        -persistent_users    : load/save users from file (default ~/.config/shell2telegram.json)
        -users_db=<FILENAME> : file for store users
        -cache=NNN           : caching command out for NNN seconds
        -public              : bot is public (dont add /auth* commands)
        -version
        -help

If not define -allow-users/-root-users options - authorize users via secret code from console or via chat with exists root users.

All text after /chat_command will be sent to STDIN of shell command.

Special chat commands
---------------------

for private chats only:

  * `/:plain_text` - get user message without any /command.

TODO:

  * `/:image` - for get image from user. Example: `/:image 'cat > file.jpg; echo ok'`
  * `/:file`  - for get file from user
  * `/:location`  - for get geo-location from user

Possible long-running shell processes (for example alarm/timer bot).

Autodetect images (png/jpg/gif/bmp) out from shell command, for example: `/get_image 'cat file.png'`

Setting environment variables for shell commands:

  * S2T_LOGIN - telegram @login (may be empty)
  * S2T_USERID - telegram user ID
  * S2T_USERNAME - telegram user name
  * S2T_CHATID - chat ID

Modificators for bot commands
-----------------------------

  * `:desc` - setting the description of command, `/cmd:desc="Command name" 'shell cmd'`
  * `:vars` - to create environment variables instead of text output to STDIN, `/cmd:vars=VAR1,VAR2 'echo $VAR1 / $VAR2'`
  * `:md` - to send message as markdown text, `/cmd:md 'echo "*bold* and _italic_"'`

TODO:

  * `/cmd:cron=3600` — periodic exec command, `/cmd:on args` - on, `/cmd:off` - off

Predefined bot commands
-----------------------

  * `/help` - list available commands
  * `/auth` - begin authorize new user
  * `/auth <CODE>` - authorize with code from console or from exists root user
  * `/authroot` - same for new root user
  * `/authroot <CODE>` - same for new root user

for root users only:

  * `/shell2telegram stat` - show users statistics
  * `/shell2telegram search <query>` - search users by name/id
  * `/shell2telegram ban <user_id|@username>` - ban user
  * `/shell2telegram exit` - terminate bot (for run with -add-exit)
  * `/shell2telegram desc <description>` - set bot description
  * `/shell2telegram rm </command>` - delete command
  * `/shell2telegram broadcast_to_root <message>` - send message to all root users in private chat
  * `/shell2telegram message_to_user <user_id|@username> <message>` - send message to user in private chat
  * `/shell2telegram version` - show version

Examples
--------

    # system information
    shell2telegram /top:desc="System information" 'top -l 1 | head -10' /date 'date' /ps 'ps aux -m | head -20'
    
    # sort any input
    shell2telegram /:plain_text sort
    
    # alarm bot:
    # /alarm time_in_seconds message
    shell2telegram /alarm:vars=SLEEP,MSG 'sleep $SLEEP; echo Hello $S2T_USERNAME; echo Alarm: $MSG'
    
    # sound volume control via telegram (Mac OS)
    shell2telegram /get  'osascript -e "output volume of (get volume settings)"' \
                   /up   'osascript -e "set volume output volume (($(osascript -e "output volume of (get volume settings)")+10))"' \
                   /down 'osascript -e "set volume output volume (($(osascript -e "output volume of (get volume settings)")-10))"'

Links
-----

  * [Telegram channel about shell2telegram](https://telegram.me/shell2telegram)
  * [About Telegram bots](https://core.telegram.org/bots)
  * [Golang bindings for the Telegram Bot API](https://github.com/go-telegram-bot-api/telegram-bot-api)
  * [shell2http - shell commands as http-server](https://github.com/msoap/shell2http)
