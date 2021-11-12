package main

import (
	"github.com/vipho/randomcon/database"
)

type DB interface {
	IsSearching(database.DBUser) (bool, error)
	Search(database.DBUser) ([]int, error)
	UnSearch(database.DBUser) error
	AddRoom([]int) (uint64, error)
	GetRoom(uint64) (database.DBRoom, error)
	GetUser(int) (database.DBUser, error)
	Close()
}

type Messages struct {
	Start                string
	SearchStarted        string
	SearchAlreadyStarted string
	SearchStopped        string
	RoomWelcome          string
	RoomNotInside        string
	MinorError           string
}
