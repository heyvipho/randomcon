package main

import (
	"github.com/vipho/randomcon/database"
)

type DB interface {
	IsSearching(database.DBUser) (bool, error)
	Search(database.DBUser) ([]int, error)
	UnSearch(database.DBUser) error
	AddRoom([]int) (uint64, error)
	IsRoomExist(uint64) (bool, error)
	DelRoom(uint64) ([]int, error)
	GetRoom(uint64) (database.DBRoom, error)
	GetUser(int) (database.DBUser, error)
	Close()
}

type Messages struct {
	Start                    string
	SearchStarting           string
	SearchAlreadyStarted     string
	SearchStopped            string
	SearchYouAreNotSearching string
	RoomWelcome              string
	RoomDestroy              string
	RoomNotInside            string
	MinorError               string
}
