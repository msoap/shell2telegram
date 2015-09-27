package main

// Users in chat
type User struct {
	UserName     string
	AuthCode     string
	IsAuthorized bool
}

type Users map[int]*User
