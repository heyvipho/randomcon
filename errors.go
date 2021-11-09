package main

import "errors"

var (
	ErrDBSearchAlreadyStarted = errors.New("Search already started")
)
