package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Syfaro/telegram-bot-api"
)

// version
const VERSION = "1.0"

// bot default timeout
const DEFAULT_BOT_TIMEOUT = 60

// Command - one command type
type Commands map[string]string

// Config - config struct
type Config struct {
	token       string   // bot token
	addExit     bool     // add /exit command
	botTimeout  int      // bot timeout
	allowUsers  []string // users telegram-names who allow chats with bot
	rootUsers   []string // users telegram-names who confirm new users through of it private chat
	allowAll    bool     // allow all user (DANGEROUS!)
	logCommands bool     // logging all commands
}

// ----------------------------------------------------------------------------
// get config
func getConfig() (commands Commands, appConfig Config, err error) {
	flag.StringVar(&appConfig.token, "tb-token", "", "setting bot token (or set TB_TOKEN variable)")
	flag.BoolVar(&appConfig.addExit, "add-exit", false, "adding \"/shell2telegram exit\" command for terminate bot (for roots only)")
	flag.IntVar(&appConfig.botTimeout, "timeout", DEFAULT_BOT_TIMEOUT, "setting timeout for bot")
	flag.BoolVar(&appConfig.allowAll, "allow-all", false, "allow all users (DANGEROUS!)")
	flag.BoolVar(&appConfig.logCommands, "log-commands", false, "logging all commands")
	logFilename := flag.String("log", "", "log filename, default - STDOUT")
	allowUsers := flag.String("allow-users", "", "telegram users who are allowed to chat with the bot (\"user1,user2\")")
	rootUsers := flag.String("root-users", "", "telegram users, who confirms new users in their private chat (\"user1,user2\")")

	flag.Usage = func() {
		fmt.Printf("usage: %s [options] /chat_command \"shell command\" /chat_command2 \"shell command2\"\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}
	version := flag.Bool("version", false, "get version")
	flag.Parse()
	if *version {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	// setup log file
	if len(*logFilename) > 0 {
		fh_log, err := os.OpenFile(*logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("error opening log file: %v", err)
		}
		log.SetOutput(fh_log)
	}

	// setup users and roots
	if *allowUsers != "" {
		appConfig.allowUsers = strings.Split(*allowUsers, ",")
	}
	if *rootUsers != "" {
		appConfig.rootUsers = strings.Split(*rootUsers, ",")
	}

	commands = Commands{}
	// need >= 2 arguments and count of it must be even
	args := flag.Args()
	if len(args) < 2 || len(args)%2 == 1 {
		return commands, appConfig, fmt.Errorf("error: need pairs of chat-command and shell-command")
	}

	for i := 0; i < len(args); i += 2 {
		path, cmd := args[i], args[i+1]
		if path[0] != '/' {
			return commands, appConfig, fmt.Errorf("error: path %s dont starts with /", path)
		}
		commands[path] = cmd
	}

	if appConfig.token == "" {
		if appConfig.token = os.Getenv("TB_TOKEN"); appConfig.token == "" {
			return commands, appConfig, fmt.Errorf("TB_TOKEN environment var not found. See https://core.telegram.org/bots#botfather for more information\n")
		}
	}

	return commands, appConfig, nil
}

// ----------------------------------------------------------------------------
func sendMessageWithLogging(bot *tgbotapi.BotAPI, chatId int, replayMsg string) {
	_, err := bot.SendMessage(tgbotapi.NewMessage(chatId, replayMsg))
	if err != nil {
		log.Print("Bot send message error: ", err)
	}
}

// ----------------------------------------------------------------------------
func main() {
	commands, appConfig, err := getConfig()
	if err != nil {
		log.Fatal(err)
	}

	bot, err := tgbotapi.NewBotAPI(appConfig.token)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Authorized on bot account: %s", bot.Self.UserName)

	var tgbotConfig tgbotapi.UpdateConfig = tgbotapi.NewUpdate(0)
	tgbotConfig.Timeout = appConfig.botTimeout
	err = bot.UpdatesChan(tgbotConfig)
	if err != nil {
		log.Fatal(err)
	}

	doExit := false
	users := NewUsers(appConfig)
	vacuumTicker := time.Tick(SECONDS_FOR_OLD_USERS_BEFORE_VACUUM * time.Second)

LOOP:
	for {
		select {
		case telegramUpdate := <-bot.Updates:

			messageText := regexp.MustCompile(`\s+`).Split(telegramUpdate.Message.Text, 2)
			messageCmd, messageArgs := messageText[0], ""
			if len(messageText) > 1 {
				messageArgs = messageText[1]
			}

			replayMsg := ""

			if len(messageText) > 0 && len(messageCmd) > 0 && messageCmd[0] == '/' {

				userFrom := telegramUpdate.Message.From

				users.AddNew(telegramUpdate.Message)
				allowExec := appConfig.allowAll || users.IsAuthorized(userFrom.ID)

				// commands ---
				switch {
				case messageCmd == "/auth" || messageCmd == "/authroot":

					forRoot := messageCmd == "/authroot"

					if len(messageText) == 1 {

						replayMsg = "See code in terminal with shell2telegram or ask code from root user and type:\n" + messageCmd + " code"
						authCode := users.DoLogin(userFrom.ID, forRoot)

						rootRoleStr := ""
						if forRoot {
							rootRoleStr = "root "
						}
						secretCodeMsg := fmt.Sprintf("Request %saccess for %s. Code: %s\n", rootRoleStr, users.String(userFrom.ID), authCode)
						fmt.Print(secretCodeMsg)
						users.broadcastForRoots(bot, secretCodeMsg)

					} else if len(messageText) > 1 {
						if users.IsValidCode(userFrom.ID, messageArgs, forRoot) {
							users.list[userFrom.ID].IsAuthorized = true
							if forRoot {
								users.list[userFrom.ID].IsRoot = true
								replayMsg = fmt.Sprintf("You (%s) authorized as root.", users.String(userFrom.ID))
								log.Print("root authorized: ", users.String(userFrom.ID))
							} else {
								replayMsg = fmt.Sprintf("You (%s) authorized.", users.String(userFrom.ID))
								log.Print("authorized: ", users.String(userFrom.ID))
							}
						} else {
							replayMsg = fmt.Sprintf("Code is not valid.")
						}
					}

				case messageCmd == "/help":

					if allowExec {
						for cmd, shell_cmd := range commands {
							replayMsg += fmt.Sprintf("%s - %s\n", cmd, shell_cmd)
						}
						if users.IsRoot(userFrom.ID) {
							replayMsg += fmt.Sprintf("%s - %s\n", "/shell2telegram stat", "get stat about users")
							if appConfig.addExit {
								replayMsg += fmt.Sprintf("%s - %s\n", "/shell2telegram exit", "terminate bot")
							}
						}
					}
					replayMsg += fmt.Sprintf("%s - %s\n", "/auth [code]", "authorize user")
					replayMsg += fmt.Sprintf("%s - %s\n", "/authroot [code]", "authorize user as root")
					replayMsg += fmt.Sprintf("%s - %s\n", "/shell2telegram version", "show version")

				case messageCmd == "/shell2telegram" && messageArgs == "stat" && users.IsRoot(userFrom.ID):

					for userId, user := range users.list {
						replayMsg += fmt.Sprintf("%s: auth: %v, root: %v, count: %d, last: %v\n",
							users.String(userId),
							user.IsAuthorized,
							user.IsRoot,
							user.Counter,
							user.LastAccessTime.Format("2006-01-02 15:04:05"),
						)
					}

				case messageCmd == "/shell2telegram" && messageArgs == "exit" && users.IsRoot(userFrom.ID) && appConfig.addExit:

					replayMsg = "bye..."
					doExit = true

				case messageCmd == "/shell2telegram" && messageArgs == "version":

					replayMsg = fmt.Sprintf("shell2telegram %s", VERSION)

				case allowExec:
					if cmd, found := commands[messageCmd]; found {

						shell, params := "sh", []string{"-c", cmd}
						osExecCommand := exec.Command(shell, params...)
						osExecCommand.Stderr = os.Stderr

						// write all arguments to STDIN
						if messageArgs != "" {
							stdin, err := osExecCommand.StdinPipe()
							if err == nil {
								io.WriteString(stdin, messageArgs)
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
				} // switch for commands

				if replayMsg != "" {
					sendMessageWithLogging(bot, telegramUpdate.Message.Chat.ID, replayMsg)
					if appConfig.logCommands {
						log.Printf("%d @%s: %s", userFrom.ID, userFrom.UserName, telegramUpdate.Message.Text)
					}

					if doExit {
						break LOOP
					}
				}
			}

		case <-vacuumTicker:
			users.clearOldUsers()
		}
	}
}
