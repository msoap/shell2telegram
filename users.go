package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
)

// Users in chat
type User struct {
	UserName     string
	AuthCode     string
	IsAuthorized bool
}

type Users map[int]*User

// length of random code in bytes
const CODE_BYTES_LENGTH = 15

// add new user if not exists
func (users Users) AddNew(user_id int, user_name string) {
	if _, ok := users[user_id]; !ok {
		users[user_id] = &User{
			UserName:     user_name,
			AuthCode:     "",
			IsAuthorized: false,
		}
	}
}

// generate code
func (users Users) DoLogin(user_id int) {
	users[user_id].IsAuthorized = false
	users[user_id].AuthCode = getRandomCode()
}

// check code for user
func (users Users) IsValidCode(user_id int, code string) bool {
	result := code != "" && code == users[user_id].AuthCode
	if result {
		users[user_id].IsAuthorized = true
	}

	return result
}

// check code for user
func (users Users) IsAuthorized(user_id int) bool {
	allowExec := false
	if _, ok := users[user_id]; ok {
		allowExec = users[user_id].IsAuthorized
	}

	return allowExec
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
