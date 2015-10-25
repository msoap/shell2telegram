package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Syfaro/telegram-bot-api"
)

// User - one telegram user who interact with bot
type User struct {
	UserID         int       `json:"user_id"`          // telegram UserID
	UserName       string    `json:"user_name"`        // telegram @login
	FirstName      string    `json:"first_name"`       // telegram name
	LastName       string    `json:"last_name"`        // -//-
	AuthCode       string    `json:"auth_code"`        // code for authorize
	AuthCodeRoot   string    `json:"auth_code_root"`   // code for authorize root
	IsAuthorized   bool      `json:"is_authorized"`    // user allow chat with bot
	IsRoot         bool      `json:"is_root"`          // user is root, allow authorize/ban other users, remove commands, stop bot
	PrivateChatID  int       `json:"private_chat_id"`  // last private chat with bot
	Counter        int       `json:"counter"`          // how many commands send
	LastAccessTime time.Time `json:"last_access_time"` // time of last command
}

// Users in chat
type Users struct {
	list                   map[int]*User
	predefinedAllowedUsers map[string]bool
	predefinedRootUsers    map[string]bool
	needSaveDB             bool // non-saved changes in list
}

// UsersDB -  save list of Users into JSON
type UsersDB struct {
	Users    []User    `json:"users"`
	DateTime time.Time `json:"date_time"`
}

// SECONDS_FOR_OLD_USERS_BEFORE_VACUUM - clear old users after 20 minutes after login
const SECONDS_FOR_OLD_USERS_BEFORE_VACUUM = 1200

// NewUsers - create Users object
func NewUsers(appConfig Config) Users {
	users := Users{
		list: map[int]*User{},
		predefinedAllowedUsers: map[string]bool{},
		predefinedRootUsers:    map[string]bool{},
		needSaveDB:             true,
	}

	if appConfig.persistentUsers {
		users.LoadFromDB(appConfig.usersDB)
	}

	for _, name := range appConfig.predefinedAllowedUsers {
		users.predefinedAllowedUsers[name] = true
	}
	for _, name := range appConfig.predefinedRootUsers {
		users.predefinedAllowedUsers[name] = true
		users.predefinedRootUsers[name] = true
	}
	return users
}

// AddNew - add new user if not exists
func (users *Users) AddNew(tgbotMessage tgbotapi.Message) {
	privateChatID := 0
	if !tgbotMessage.IsGroup() {
		privateChatID = tgbotMessage.Chat.ID
	}

	if _, ok := users.list[tgbotMessage.From.ID]; ok && privateChatID > 0 && privateChatID != users.list[tgbotMessage.From.ID].PrivateChatID {
		users.list[tgbotMessage.From.ID].PrivateChatID = privateChatID
		users.needSaveDB = true
	} else if !ok {
		users.list[tgbotMessage.From.ID] = &User{
			UserID:        tgbotMessage.From.ID,
			UserName:      tgbotMessage.From.UserName,
			FirstName:     tgbotMessage.From.FirstName,
			LastName:      tgbotMessage.From.LastName,
			IsAuthorized:  users.predefinedAllowedUsers[tgbotMessage.From.UserName],
			IsRoot:        users.predefinedRootUsers[tgbotMessage.From.UserName],
			PrivateChatID: privateChatID,
		}
		users.needSaveDB = true
	}

	// collect stat
	users.list[tgbotMessage.From.ID].LastAccessTime = time.Now()
	if users.list[tgbotMessage.From.ID].IsAuthorized {
		users.list[tgbotMessage.From.ID].Counter++
	}
}

// DoLogin - generate secret code
func (users *Users) DoLogin(userID int, forRoot bool) string {
	code := getRandomCode()
	if forRoot {
		users.list[userID].IsRoot = false
		users.list[userID].AuthCodeRoot = code
	} else {
		users.list[userID].IsAuthorized = false
		users.list[userID].AuthCode = code
	}
	users.needSaveDB = true

	return code
}

// SetAuthorized - set user authorized or authorized as root
func (users *Users) SetAuthorized(userID int, forRoot bool) {
	users.list[userID].IsAuthorized = true
	users.list[userID].AuthCode = ""
	if forRoot {
		users.list[userID].IsRoot = true
		users.list[userID].AuthCodeRoot = ""
	}
	users.needSaveDB = true
}

// IsValidCode - check secret code for user
func (users Users) IsValidCode(userID int, code string, forRoot bool) bool {
	var result bool
	if forRoot {
		result = code != "" && code == users.list[userID].AuthCodeRoot
	} else {
		result = code != "" && code == users.list[userID].AuthCode
	}
	return result
}

// IsAuthorized - check user is authorized
func (users Users) IsAuthorized(userID int) bool {
	isAuthorized := false
	if _, ok := users.list[userID]; ok && users.list[userID].IsAuthorized {
		isAuthorized = true
	}

	return isAuthorized
}

// IsRoot - check user is root
func (users Users) IsRoot(userID int) bool {
	isRoot := false
	if _, ok := users.list[userID]; ok && users.list[userID].IsRoot {
		isRoot = true
	}

	return isRoot
}

