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

func main() {
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

	// need >= 2 arguments and count of it must be even
	args := flag.Args()
	if len(args) < 2 || len(args)%2 == 1 {
		log.Fatal("error: need pairs of path and shell command")
	}

	cmd_handlers := map[string]string{}

	for i := 0; i < len(args); i += 2 {
		path, cmd := args[i], args[i+1]
		if path[0] != '/' {
			log.Fatalf("error: path %s dont starts with /", path)
		}
		cmd_handlers[path] = cmd
	}

	tb_token := os.Getenv("TB_TOKEN")
	if tb_token == "" {
		log.Fatal("TB_TOKEN env var not found. See https://core.telegram.org/bots#botfather for more information\n")
	}

	bot, err := tgbotapi.NewBotAPI(tb_token)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	var ucfg tgbotapi.UpdateConfig = tgbotapi.NewUpdate(0)
	ucfg.Timeout = 60
	err = bot.UpdatesChan(ucfg)

	for {
		select {
		case update := <-bot.Updates:

			chat_id := update.Message.Chat.ID

			parts := regexp.MustCompile(`\s+`).Split(update.Message.Text, 2)
			if len(parts) > 0 {
				if parts[0] == "/help" {
					reply := ""
					for cmd, shell_cmd := range cmd_handlers {
						reply += fmt.Sprintf("%s\n %s\n", cmd, shell_cmd)
					}
					msg := tgbotapi.NewMessage(chat_id, reply)
					bot.SendMessage(msg)
				}

				if cmd, found := cmd_handlers[parts[0]]; found {
					shell, params := "sh", []string{"-c", cmd}
					if len(parts) > 1 {
						params = append(params, parts[1])
					}

					os_exec_command := exec.Command(shell, params...)
					os_exec_command.Stderr = os.Stderr
					shell_out, err := os_exec_command.Output()
					if err != nil {
						log.Println("exec error: ", err)
						reply := fmt.Sprintf("exec error: %s", err)
						msg := tgbotapi.NewMessage(chat_id, reply)
						bot.SendMessage(msg)
					} else {
						reply := string(shell_out)
						msg := tgbotapi.NewMessage(chat_id, reply)
						bot.SendMessage(msg)
					}
				}
			}
		}
	}
}
