package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"

	"github.com/Syfaro/telegram-bot-api"
)

// version
const VERSION = "1.0"

// Command - one command type
type Commands map[string]string

// Config - config struct
type Config struct {
	token      string // bot token
	addExit    bool   // add /exit command
	botTimeout int    // bot timeout
}

// ------------------------------------------------------------------
// get config
func getConfig() (commands Commands, app_config Config, err error) {
	flag.StringVar(&app_config.token, "tb-token", "", "set bot token (or set TB_TOKEN variable)")
	flag.BoolVar(&app_config.addExit, "add-exit", false, "add /exit command for terminate bot")
	flag.IntVar(&app_config.botTimeout, "timeout", 60, "bot timeout")

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
			return commands, app_config, fmt.Errorf("TB_TOKEN env var not found. See https://core.telegram.org/bots#botfather for more information\n")
		}
	}

	return commands, app_config, nil
}

// ------------------------------------------------------------------
func main() {
	commands, app_config, err := getConfig()
	if err != nil {
		log.Fatal(err)
	}

	bot, err := tgbotapi.NewBotAPI(app_config.token)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	var ucfg tgbotapi.UpdateConfig = tgbotapi.NewUpdate(0)
	ucfg.Timeout = app_config.botTimeout
	err = bot.UpdatesChan(ucfg)

	go_exit := false
	users := make(Users)

LOOP:
	for {
		select {
		case telegram_update := <-bot.Updates:

			chat_id := telegram_update.Message.Chat.ID

			parts := regexp.MustCompile(`\s+`).Split(telegram_update.Message.Text, 2)
			replay_msg := ""

			if len(parts) > 0 && len(parts[0]) > 0 && parts[0][0] == '/' {

				user_from := telegram_update.Message.From
				allowExec := users.IsAuthorized(user_from.ID)

				if parts[0] == "/auth" {

					users.AddNew(user_from.ID, user_from.UserName)

					if len(parts) == 1 || parts[1] == "" {
						replay_msg = "See code in terminal with shell2telegram and type:\n/auth code"
						users.DoLogin(user_from.ID)
						fmt.Printf("Code (for %s): %s\n", user_from.UserName, users[user_from.ID].AuthCode)
					} else if len(parts) > 1 {
						if users.IsValidCode(user_from.ID, parts[1]) {
							replay_msg = fmt.Sprintf("You (%s) authorized.", user_from.UserName)
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
					replay_msg += fmt.Sprintf("%s - %s\n", "/auth [code]", "authorize user")

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
					shell_out, err := os_exec_command.Output()
					if err != nil {
						log.Println("exec error: ", err)
						replay_msg = fmt.Sprintf("exec error: %s", err)
					} else {
						replay_msg = string(shell_out)
					}
				}

				if replay_msg != "" {
					bot.SendMessage(tgbotapi.NewMessage(chat_id, replay_msg))
					if go_exit {
						break LOOP
					}
				}
			}
		}
	}
}