// BroadcastForRoots - send message to all root users
func (users Users) BroadcastForRoots(messageSignal chan<- BotMessage, message string, excludeID int) {
	for userID, user := range users.list {
		if user.IsRoot && user.PrivateChatID > 0 && (excludeID == 0 || excludeID != userID) {
			sendMessage(messageSignal, user.PrivateChatID, []byte(message))
		}
	}
}

// String - format user name
func (users Users) String(userID int) string {
	result := fmt.Sprintf("%s %s", users.list[userID].FirstName, users.list[userID].LastName)
	if users.list[userID].UserName != "" {
		result += fmt.Sprintf(" (@%s)", users.list[userID].UserName)
	}
	return result
}

// StringVerbose - format user name with all fields
func (users Users) StringVerbose(userID int) string {
	user := users.list[userID]
	result := fmt.Sprintf("%s: id: %d, auth: %v, root: %v, count: %d, last: %v",
		users.String(userID),
		userID,
		user.IsAuthorized,
		user.IsRoot,
		user.Counter,
		user.LastAccessTime.Format("2006-01-02 15:04:05"),
	)
	return result
}

// ClearOldUsers - clear old users without login
func (users *Users) ClearOldUsers() {
	for id, user := range users.list {
		if !user.IsAuthorized && !user.IsRoot && user.Counter == 0 &&
			time.Now().Sub(user.LastAccessTime).Seconds() > SECONDS_FOR_OLD_USERS_BEFORE_VACUUM {
			log.Printf("Vacuum: %d, %s", id, users.String(id))
			delete(users.list, id)
			users.needSaveDB = true
		}
	}
}

// GetUserIDByName - find user by login
func (users Users) GetUserIDByName(userName string) int {
	userID := 0
	for id, user := range users.list {
		if userName == user.UserName {
			userID = id
			break
		}
	}

	return userID
}

// BanUser - ban user by ID
func (users *Users) BanUser(userID int) bool {

	if _, ok := users.list[userID]; ok {
		users.list[userID].IsAuthorized = false
		users.list[userID].IsRoot = false
		if users.list[userID].UserName != "" {
			delete(users.predefinedAllowedUsers, users.list[userID].UserName)
			delete(users.predefinedRootUsers, users.list[userID].UserName)
		}
		users.needSaveDB = true
		return true
	}

	return false
}

// Search - search users
func (users Users) Search(query string) (result []int) {
	queryUserID, _ := strconv.Atoi(query)
	query = strings.ToLower(query)
	queryAsLogin := cleanUserName(query)

	for userID, user := range users.list {
		if queryUserID == userID ||
			strings.Contains(strings.ToLower(user.UserName), queryAsLogin) ||
			strings.Contains(strings.ToLower(user.FirstName), query) ||
			strings.Contains(strings.ToLower(user.LastName), query) {
			result = append(result, userID)
		}
	}

	return result
}

// FindByIDOrUserName - find users or by ID or by @name
func (users Users) FindByIDOrUserName(userName string) int {
	userID, err := strconv.Atoi(userName)
	if err == nil {
		if _, ok := users.list[userID]; !ok {
			userID = 0
		}
	} else {
		userName = cleanUserName(userName)
		userID = users.GetUserIDByName(userName)
	}

	return userID
}

// SendMessageToPrivate - send message to user to private chat
func (users Users) SendMessageToPrivate(messageSignal chan<- BotMessage, userID int, message string) bool {
	if user, ok := users.list[userID]; ok && user.PrivateChatID > 0 {
		sendMessage(messageSignal, user.PrivateChatID, []byte(message))
		return true
	}
	return false
}

// LoadFromDB - load users list from json file
func (users *Users) LoadFromDB(usersDBFile string) {
	usersList := UsersDB{}

	fileNamePath := getDBFilePath(usersDBFile, false)
	usersJSON, err := ioutil.ReadFile(fileNamePath)
	if err == nil {
		if err = json.Unmarshal(usersJSON, &usersList); err == nil {
			for _, user := range usersList.Users {
				users.list[user.UserID] = &user
			}
		}
	}
	if err == nil {
		log.Printf("Loaded usersDB json from: %s", fileNamePath)
	} else {
		log.Printf("Load usersDB (%s) error: %s", fileNamePath, err)
	}

	users.needSaveDB = false
}

// SaveToDB - save users list to json file
func (users *Users) SaveToDB(usersDBFile string) {
	if users.needSaveDB {
		usersList := UsersDB{
			Users:    []User{},
			DateTime: time.Now(),
		}
		for _, user := range users.list {
			usersList.Users = append(usersList.Users, *user)
		}

		fileNamePath := getDBFilePath(usersDBFile, true)
		json, err := json.MarshalIndent(usersList, "", "  ")
		if err == nil {
			err = ioutil.WriteFile(fileNamePath, json, 0644)
		}

		if err == nil {
			log.Printf("Saved usersDB json to: %s", fileNamePath)
		} else {
			log.Printf("Save usersDB (%s) error: %s", fileNamePath, err)
		}

		users.needSaveDB = false
	}
}
