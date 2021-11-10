package main

import (
	"github.com/vipho/randomcon/database"
)

type DB interface {
	IsSearching(database.DBUser) (bool, error)
	Search(database.DBUser) ([]int, error)
	UnSearch(database.DBUser) error
	AddRoom([]int) (uint64, error)
	GetUser(int) (database.DBUser, error)
	Close()
}

type Messages struct {
	Start string
}
