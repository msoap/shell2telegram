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
func (users Users) AddNew(tgbot_user tgbotapi.User, tgbot_chat tgbotapi.UserOrGroupChat) {
	privateChatID := 0
	if tgbot_chat.Title == "" {
		privateChatID = tgbot_chat.ID
	}

	if _, ok := users.list[tgbot_user.ID]; ok && privateChatID > 0 {
		users.list[tgbot_user.ID].PrivateChatID = privateChatID
	} else if !ok {
		users.list[tgbot_user.ID] = &User{
			UserName:      tgbot_user.UserName,
			FirstName:     tgbot_user.FirstName,
			LastName:      tgbot_user.LastName,
			IsAuthorized:  users.allowedUsers[tgbot_user.UserName],
			IsRoot:        users.rootUsers[tgbot_user.UserName],
			PrivateChatID: privateChatID,
		}
	}

	// collect stat
	users.list[tgbot_user.ID].LastAccessTime = time.Now()
	if users.list[tgbot_user.ID].IsAuthorized {
		users.list[tgbot_user.ID].Counter++
	}
}

// generate code
func (users Users) DoLogin(user_id int) {
	users.list[user_id].IsAuthorized = false
	users.list[user_id].AuthCode = getRandomCode()
}

// check code for user
func (users Users) IsValidCode(user_id int, code string) bool {
	result := code != "" && code == users.list[user_id].AuthCode
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
