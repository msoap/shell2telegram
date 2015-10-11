package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Syfaro/telegram-bot-api"
)

// User - one telegram user who interact with bot
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
	list                   map[int]*User
	predefinedAllowedUsers map[string]bool
	predefinedRootUsers    map[string]bool
}

// clear old users after 20 minutes after login
const SECONDS_FOR_OLD_USERS_BEFORE_VACUUM = 1200

// NewUsers - create Users object
func NewUsers(appConfig Config) Users {
	users := Users{
		list: map[int]*User{},
		predefinedAllowedUsers: map[string]bool{},
		predefinedRootUsers:    map[string]bool{},
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
func (users Users) AddNew(tgbotMessage tgbotapi.Message) {
	privateChatID := 0
	if !tgbotMessage.IsGroup() {
		privateChatID = tgbotMessage.Chat.ID
	}

	if _, ok := users.list[tgbotMessage.From.ID]; ok && privateChatID > 0 {
		users.list[tgbotMessage.From.ID].PrivateChatID = privateChatID
	} else if !ok {
		users.list[tgbotMessage.From.ID] = &User{
			UserName:      tgbotMessage.From.UserName,
			FirstName:     tgbotMessage.From.FirstName,
			LastName:      tgbotMessage.From.LastName,
			IsAuthorized:  users.predefinedAllowedUsers[tgbotMessage.From.UserName],
			IsRoot:        users.predefinedRootUsers[tgbotMessage.From.UserName],
			PrivateChatID: privateChatID,
		}
	}

	// collect stat
	users.list[tgbotMessage.From.ID].LastAccessTime = time.Now()
	if users.list[tgbotMessage.From.ID].IsAuthorized {
		users.list[tgbotMessage.From.ID].Counter++
	}
}

// DoLogin - generate secret code
func (users Users) DoLogin(userID int, forRoot bool) string {
	code := getRandomCode()
	if forRoot {
		users.list[userID].IsRoot = false
		users.list[userID].AuthCodeRoot = code
	} else {
		users.list[userID].IsAuthorized = false
		users.list[userID].AuthCode = code
	}
	return code
}

// SetAuthorized - set user authorized or authorized as root
func (users Users) SetAuthorized(userID int, forRoot bool) {
	users.list[userID].IsAuthorized = true
	users.list[userID].AuthCode = ""
	if forRoot {
		users.list[userID].IsRoot = true
		users.list[userID].AuthCodeRoot = ""
	}
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
			sendMessage(messageSignal, user.PrivateChatID, message)
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
func (users Users) ClearOldUsers() {
	for id, user := range users.list {
		if !user.IsAuthorized && !user.IsRoot && user.Counter == 0 &&
			time.Now().Sub(user.LastAccessTime).Seconds() > SECONDS_FOR_OLD_USERS_BEFORE_VACUUM {
			log.Printf("Vacuum: %d, %s", id, users.String(id))
			delete(users.list, id)
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
func (users Users) BanUser(userID int) bool {
	if _, ok := users.list[userID]; ok {
		users.list[userID].IsAuthorized = false
		users.list[userID].IsRoot = false
		if users.list[userID].UserName != "" {
			delete(users.predefinedAllowedUsers, users.list[userID].UserName)
			delete(users.predefinedRootUsers, users.list[userID].UserName)
		}
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
		sendMessage(messageSignal, user.PrivateChatID, message)
		return true
	}
	return false
}
