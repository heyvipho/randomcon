package main

import (
	"bytes"
	"encoding/gob"
	"log"
	"strconv"
	"strings"
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
	err := db.i.Update(func(txn *badger.Txn) error {
		key := []byte(DBPrefixes.Search)

		_, err := db.Get(txn, key)

		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}

			if err := db.Set(txn, key, DBSearch{}); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (db *DB) IsSearching(u DBUser) (bool, error) {
	yes := false

	err := db.i.View(func(txn *badger.Txn) error {
		key := []byte(DBPrefixes.Search)

		b, err := db.Get(txn, key)
		if err != nil {
			return err
		}

		search, err := db.BytesToSearch(b)
		if err != nil {
			return err
		}

		index := indexOfString(search, u.TBUser.Recipient())

		if index != -1 {
			yes = true
		}

		return nil
	})

	return yes, err
}

func (db *DB) Search(user DBUser) error {
	err := db.i.Update(func(txn *badger.Txn) error {
		key := []byte(DBPrefixes.Search)

		b, err := db.Get(txn, key)
		if err != nil {
			return err
		}

		search, err := db.BytesToSearch(b)
		if err != nil {
			return err
		}

		yes, err := db.IsSearching(user)
		if err != nil {
			return err
		}

		if yes {
			return ErrDBSearchAlreadyStarted
		}

		if len(search) > 0 {
			return db.AddRoom(user.TBUser.Recipient(), search[0])
		}

		search = append(search, user.TBUser.Recipient())

		if err := db.Set(txn, key, search); err != nil {
			return err
		}

		return nil
	})

	return err
}

func (db *DB) UnSearch(user DBUser) error {
	err := db.i.Update(func(txn *badger.Txn) error {
		key := []byte(DBPrefixes.Search)

		b, err := db.Get(txn, key)
		if err != nil {
			return err
		}

		search, err := db.BytesToSearch(b)
		if err != nil {
			return err
		}

		index := indexOfString(search, user.TBUser.Recipient())

		if index == -1 {
			return nil
		}

		search = append(search[:index], search[index+1:]...)

		if err := db.Set(txn, key, search); err != nil {
			return err
		}

		return nil
	})

	return err
}

func (db *DB) BytesToSearch(b []byte) (DBSearch, error) {
	var s DBSearch
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	if err := dec.Decode(&s); err != nil {
		return s, err
	}
	return s, nil
}

func (db *DB) AddRoom(users ...string) error {
	err := db.i.Update(func(txn *badger.Txn) error {
		roomNum := <-db.RoomCount

		key := db.P(DBPrefixes.Room, strconv.FormatUint(bytesToUint64(roomNum), 10))

		for _, v := range users {
			ukey := db.P(DBPrefixes.User, v)

			b, err := db.Get(txn, ukey)
			if err != nil {
				return err
			}

			user, err := db.BytesToUser(b)
			if err != nil {
				return err
			}

			if len(user.CurrentRoom) > 0 {
				rkey := db.P(DBPrefixes.Room, strconv.FormatUint(bytesToUint64(user.CurrentRoom), 10))
				if err := db.DelRoom(rkey); err != nil && err != badger.ErrKeyNotFound {
					return err
				}
			}

			user.CurrentRoom = roomNum

			if err := db.Set(txn, ukey, user); err != nil {
				return err
			}
		}

		room := DBRoom{
			Users: users,
		}
		if err := db.Set(txn, key, room); err != nil {
			return err
		}

		if err := tb.SearchingComplete(users, roomNum); err != nil {
			return err
		}

		return nil
	})

	return err
}

func (db *DB) BytesToRoom(b []byte) (DBRoom, error) {
	var s DBRoom
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	if err := dec.Decode(&s); err != nil {
		return s, err
	}
	return s, nil
}

func (db *DB) DelRoom(rkey []byte) error {
	err := db.i.Update(func(txn *badger.Txn) error {
		b, err := db.Get(txn, rkey)
		if err != nil {
			return err
		}

		room, err := db.BytesToRoom(b)
		if err != nil {
			return err
		}

		if err := txn.Delete(rkey); err != nil {
			return err
		}

		for _, v := range room.Users {
			ukey := db.P(DBPrefixes.User, v)

			b, err := db.Get(txn, ukey)
			if err != nil {
				return err
			}

			user, err := db.BytesToUser(b)
			if err != nil {
				return err
			}

			user.CurrentRoom = nil

			if err := db.Set(txn, ukey, user); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (db *DB) InitUser(tbUser telebot.User) (DBUser, error) {
	var user DBUser

	err := db.i.Update(func(txn *badger.Txn) error {
		key := db.P(DBPrefixes.User, tbUser.Recipient())
		b, err := db.Get(txn, key)

		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}

			newb, err := db.StructToBytes(DBUser{TBUser: tbUser})
			if err != nil {
				return err
			}

			if err := txn.Set(key, newb); err != nil {
				return err
			}

			b, err = db.Get(txn, key)
			if err != nil {
				return err
			}
		}

		user, err = db.BytesToUser(b)

		return err
	})

	if err != nil {
		return user, err
	}

	return user, nil
}

func (db *DB) BytesToUser(b []byte) (DBUser, error) {
	var s DBUser
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	if err := dec.Decode(&s); err != nil {
		return s, err
	}
	return s, nil
}

func (db *DB) StructToBytes(s interface{}) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(s); err != nil {
		return []byte{}, err
	}
	return buff.Bytes(), nil
}

func (db *DB) Get(txn *badger.Txn, key []byte) ([]byte, error) {
	item, err := txn.Get(key)
	if err != nil {
		return []byte{}, err
	}

	b, err := item.ValueCopy(nil)
	if err != nil {
		return []byte{}, err
	}

	return b, nil
}

func (db *DB) Set(txn *badger.Txn, key []byte, s interface{}) error {
	b, err := db.StructToBytes(s)
	if err != nil {
		return err
	}

	if err := txn.Set(key, b); err != nil {
		return err
	}

	return nil
}

// func (db *DB) Set(k []byte, v []byte) error {
// 	// Start a writable transaction.
// 	txn := db.i.NewTransaction(true)
// 	defer txn.Discard()

// 	// Use the transaction...
// 	err := txn.Set(k, v)
// 	if err != nil {
// 		return err
// 	}

// 	// Commit the transaction and check for error.
// 	if err := txn.Commit(); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (db *DB) Get(k []byte) ([]byte, error) {
// 	txn := db.i.NewTransaction(true)
// 	defer txn.Discard()

// 	item, err := txn.Get(k)
// 	if err != nil {
// 		return nil, err
// 	}

// 	v, err := item.ValueCopy(nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if err := txn.Commit(); err != nil {
// 		return nil, err
// 	}

// 	return v, nil
// }

// func (db *DB) Del(k []byte) error {
// 	txn := db.i.NewTransaction(true)
// 	defer txn.Discard()

// 	err := txn.Delete(k)
// 	if err != nil {
// 		return err
// 	}

// 	if err := txn.Commit(); err != nil {
// 		return err
// 	}

// 	return nil
// }

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
