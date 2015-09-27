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
	token      string   // bot token
	addExit    bool     // add /exit command
	botTimeout int      // bot timeout
	allowUsers []string // users telegram-names who allow chats with bot
	rootUsers  []string // users telegram-names who confirm new users through of it private chat
}

// ----------------------------------------------------------------------------
// get config
func getConfig() (commands Commands, app_config Config, err error) {
	flag.StringVar(&app_config.token, "tb-token", "", "set bot token (or set TB_TOKEN variable)")
	flag.BoolVar(&app_config.addExit, "add-exit", false, "add /exit command for terminate bot")
	flag.IntVar(&app_config.botTimeout, "timeout", DEFAULT_BOT_TIMEOUT, "bot timeout")
	allowUsers := flag.String("allow-users", "", "users telegram-names who allow chats with bot (\"user1,user2\")")
	rootUsers := flag.String("root-users", "", "users telegram-names who confirm new users through of it private chat (\"user1,user2\")")

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

	if *allowUsers != "" {
		app_config.allowUsers = strings.Split(*allowUsers, ",")
	}
	if *rootUsers != "" {
		app_config.rootUsers = strings.Split(*rootUsers, ",")
	}

	commands = Commands{}
	// need >= 2 arguments and count of it must be even
	args := flag.Args()
	if len(args) < 2 || len(args)%2 == 1 {
		return commands, app_config, fmt.Errorf("error: need pairs of chat-command and shell-command")
	}

	for i := 0; i < len(args); i += 2 {
		path, cmd := args[i], args[i+1]
		if path[0] != '/' {
			return commands, app_config, fmt.Errorf("error: path %s dont starts with /", path)
		}
		commands[path] = cmd
	}

	if app_config.token == "" {
		if app_config.token = os.Getenv("TB_TOKEN"); app_config.token == "" {
			return commands, app_config, fmt.Errorf("TB_TOKEN environment var not found. See https://core.telegram.org/bots#botfather for more information\n")
		}
	}

	return commands, app_config, nil
}

// ----------------------------------------------------------------------------
func sendMessageWithLogging(bot *tgbotapi.BotAPI, chat_id int, replay_msg string) {
	_, err := bot.SendMessage(tgbotapi.NewMessage(chat_id, replay_msg))
	if err != nil {
		log.Print("Bot send message error: ", err)
	}
}

// ----------------------------------------------------------------------------
func main() {
	commands, app_config, err := getConfig()
	if err != nil {
		log.Fatal(err)
	}

	bot, err := tgbotapi.NewBotAPI(app_config.token)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Authorized on bot account: %s", bot.Self.UserName)

	var ucfg tgbotapi.UpdateConfig = tgbotapi.NewUpdate(0)
	ucfg.Timeout = app_config.botTimeout
	err = bot.UpdatesChan(ucfg)
	if err != nil {
		log.Fatal(err)
	}

	go_exit := false
	users := NewUsers(app_config)

LOOP:
	for {
		select {
		case telegram_update := <-bot.Updates:

			chat_id := telegram_update.Message.Chat.ID

			parts := regexp.MustCompile(`\s+`).Split(telegram_update.Message.Text, 2)
			replay_msg := ""

			if len(parts) > 0 && len(parts[0]) > 0 && parts[0][0] == '/' {

				user_from := telegram_update.Message.From

				users.AddNew(user_from, telegram_update.Message.Chat)
				allowExec := users.IsAuthorized(user_from.ID)

				if parts[0] == "/auth" {

					if len(parts) == 1 || parts[1] == "" {

						replay_msg = "See code in terminal with shell2telegram or ack code from root user and type:\n/auth code"
						users.DoLogin(user_from.ID)

						secretCodeMsg := fmt.Sprintf("Request access for %s. Code: %s\n", users.String(user_from.ID), users.list[user_from.ID].AuthCode)
						fmt.Print(secretCodeMsg)
						users.broadcastForRoots(bot, secretCodeMsg)

					} else if len(parts) > 1 {
						if users.IsValidCode(user_from.ID, parts[1]) {
							replay_msg = fmt.Sprintf("You (%s) authorized.", users.String(user_from.ID))
							users.list[user_from.ID].IsAuthorized = true
						} else {
							replay_msg = fmt.Sprintf("Code is not valid.")
						}
					}

				} else if parts[0] == "/help" {

					if allowExec {
						for cmd, shell_cmd := range commands {
							replay_msg += fmt.Sprintf("%s - %s\n", cmd, shell_cmd)
						}
						if app_config.addExit {
							replay_msg += fmt.Sprintf("%s - %s\n", "/exit", "terminate bot")
						}
					}
					if users.IsRoot(user_from.ID) {
						replay_msg += fmt.Sprintf("%s - %s\n", "/shell2telegram stat", "get stat about users")
					}
					replay_msg += fmt.Sprintf("%s - %s\n", "/auth [code]", "authorize user")

				} else if allowExec && users.IsRoot(user_from.ID) && parts[0] == "/shell2telegram" && len(parts) > 1 && parts[1] == "stat" {

					for user_id, user := range users.list {
						replay_msg += fmt.Sprintf("%s: auth: %v, root: %v, count: %d, last: %v\n",
							users.String(user_id),
							user.IsAuthorized,
							user.IsRoot,
							user.Counter,
							user.LastAccessTime.Format("2006-01-02 15:04:05"),
						)
					}

				} else if allowExec && app_config.addExit && parts[0] == "/exit" {

					replay_msg = "bye..."
					go_exit = true

				} else if cmd, found := commands[parts[0]]; allowExec && found {

					shell, params := "sh", []string{"-c", cmd}
					if len(parts) > 1 {
						params = append(params, parts[1])
					}

					os_exec_command := exec.Command(shell, params...)
					os_exec_command.Stderr = os.Stderr

					// write all arguments to STDIN
					if len(parts) > 1 && parts[1] != "" {
						stdin, err := os_exec_command.StdinPipe()
						if err == nil {
							io.WriteString(stdin, parts[1])
							stdin.Close()
						} else {
							log.Print("get STDIN error: ", err)
						}
					}

					shell_out, err := os_exec_command.Output()
					if err != nil {
						log.Println("exec error: ", err)
						replay_msg = fmt.Sprintf("exec error: %s", err)
					} else {
						replay_msg = string(shell_out)
					}
				}

				if replay_msg != "" {
					sendMessageWithLogging(bot, chat_id, replay_msg)

					if go_exit {
						break LOOP
					}
				}
			}
		}
	}
}
