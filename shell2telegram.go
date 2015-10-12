package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Syfaro/telegram-bot-api"
)

// version
const VERSION = "1.2"

// bot default timeout
const DEFAULT_BOT_TIMEOUT = 60

// size of channel for bot messages
const MESSAGES_QUEUE_SIZE = 10

// max length of one bot message
const MAX_MESSAGE_LENGTH = 4096

// Command - one user command
type Command struct {
	shell       string   // shell command (/cmd)
	description string   // command description for list in /help (/cmd:desc="Command name")
	vars        []string // environment vars for user text, split by `/s+` to vars (/cmd:vars=SUBCOMMAND,ARGS)
}

// Commands - list of all commands
type Commands map[string]Command

// Config - config struct
type Config struct {
	token                  string   // bot token
	addExit                bool     // adding /shell2telegram exit command
	botTimeout             int      // bot timeout
	predefinedAllowedUsers []string // telegram users who are allowed to chat with the bot
	predefinedRootUsers    []string // telegram users, who confirms new users in their private chat
	allowAll               bool     // allow all user (DANGEROUS!)
	logCommands            bool     // logging all commands
	description            string   // description of bot
}

// BotMessage - record for send via channel for send message to telegram chat
type BotMessage struct {
	chatID  int
	message string
}

// ----------------------------------------------------------------------------
// get config
func getConfig() (commands Commands, appConfig Config, err error) {
	flag.StringVar(&appConfig.token, "tb-token", "", "setting bot token (or set TB_TOKEN variable)")
	flag.BoolVar(&appConfig.addExit, "add-exit", false, "adding \"/shell2telegram exit\" command for terminate bot (for roots only)")
	flag.IntVar(&appConfig.botTimeout, "timeout", DEFAULT_BOT_TIMEOUT, "setting timeout for bot")
	flag.BoolVar(&appConfig.allowAll, "allow-all", false, "allow all users (DANGEROUS!)")
	flag.BoolVar(&appConfig.logCommands, "log-commands", false, "logging all commands")
	flag.StringVar(&appConfig.description, "description", "", "setting description of bot")
	logFilename := flag.String("log", "", "log filename, default - STDOUT")
	predefinedAllowedUsers := flag.String("allow-users", "", "telegram users who are allowed to chat with the bot (\"user1,user2\")")
	predefinedRootUsers := flag.String("root-users", "", "telegram users, who confirms new users in their private chat (\"user1,user2\")")
	version := flag.Bool("version", false, "get version")

	flag.Usage = func() {
		fmt.Printf("usage: %s [options] %s\n%s\n%s\n\noptions:\n",
			os.Args[0],
			`/chat_command "shell command" /chat_command2 "shell command2"`,
			"All text after /chat_command will be sent to STDIN of shell command.",
			"If chat command is /:plain_text - get user message without any /command (for private chats only)",
		)
		flag.PrintDefaults()
		os.Exit(0)
	}
	flag.Parse()

	if *version {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	// setup log file
	if len(*logFilename) > 0 {
		fhLog, err := os.OpenFile(*logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("error opening log file: %v", err)
		}
		log.SetOutput(fhLog)
	}

	// setup users and roots
	if *predefinedAllowedUsers != "" {
		appConfig.predefinedAllowedUsers = strings.Split(*predefinedAllowedUsers, ",")
	}
	if *predefinedRootUsers != "" {
		appConfig.predefinedRootUsers = strings.Split(*predefinedRootUsers, ",")
	}

	commands = Commands{}
	// need >= 2 arguments and count of it must be even
	args := flag.Args()
	if len(args) < 2 || len(args)%2 == 1 {
		return commands, appConfig, fmt.Errorf("error: need pairs of /chat-command and shell-command")
	}

	for i := 0; i < len(args); i += 2 {
		path, command, err := parseBotCommand(args[i], args[i+1]) // (/path, shell_command)
		if err != nil {
			return commands, appConfig, err
		}
		commands[path] = command
	}

	if appConfig.token == "" {
		if appConfig.token = os.Getenv("TB_TOKEN"); appConfig.token == "" {
			return commands, appConfig, fmt.Errorf("TB_TOKEN environment var not found. See https://core.telegram.org/bots#botfather for more information\n")
		}
	}

	return commands, appConfig, nil
}

// ----------------------------------------------------------------------------
func sendMessage(messageSignal chan<- BotMessage, chatID int, message string) {
	go func() {
		messagesList := []string{message}

		if len(message) > MAX_MESSAGE_LENGTH {
			messagesList = splitStringLinesBySize(message, MAX_MESSAGE_LENGTH)
		}

		for _, messageChunk := range messagesList {
			messageSignal <- BotMessage{
				chatID:  chatID,
				message: messageChunk,
			}
		}
	}()
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

	log.Printf("Authorized on bot account: @%s", bot.Self.UserName)

	tgbotConfig := tgbotapi.NewUpdate(0)
	tgbotConfig.Timeout = appConfig.botTimeout
	err = bot.UpdatesChan(tgbotConfig)
	if err != nil {
		log.Fatal(err)
	}

	users := NewUsers(appConfig)
	messageSignal := make(chan BotMessage, MESSAGES_QUEUE_SIZE)
	vacuumTicker := time.Tick(SECONDS_FOR_OLD_USERS_BEFORE_VACUUM * time.Second)
	exitSignal := make(chan struct{})

	// all /shell2telegram sub-commands handlers
	internalCommands := map[string]func(Ctx) string{
		"stat":              cmdShell2telegramStat,
		"ban":               cmdShell2telegramBan,
		"search":            cmdShell2telegramSearch,
		"desc":              cmdShell2telegramDesc,
		"rm":                cmdShell2telegramRm,
		"exit":              cmdShell2telegramExit,
		"version":           cmdShell2telegramVersion,
		"broadcast_to_root": cmdShell2telegramBroadcastToRoot,
		"message_to_user":   cmdShell2telegramMessageToUser,
	}

	doExit := false
	for !doExit {
		select {
		case telegramUpdate := <-bot.Updates:

			messageCmd, messageArgs := splitStringHalfBySpace(telegramUpdate.Message.Text)
			replayMsg := ""

			allowPlainText := false
			if _, ok := commands["/:plain_text"]; ok {
				allowPlainText = true
			}

			if len(messageCmd) > 0 && (messageCmd[0] == '/' || allowPlainText) {

				users.AddNew(telegramUpdate.Message)
				userID := telegramUpdate.Message.From.ID
				allowExec := appConfig.allowAll || users.IsAuthorized(userID)

				ctx := Ctx{
					appConfig:     &appConfig,
					commands:      commands,
					users:         users,
					userID:        userID,
					allowExec:     allowExec,
					allMessage:    telegramUpdate.Message.Text,
					messageCmd:    messageCmd,
					messageArgs:   messageArgs,
					messageSignal: messageSignal,
					chatID:        telegramUpdate.Message.Chat.ID,
					exitSignal:    exitSignal,
				}

				switch {
				// commands .................................
				case messageCmd == "/auth" || messageCmd == "/authroot":
					replayMsg = cmdAuth(ctx)

				case messageCmd == "/help":
					replayMsg = cmdHelp(ctx)

				case messageCmd == "/shell2telegram" && users.IsRoot(userID):
					messageSubCmd, messageArgs := splitStringHalfBySpace(messageArgs)
					ctx.messageArgs = messageArgs
					if cmdHandler, ok := internalCommands[messageSubCmd]; ok {
						replayMsg = cmdHandler(ctx)
					} else {
						replayMsg = "Sub-command not found"
					}

				case allowExec && allowPlainText && messageCmd[0] != '/':
					// send message from goroutine after exec shell command
					_ = cmdPlainText(ctx)

				case allowExec:
					_ = cmdUser(ctx)

				} // switch for commands

				if appConfig.logCommands {
					log.Printf("%s: %s", users.String(userID), telegramUpdate.Message.Text)
				}

				sendMessage(messageSignal, telegramUpdate.Message.Chat.ID, replayMsg)
			}

		case botMessage := <-messageSignal:
			if !stringIsEmpty(botMessage.message) {
				_, err := bot.SendMessage(tgbotapi.NewMessage(botMessage.chatID, botMessage.message))
				if err != nil {
					log.Print("Bot send message error: ", err)
				}
			}

		case <-vacuumTicker:
			users.ClearOldUsers()

		case <-exitSignal:
			doExit = true
		}
	}
}
