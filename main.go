package main

import (
	"log"
	"path"

	"github.com/vipho/randomcon/database"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	var err error

	c, err := CreateConfig()
	if err != nil {
		log.Panic(err)
	}

	db, err := database.CreateDB(path.Join(c.DataPath, "db"))
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()

	tb, err := CreateTB(c.TBToken, &c.TBMessages, db)
	if err != nil {
		log.Panic(err)
	}
	tb.Start()
}
