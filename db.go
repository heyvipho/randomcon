package main

import (
	"bytes"
	"encoding/gob"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"gopkg.in/tucnak/telebot.v2"
)

var DBPrefixes = struct {
	User      string
	Search    string
	Room      string
	RoomCount string
}{
	User:      "user",
	Search:    "search",
	Room:      "room",
	RoomCount: "roomcount",
}

type DB struct {
	i         *badger.DB
	RoomCount chan []byte
	MuRoom    *sync.Mutex
	MuSearch  *sync.Mutex
}

func CreateDB() DB {
	db, err := badger.Open(badger.DefaultOptions(c.DBPath))
	if err != nil {
		log.Panic(err)
	}

	s := DB{
		i:         db,
		RoomCount: make(chan []byte),
	}

	if err := s.InitSearch(); err != nil {
		log.Panic(err)
	}

	go s.RoomIncrement()

	return s
}

func (db *DB) InitSearch() error {
	_, err := db.GetSearch()
	if err != nil {
		if err != badger.ErrKeyNotFound {
			return err
		}

		if err := db.SetSearch(DBSearch{}); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) IsSearching(u DBUser) (bool, error) {
	s, err := db.GetSearch()
	if err != nil {
		return false, err
	}

	index := indexOfString(s, u.TBUser.Recipient())

	if index == -1 {
		return false, nil
	}

	return true, nil
}

func (db *DB) Search(u DBUser) error {
	db.MuSearch.Lock()
	defer db.MuSearch.Unlock()

	s, err := db.GetSearch()
	if err != nil {
		return err
	}

	yes, err := db.IsSearching(u)
	if err != nil {
		return err
	}

	if yes {
		return nil
	}

	if len(s) > 0 {
		db.AddRoom(u.TBUser.Recipient(), s[0])
		return nil
	}

	s = append(s, u.TBUser.Recipient())

	if err := db.SetSearch(s); err != nil {
		return err
	}

	return nil
}

func (db *DB) UnSearch(u DBUser) error {
	db.MuSearch.Lock()
	defer db.MuSearch.Unlock()

	s, err := db.GetSearch()
	if err != nil {
		return err
	}

	index := indexOfString(s, u.TBUser.Recipient())

	if index == -1 {
		return nil
	}

	s = append(s[:index], s[index+1:]...)

	if err := db.SetSearch(s); err != nil {
		return err
	}

	return nil
}

func (db *DB) SetSearch(s DBSearch) error {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(s); err != nil {
		return err
	}
	key := []byte(DBPrefixes.Search)
	if err := db.Set(key, b.Bytes()); err != nil {
		return err
	}
	return nil
}

func (db *DB) GetSearch() (DBSearch, error) {
	key := []byte(DBPrefixes.Search)
	var s DBSearch
	b, err := db.Get(key)
	if err != nil {
		return s, err
	}
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	if err := dec.Decode(&s); err != nil {
		return s, err
	}
	return s, nil
}

func (db *DB) AddRoom(users ...string) error {
	db.MuRoom.Lock()
	defer db.MuRoom.Unlock()

	roomNum := <-db.RoomCount

	rkey := db.P(DBPrefixes.Room, strconv.FormatUint(bytesToUint64(roomNum), 10))

	for _, v := range users {
		ukey := db.P(DBPrefixes.User, v)

		u, err := db.GetUser(ukey)
		if err != nil {
			return err
		}

		rkey := db.P(DBPrefixes.Room, strconv.FormatUint(bytesToUint64(u.CurrentRoom), 10))
		if err := db.DelRoom(rkey); err != nil && err != badger.ErrKeyNotFound {
			return err
		}

		u.CurrentRoom = roomNum

		if err := db.SetUser(ukey, u); err != nil {
			return err
		}
	}

	room := DBRoom{
		users,
	}
	if err := db.SetRoom(rkey, room); err != nil {
		return err
	}

	return nil
}

func (db *DB) SetRoom(k []byte, s DBRoom) error {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(s); err != nil {
		return err
	}
	if err := db.Set(k, b.Bytes()); err != nil {
		return err
	}
	return nil
}

func (db *DB) GetRoom(k []byte) (DBRoom, error) {
	var s DBRoom
	b, err := db.Get(k)
	if err != nil {
		return s, err
	}
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	if err := dec.Decode(&s); err != nil {
		return s, err
	}
	return s, nil
}

func (db *DB) DelRoom(k []byte) error {
	room, err := db.GetRoom(k)
	if err != nil {
		return err
	}

	for _, v := range room.users {
		key := db.P(DBPrefixes.User, v)

		u, err := db.GetUser(key)
		if err != nil {
			return err
		}

		u.CurrentRoom = nil

		if err := db.SetUser(key, u); err != nil {
			return err
		}
	}

	if err := db.Del(k); err != nil {
		return err
	}

	return nil
}

func (db *DB) InitUser(u telebot.User) (DBUser, error) {
	key := db.P(DBPrefixes.User, u.Recipient())
	s, err := db.GetUser(key)
	if err != nil {
		if err != badger.ErrKeyNotFound {
			return s, err
		}

		if err := db.SetUser(key, DBUser{TBUser: u}); err != nil {
			return s, err
		}

		s, err = db.GetUser(key)
		if err != nil {
			return s, err
		}
	}
	return s, nil
}

func (db *DB) SetUser(k []byte, s DBUser) error {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(s); err != nil {
		return err
	}
	if err := db.Set(k, b.Bytes()); err != nil {
		return err
	}
	return nil
}

func (db *DB) GetUser(key []byte) (DBUser, error) {
	var s DBUser
	b, err := db.Get(key)
	if err != nil {
		return s, err
	}
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	if err := dec.Decode(&s); err != nil {
		return s, err
	}
	return s, nil
}

func (db *DB) Set(k []byte, v []byte) error {
	// Start a writable transaction.
	txn := db.i.NewTransaction(true)
	defer txn.Discard()

	// Use the transaction...
	err := txn.Set(k, v)
	if err != nil {
		return err
	}

	// Commit the transaction and check for error.
	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *DB) Get(k []byte) ([]byte, error) {
	txn := db.i.NewTransaction(true)
	defer txn.Discard()

	item, err := txn.Get(k)
	if err != nil {
		return nil, err
	}

	v, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return v, nil
}

func (db *DB) Del(k []byte) error {
	txn := db.i.NewTransaction(true)
	defer txn.Discard()

	err := txn.Delete(k)
	if err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *DB) RoomIncrement() {
	add := func(existing, new []byte) []byte {
		return uint64ToBytes(bytesToUint64(existing) + bytesToUint64(new))
	}

	key := []byte(DBPrefixes.RoomCount)

	m := db.i.GetMergeOperator(key, add, 200*time.Millisecond)
	defer m.Stop()

	for {
		m.Add(uint64ToBytes(1))

		res, err := m.Get()
		if err != nil {
			log.Fatal(err)
		}
		db.RoomCount <- res
	}
}

func (db *DB) P(s ...string) []byte {
	strings.Join(s, "-")
	return []byte(strings.Join(s, "-"))
}

func (db *DB) Close() {
	db.i.Close()
}
