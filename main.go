package main

import "log"

var c Config
var tb TB
var db DB

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	c = CreateConfig()

	db = CreateDB()
	defer db.Close()

	tb = CreateTB()
	tb.Start()
}
