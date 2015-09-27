package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"

	"github.com/Syfaro/telegram-bot-api"
)

// One User
type User struct {
	UserName     string
	AuthCode     string
	IsAuthorized bool
}

// Users in chat
type Users struct {
	list             map[int]*User
	listAllowedUsers map[string]bool
}

// length of random code in bytes
const CODE_BYTES_LENGTH = 15

// new Users object
func NewUsers(allowUsers []string) Users {
	users := Users{
		list:             map[int]*User{},
		listAllowedUsers: map[string]bool{},
	}

	for _, name := range allowUsers {
		users.listAllowedUsers[name] = true
	}
	return users
}

// add new user if not exists
func (users Users) AddNew(user_id int, user_name string) {
	if _, ok := users.list[user_id]; !ok {
		users.list[user_id] = &User{
			UserName:     user_name,
			AuthCode:     "",
			IsAuthorized: users.listAllowedUsers[user_name],
		}
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
	if result {
		users.list[user_id].IsAuthorized = true
	}

	return result
}

// check code for user
func (users Users) IsAuthorized(tgbot_user tgbotapi.User) bool {
	isAuthorized := false
	if tgbot_user.UserName != "" && users.listAllowedUsers[tgbot_user.UserName] {
		isAuthorized = true
	} else if _, ok := users.list[tgbot_user.ID]; ok && users.list[tgbot_user.ID].IsAuthorized {
		isAuthorized = true
	}

	return isAuthorized
}

// generate random code for authorize user
func getRandomCode() string {
	buffer := make([]byte, CODE_BYTES_LENGTH)
	_, err := rand.Read(buffer)
	if err != nil {
		log.Fatal("Get code error:", err)
	}

	return base64.URLEncoding.EncodeToString(buffer)
}
