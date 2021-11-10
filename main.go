package main

import "log"

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	var err error

	c, err := CreateConfig()
	if err != nil {
		log.Panic(err)
	}

	db, err := CreateDB(c)
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()

	tb, err := CreateTB(c, db)
	if err != nil {
		log.Panic(err)
	}
	tb.Start()
}
