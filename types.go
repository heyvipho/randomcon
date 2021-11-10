package main

type DBUser struct {
	ID          int
	CurrentRoom []byte
	// RecentCons  []string
}

type DBRoom struct {
	Users []int
}

type DBSearch []int
