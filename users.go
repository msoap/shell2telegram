package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/Syfaro/telegram-bot-api"
)

// One User
type User struct {
	UserName       string
	FirstName      string
	LastName       string
	AuthCode       string
	AuthCodeRoot   string
	IsAuthorized   bool
	IsRoot         bool
	PrivateChatID  int
	Counter        int
	LastAccessTime time.Time
}

// Users in chat
type Users struct {
	list         map[int]*User
	allowedUsers map[string]bool
	rootUsers    map[string]bool
}

// length of random code in bytes
const CODE_BYTES_LENGTH = 15

// clear old users after 20 minutes after login
const SECONDS_FOR_OLD_USERS_BEFORE_VACUUM = 1200

// new Users object
func NewUsers(appConfig Config) Users {
	users := Users{
		list:         map[int]*User{},
		allowedUsers: map[string]bool{},
		rootUsers:    map[string]bool{},
	}

	for _, name := range appConfig.allowUsers {
		users.allowedUsers[name] = true
	}
	for _, name := range appConfig.rootUsers {
		users.allowedUsers[name] = true
		users.rootUsers[name] = true
	}
	return users
}

// add new user if not exists
func (users Users) AddNew(tgbot_message tgbotapi.Message) {
	privateChatID := 0
	if !tgbot_message.IsGroup() {
		privateChatID = tgbot_message.Chat.ID
	}

	if _, ok := users.list[tgbot_message.From.ID]; ok && privateChatID > 0 {
		users.list[tgbot_message.From.ID].PrivateChatID = privateChatID
	} else if !ok {
		users.list[tgbot_message.From.ID] = &User{
			UserName:      tgbot_message.From.UserName,
			FirstName:     tgbot_message.From.FirstName,
			LastName:      tgbot_message.From.LastName,
			IsAuthorized:  users.allowedUsers[tgbot_message.From.UserName],
			IsRoot:        users.rootUsers[tgbot_message.From.UserName],
			PrivateChatID: privateChatID,
		}
	}

	// collect stat
	users.list[tgbot_message.From.ID].LastAccessTime = time.Now()
	if users.list[tgbot_message.From.ID].IsAuthorized {
		users.list[tgbot_message.From.ID].Counter++
	}
}

// generate code
func (users Users) DoLogin(user_id int, for_root bool) {
	if for_root {
		users.list[user_id].IsRoot = false
		users.list[user_id].AuthCodeRoot = getRandomCode()
	} else {
		users.list[user_id].IsAuthorized = false
		users.list[user_id].AuthCode = getRandomCode()
	}
}

// check code for user
func (users Users) IsValidCode(user_id int, code string, for_root bool) bool {
	var result bool
	if for_root {
		result = code != "" && code == users.list[user_id].AuthCodeRoot
	} else {
		result = code != "" && code == users.list[user_id].AuthCode
	}
	return result
}

// check user is authorized
func (users Users) IsAuthorized(user_id int) bool {
	isAuthorized := false
	if _, ok := users.list[user_id]; ok && users.list[user_id].IsAuthorized {
		isAuthorized = true
	}

	return isAuthorized
}

// check user is root
func (users Users) IsRoot(user_id int) bool {
	isRoot := false
	if _, ok := users.list[user_id]; ok && users.list[user_id].IsRoot {
		isRoot = true
	}

	return isRoot
}

// send message to all root users
func (users Users) broadcastForRoots(bot *tgbotapi.BotAPI, message string) {
	for _, user := range users.list {
		if user.IsRoot && user.PrivateChatID > 0 {
			sendMessageWithLogging(bot, user.PrivateChatID, message)
		}
	}
}

// Format user name
func (users Users) String(user_id int) string {
	result := fmt.Sprintf("%s %s", users.list[user_id].FirstName, users.list[user_id].LastName)
	if users.list[user_id].UserName != "" {
		result += fmt.Sprintf(" (@%s)", users.list[user_id].UserName)
	}
	return result
}

// clear old users without login
func (users Users) clearOldUsers() {
	for id, user := range users.list {
		if !user.IsAuthorized && !user.IsRoot && user.Counter == 0 &&
			time.Now().Sub(user.LastAccessTime).Seconds() > SECONDS_FOR_OLD_USERS_BEFORE_VACUUM {
			log.Printf("Vacuum: %d, %s", id, users.String(id))
			delete(users.list, id)
		}
	}
}

// generate random code for authorize user
func getRandomCode() string {
	buffer := make([]byte, CODE_BYTES_LENGTH)
	_, err := rand.Read(buffer)
	if err != nil {
		log.Print("Get code error: ", err)
		return ""
	}

	return base64.URLEncoding.EncodeToString(buffer)
}
