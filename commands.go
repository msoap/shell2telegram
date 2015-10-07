package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/Syfaro/telegram-bot-api"
)

// Ctx - context for bot command function (users, command, args, ...)
type Ctx struct {
	bot         *tgbotapi.BotAPI
	appConfig   *Config       // configuration
	commands    Commands      // all chat commands
	users       Users         // all users
	userID      int           // current user
	allowExec   bool          // is user authorized
	allMessage  string        // user message completely
	messageCmd  string        // command name
	messageArgs string        // command arguments
	exitChan    chan struct{} // for signal for terminate
}

// /auth and /authroot - authorize users
func cmdAuth(ctx Ctx) (replayMsg string) {
	forRoot := ctx.messageCmd == "/authroot"

	if ctx.messageArgs == "" {

		replayMsg = "See code in terminal with shell2telegram or ask code from root user and type:\n" + ctx.messageCmd + " code"
		authCode := ctx.users.DoLogin(ctx.userID, forRoot)

		rootRoleStr := ""
		if forRoot {
			rootRoleStr = "root "
		}
		secretCodeMsg := fmt.Sprintf("Request %saccess for %s. Code: %s\n", rootRoleStr, ctx.users.String(ctx.userID), authCode)
		fmt.Print(secretCodeMsg)
		ctx.users.BroadcastForRoots(ctx.bot, secretCodeMsg)

	} else {
		if ctx.users.IsValidCode(ctx.userID, ctx.messageArgs, forRoot) {
			ctx.users.SetAuthorized(ctx.userID, forRoot)
			if forRoot {
				replayMsg = fmt.Sprintf("You (%s) authorized as root.", ctx.users.String(ctx.userID))
				log.Print("root authorized: ", ctx.users.String(ctx.userID))
			} else {
				replayMsg = fmt.Sprintf("You (%s) authorized.", ctx.users.String(ctx.userID))
				log.Print("authorized: ", ctx.users.String(ctx.userID))
			}
		} else {
			replayMsg = fmt.Sprintf("Code is not valid.")
		}
	}

	return replayMsg
}

// /help
func cmdHelp(ctx Ctx) (replayMsg string) {
	helpMsg := []string{}

	if ctx.allowExec {
		for cmd, shellCmd := range ctx.commands {
			helpMsg = append(helpMsg, cmd+" → "+shellCmd)
		}
	}

	helpMsg = append(helpMsg,
		"/auth [code] → authorize user",
		"/authroot [code] → authorize user as root",
	)

	if ctx.users.IsRoot(ctx.userID) {
		helpMsg = append(helpMsg,
			"/shell2telegram stat → get stat about users",
			"/shell2telegram search <query> → search users by name/id",
			"/shell2telegram ban <user_id|username> → ban user",
			"/shell2telegram desc <bot description> → set bot description",
			"/shell2telegram rm </command> → delete command",
			"/shell2telegram version → show version",
		)
		if ctx.appConfig.addExit {
			helpMsg = append(helpMsg, "/shell2telegram exit → terminate bot")
		}
	}

	if ctx.appConfig.description != "" {
		replayMsg = ctx.appConfig.description
	} else {
		replayMsg = "This bot created with shell2telegram"
	}
	replayMsg += "\n\n" +
		"available commands:\n" +
		strings.Join(helpMsg, "\n")

	return replayMsg
}

// /shell2telegram stat
func cmdShell2telegramStat(ctx Ctx) (replayMsg string) {
	for userID := range ctx.users.list {
		replayMsg += ctx.users.StringVerbose(userID) + "\n"
	}

	return replayMsg
}

// /shell2telegram search
func cmdShell2telegramSearch(ctx Ctx) (replayMsg string) {
	query := ctx.messageArgs

	if query == "" {
		return "Please set query: /shell2telegram search <query>"
	}

	for _, userID := range ctx.users.Search(query) {
		replayMsg += ctx.users.StringVerbose(userID) + "\n"
	}

	return replayMsg
}

// /shell2telegram ban
func cmdShell2telegramBan(ctx Ctx) (replayMsg string) {
	userName := ctx.messageArgs

	if userName == "" {
		return "Please set user_id or login: /shell2telegram ban <user_id|@username>"
	}

	userID, err := strconv.Atoi(userName)
	if err != nil {
		userName = regexp.MustCompile("@").ReplaceAllLiteralString(userName, "")
		userID = ctx.users.GetUserIDByName(userName)
	}

	if userID > 0 && ctx.users.BanUser(userID) {
		replayMsg = fmt.Sprintf("User %s banned", ctx.users.String(userID))
	} else {
		replayMsg = "User not found"
	}

	return replayMsg
}

// all commands from command-line
func cmdUser(ctx Ctx) (replayMsg string) {
	if cmd, found := ctx.commands[ctx.messageCmd]; found {
		replayMsg = _execShell(cmd, ctx.messageArgs)
	}

	return replayMsg
}

// plain text handler
func cmdPlainText(ctx Ctx) (replayMsg string) {
	if cmd, found := ctx.commands["/:plain_text"]; found {
		replayMsg = _execShell(cmd, ctx.allMessage)
	}

	return replayMsg
}

// set bot description
func cmdShell2telegramDesc(ctx Ctx) (replayMsg string) {
	description := ctx.messageArgs

	if description == "" {
		return "Please set description: /shell2telegram desc <bot description>"
	}

	ctx.appConfig.description = description
	replayMsg = "Bot description set to: " + description

	return replayMsg
}

// /shell2telegram rm "/command" - delete command
func cmdShell2telegramRm(ctx Ctx) (replayMsg string) {
	commandName := ctx.messageArgs

	if commandName == "" {
		return "Please set command for delete: /shell2telegram rm </command>"
	}
	if _, ok := ctx.commands[commandName]; ok {
		delete(ctx.commands, commandName)
		replayMsg = "Deleted command: " + commandName
	} else {
		replayMsg = fmt.Sprintf("Command %s not found", commandName)
	}

	return replayMsg
}

// /shell2telegram version - get version
func cmdShell2telegramVersion(ctx Ctx) (replayMsg string) {
	replayMsg = fmt.Sprintf("shell2telegram %s", VERSION)
	return replayMsg
}

// /shell2telegram exit - terminate bot
func cmdShell2telegramExit(ctx Ctx) (replayMsg string) {
	if ctx.appConfig.addExit {
		replayMsg = "bye..."
		go func() {
			ctx.exitChan <- struct{}{}
		}()
	}
	return replayMsg
}

// internal function for exec shell commands
func _execShell(shellCmd, input string) (result string) {
	shell, params := "sh", []string{"-c", shellCmd}
	osExecCommand := exec.Command(shell, params...)
	osExecCommand.Stderr = os.Stderr

	// write user input to STDIN
	if input != "" {
		stdin, err := osExecCommand.StdinPipe()
		if err == nil {
			io.WriteString(stdin, input)
			stdin.Close()
		} else {
			log.Print("get STDIN error: ", err)
		}
	}

	shellOut, err := osExecCommand.Output()
	if err != nil {
		log.Print("exec error: ", err)
		result = fmt.Sprintf("exec error: %s", err)
	} else {
		result = string(shellOut)
	}

	return result
}
