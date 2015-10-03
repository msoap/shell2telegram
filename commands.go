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
	appConfig   Config   // configuration
	commands    Commands // all chat commands
	users       Users    // all users
	userID      int      // current user
	allowExec   bool     // is user authorized
	messageCmd  string   // command name
	messageArgs string   // command arguments
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
	helpMsg := []string{}

	if ctx.allowExec {
		for cmd, shellCmd := range ctx.commands {
			helpMsg = append(helpMsg, cmd+" → "+shellCmd)
		}
		helpMsg = append(helpMsg,
			"/auth [code] → authorize user",
			"/authroot [code] → authorize user as root",
		)
		if ctx.users.IsRoot(ctx.userID) {
			helpMsg = append(helpMsg,
				"/shell2telegram stat → get stat about users",
				"/shell2telegram ban <user_id|username> → ban user",
			)
			if ctx.appConfig.addExit {
				helpMsg = append(helpMsg, "/shell2telegram exit → terminate bot")
			}
		}
	}

	helpMsg = append(helpMsg, "/shell2telegram version → show version")

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
	for userID, user := range ctx.users.list {
		replayMsg += fmt.Sprintf("%s: id: %d, auth: %v, root: %v, count: %d, last: %v\n",
			ctx.users.String(userID),
			userID,
			user.IsAuthorized,
			user.IsRoot,
			user.Counter,
			user.LastAccessTime.Format("2006-01-02 15:04:05"),
		)
	}

	return replayMsg
}

// /shell2telegram ban
func cmdShell2telegramBan(ctx Ctx) (replayMsg string) {
	_, userName := splitStringHalfBySpace(ctx.messageArgs)

	if userName == "" {
		return "Please set user_id or login"
	}

	userID, err := strconv.Atoi(userName)
	if err != nil {
		userName = regexp.MustCompile("@").ReplaceAllLiteralString(userName, "")
		userID = ctx.users.getUserIDByName(userName)
	}

	if userID > 0 && ctx.users.banUser(userID) {
		replayMsg = fmt.Sprintf("User %s banned", ctx.users.String(userID))
	} else {
		replayMsg = "User not found"
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
