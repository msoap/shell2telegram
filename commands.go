package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/Syfaro/telegram-bot-api"
)

// Ctx - context for bot command function (users, command, args, ...)
type Ctx struct {
	bot         *tgbotapi.BotAPI
	appConfig   Config
	commands    Commands
	users       Users
	userID      int
	allowExec   bool
	messageCmd  string
	messageArgs string
}

// authorize users /auth and /authroot
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
		ctx.users.broadcastForRoots(ctx.bot, secretCodeMsg)

	} else {
		if ctx.users.IsValidCode(ctx.userID, ctx.messageArgs, forRoot) {
			ctx.users.list[ctx.userID].IsAuthorized = true
			if forRoot {
				ctx.users.list[ctx.userID].IsRoot = true
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
	helpMsg := []string{
		"/auth [code] → authorize user",
		"/authroot [code] → authorize user as root",
	}

	if ctx.allowExec {
		for cmd, shellCmd := range ctx.commands {
			helpMsg = append(helpMsg, cmd+" → "+shellCmd)
		}
		if ctx.users.IsRoot(ctx.userID) {
			helpMsg = append(helpMsg, "/shell2telegram stat → get stat about users")
			if ctx.appConfig.addExit {
				helpMsg = append(helpMsg, "/shell2telegram exit → terminate bot")
			}
		}
	}

	helpMsg = append(helpMsg, "/shell2telegram version → show version")
	replayMsg = "This bot created with shell2telegram\n\n" +
		"available commands:\n" +
		strings.Join(helpMsg, "\n")

	return replayMsg
}

// /shell2telegram stat
func cmdShell2telegramStat(ctx Ctx) (replayMsg string) {
	for userID, user := range ctx.users.list {
		replayMsg += fmt.Sprintf("%s: auth: %v, root: %v, count: %d, last: %v\n",
			ctx.users.String(userID),
			user.IsAuthorized,
			user.IsRoot,
			user.Counter,
			user.LastAccessTime.Format("2006-01-02 15:04:05"),
		)
	}

	return replayMsg
}

// all commands from command-line
func cmdUser(ctx Ctx) (replayMsg string) {
	if cmd, found := ctx.commands[ctx.messageCmd]; found {

		shell, params := "sh", []string{"-c", cmd}
		osExecCommand := exec.Command(shell, params...)
		osExecCommand.Stderr = os.Stderr

		// write all arguments to STDIN
		if ctx.messageArgs != "" {
			stdin, err := osExecCommand.StdinPipe()
			if err == nil {
				io.WriteString(stdin, ctx.messageArgs)
				stdin.Close()
			} else {
				log.Print("get STDIN error: ", err)
			}
		}

		shellOut, err := osExecCommand.Output()
		if err != nil {
			log.Print("exec error: ", err)
			replayMsg = fmt.Sprintf("exec error: %s", err)
		} else {
			replayMsg = string(shellOut)
		}
	}

	return replayMsg
}
