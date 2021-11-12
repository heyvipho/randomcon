package database

type DBUser struct {
	ID          int
	CurrentRoom uint64
	// RecentCons  []string
}

type DBRoom struct {
	Users []int
}

type DBSearch []int
