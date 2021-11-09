package main

import "gopkg.in/tucnak/telebot.v2"

type DBUser struct {
	TBUser      telebot.User
	CurrentRoom []byte
	// RecentCons  []string
}

type DBRoom struct {
	Users []string
}

type DBSearch []string
