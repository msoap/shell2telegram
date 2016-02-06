package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/koding/cache"
	tgbotapi "gopkg.in/telegram-bot-api.v1"
)

const (
	// VERSION - current version
	VERSION = "1.2"

	// DEFAULT_BOT_TIMEOUT - bot default timeout
	DEFAULT_BOT_TIMEOUT = 60

	// MESSAGES_QUEUE_SIZE - size of channel for bot messages
	MESSAGES_QUEUE_SIZE = 10

	// MAX_MESSAGE_LENGTH - max length of one bot message
	MAX_MESSAGE_LENGTH = 4096

	// SECONDS_FOR_AUTO_SAVE_USERS_TO_DB - save users to file every 1 min (if need)
	SECONDS_FOR_AUTO_SAVE_USERS_TO_DB = 60

	// DB_FILE_NAME - DB json name
	DB_FILE_NAME = "shell2telegram.json"
)

// Command - one user command
type Command struct {
	shellCmd    string   // shell command
	description string   // command description for list in /help (/cmd:desc="Command name")
	vars        []string // environment vars for user text, split by `/s+` to vars (/cmd:vars=SUBCOMMAND,ARGS)
	isMarkdown  bool     // send message in markdown format
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
	persistentUsers        bool     // load/save users from file
	usersDB                string   // file for store users
	cache                  int      // caching command out (in seconds)
}

// message types
const (
	msgIsText int8 = iota
	msgIsPhoto
)

// BotMessage - record for send via channel for send message to telegram chat
type BotMessage struct {
	chatID      int
	messageType int8
	message     string
	fileName    string
	photo       []byte
	isMarkdown  bool
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
	flag.BoolVar(&appConfig.persistentUsers, "persistent_users", false, "load/save users from file (default ~/.config/shell2telegram.json)")
	flag.StringVar(&appConfig.usersDB, "users_db", "", "file for store users")
	flag.IntVar(&appConfig.cache, "cache", 0, "caching command out (in seconds)")
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
func sendMessage(messageSignal chan<- BotMessage, chatID int, message []byte, isMarkdown bool) {
	go func() {
		fileName := ""
		fileType := http.DetectContentType(message)
		switch fileType {
		case "image/png":
			fileName = "file.png"
		case "image/jpeg":
			fileName = "file.jpeg"
		case "image/gif":
			fileName = "file.gif"
		case "image/bmp":
			fileName = "file.bmp"
		default:
			fileName = "message"
		}

		if fileName == "message" {

			// is text message
			messageString := string(message)
			messagesList := []string{}

			if len(messageString) <= MAX_MESSAGE_LENGTH {
				messagesList = []string{messageString}
			} else {
				messagesList = splitStringLinesBySize(messageString, MAX_MESSAGE_LENGTH)
			}

			for _, messageChunk := range messagesList {
				messageSignal <- BotMessage{
					chatID:      chatID,
					messageType: msgIsText,
					message:     messageChunk,
					isMarkdown:  isMarkdown,
				}
			}

		} else {
			// is image
			messageSignal <- BotMessage{
				chatID:      chatID,
				messageType: msgIsPhoto,
				fileName:    fileName,
				photo:       message,
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
	botUpdatesChan, err := bot.GetUpdatesChan(tgbotConfig)
	if err != nil {
		log.Fatal(err)
	}

	users := NewUsers(appConfig)
	messageSignal := make(chan BotMessage, MESSAGES_QUEUE_SIZE)
	vacuumTicker := time.Tick(SECONDS_FOR_OLD_USERS_BEFORE_VACUUM * time.Second)
	saveToBDTicker := make(<-chan time.Time)
	exitSignal := make(chan struct{})
	systemExitSignal := make(chan os.Signal)
	signal.Notify(systemExitSignal, os.Interrupt, os.Kill)

	if appConfig.persistentUsers {
		saveToBDTicker = time.Tick(SECONDS_FOR_AUTO_SAVE_USERS_TO_DB * time.Second)
	}

	var cacheTTL *cache.MemoryTTL
	if appConfig.cache > 0 {
		cacheTTL = cache.NewMemoryWithTTL(time.Duration(appConfig.cache) * time.Second)
		cacheTTL.StartGC(time.Duration(appConfig.cache) * time.Second * 2)
	}

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
		case telegramUpdate := <-botUpdatesChan:

			var messageCmd, messageArgs string
			allUserMessage := telegramUpdate.Message.Text
			if len(allUserMessage) > 0 && allUserMessage[0] == '/' {
				messageCmd, messageArgs = splitStringHalfBySpace(allUserMessage)
			} else {
				messageCmd, messageArgs = "/:plain_text", allUserMessage
			}

			allowPlainText := false
			if _, ok := commands["/:plain_text"]; ok {
				allowPlainText = true
			}

			replayMsg := ""

			if len(messageCmd) > 0 && (messageCmd != "/:plain_text" || allowPlainText) {

				users.AddNew(telegramUpdate.Message)
				userID := telegramUpdate.Message.From.ID
				allowExec := appConfig.allowAll || users.IsAuthorized(userID)

				ctx := Ctx{
					appConfig:     &appConfig,
					users:         &users,
					commands:      commands,
					userID:        userID,
					allowExec:     allowExec,
					messageCmd:    messageCmd,
					messageArgs:   messageArgs,
					messageSignal: messageSignal,
					chatID:        telegramUpdate.Message.Chat.ID,
					exitSignal:    exitSignal,
					cacheTTL:      cacheTTL,
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

				case allowExec && (allowPlainText && messageCmd == "/:plain_text" || messageCmd[0] == '/'):
					cmdUser(ctx)

				} // switch for commands

				if appConfig.logCommands {
					log.Printf("%s: %s", users.String(userID), allUserMessage)
				}

				sendMessage(messageSignal, telegramUpdate.Message.Chat.ID, []byte(replayMsg), false)
			}

		case botMessage := <-messageSignal:
			switch {
			case botMessage.messageType == msgIsText && !stringIsEmpty(botMessage.message):
				messageConfig := tgbotapi.NewMessage(botMessage.chatID, botMessage.message)
				if botMessage.isMarkdown {
					messageConfig.ParseMode = tgbotapi.ModeMarkdown
				}
				_, err = bot.Send(messageConfig)
			case botMessage.messageType == msgIsPhoto && len(botMessage.photo) > 0:
				bytesPhoto := tgbotapi.FileBytes{Name: botMessage.fileName, Bytes: botMessage.photo}
				_, err = bot.Send(tgbotapi.NewPhotoUpload(botMessage.chatID, bytesPhoto))
			}

			if err != nil {
				log.Print("Bot send message error: ", err)
			}

		case <-saveToBDTicker:
			users.SaveToDB(appConfig.usersDB)

		case <-vacuumTicker:
			users.ClearOldUsers()

		case <-systemExitSignal:
			go func() {
				exitSignal <- struct{}{}
			}()

		case <-exitSignal:
			if appConfig.persistentUsers {
				users.needSaveDB = true
				users.SaveToDB(appConfig.usersDB)
			}
			doExit = true
		}
	}
}
