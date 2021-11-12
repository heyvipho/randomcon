package database

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"strconv"
	"strings"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v3"
)

var (
	ErrKeyNotFound = badger.ErrKeyNotFound
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
	i        *badger.DB
	muRoom   sync.Mutex
	muSearch sync.Mutex
}

func CreateDB(path string) (*DB, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return &DB{}, err
	}

	s := DB{
		i: db,
	}

	err = s.initSearch()

	return &s, err
}

func (db *DB) initSearch() error {
	_, err := db.getSearch()
	if err != nil {
		if err != badger.ErrKeyNotFound {
			return err
		}

		if err := db.setSearch(DBSearch{}); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) IsSearching(u DBUser) (bool, error) {
	s, err := db.getSearch()
	if err != nil {
		return false, err
	}

	index := db.indexOfInt(s, u.ID)

	if index == -1 {
		return false, nil
	}

	return true, nil
}

func (db *DB) Search(u DBUser) ([]int, error) {
	db.muSearch.Lock()
	defer db.muSearch.Unlock()

	users := []int{}

	s, err := db.getSearch()
	if err != nil {
		return users, err
	}

	yes, err := db.IsSearching(u)
	if err != nil {
		return users, err
	}

	if yes {
		return users, ErrDBSearchAlreadyStarted
	}

	if len(s) > 0 {
		users = s[:1]

		s = s[1:]

		err := db.setSearch(s)

		return users, err
	} else {
		s = append(s, u.ID)

		if err := db.setSearch(s); err != nil {
			return users, err
		}
	}

	return users, nil
}

func (db *DB) UnSearch(u DBUser) error {
	db.muSearch.Lock()
	defer db.muSearch.Unlock()

	s, err := db.getSearch()
	if err != nil {
		return err
	}

	index := db.indexOfInt(s, u.ID)

	if index == -1 {
		return nil
	}

	s = append(s[:index], s[index+1:]...)

	if err := db.setSearch(s); err != nil {
		return err
	}

	return nil
}

func (db *DB) setSearch(s DBSearch) error {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(s); err != nil {
		return err
	}
	key := []byte(DBPrefixes.Search)
	if err := db.set(key, b.Bytes()); err != nil {
		return err
	}
	return nil
}

func (db *DB) getSearch() (DBSearch, error) {
	key := []byte(DBPrefixes.Search)
	var s DBSearch
	b, err := db.get(key)
	if err != nil {
		return s, err
	}
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	if err := dec.Decode(&s); err != nil {
		return s, err
	}
	return s, nil
}

func (db *DB) AddRoom(users []int) (uint64, error) {
	db.muRoom.Lock()
	defer db.muRoom.Unlock()

	roomNum, err := db.roomIncrement()
	if err != nil {
		return 0, err
	}

	rkey := db.p(DBPrefixes.Room, strconv.FormatUint(roomNum, 10))

	for _, v := range users {
		ukey := db.p(DBPrefixes.User, strconv.Itoa(v))

		u, err := db.getUser(ukey)
		if err != nil {
			return 0, err
		}

		if u.CurrentRoom != 0 {
			if err := db.delRoom(u.CurrentRoom); err != nil && err != badger.ErrKeyNotFound {
				return 0, err
			}
		}

		u.CurrentRoom = roomNum

		if err := db.setUser(ukey, u); err != nil {
			return 0, err
		}

		if err := db.UnSearch(u); err != nil {
			return 0, err
		}
	}

	room := DBRoom{
		Users: users,
	}
	if err := db.setRoom(rkey, room); err != nil {
		return 0, err
	}

	return roomNum, nil
}

func (db *DB) setRoom(k []byte, s DBRoom) error {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(s); err != nil {
		return err
	}
	if err := db.set(k, b.Bytes()); err != nil {
		return err
	}
	return nil
}

func (db *DB) GetRoom(roomNum uint64) (DBRoom, error) {
	rkey := db.p(DBPrefixes.Room, strconv.FormatUint(roomNum, 10))

	var s DBRoom
	b, err := db.get(rkey)
	if err != nil {
		return s, err
	}
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	if err := dec.Decode(&s); err != nil {
		return s, err
	}
	return s, nil
}

func (db *DB) delRoom(roomNum uint64) error {
	room, err := db.GetRoom(roomNum)
	if err != nil {
		return err
	}

	rkey := db.p(DBPrefixes.Room, strconv.FormatUint(roomNum, 10))
	if err := db.del(rkey); err != nil {
		return err
	}

	for _, v := range room.Users {
		key := db.p(DBPrefixes.User, strconv.Itoa(v))

		u, err := db.getUser(key)
		if err != nil {
			return err
		}

		u.CurrentRoom = 0

		if err := db.setUser(key, u); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) GetUser(uID int) (DBUser, error) {
	key := db.p(DBPrefixes.User, strconv.Itoa(uID))
	s, err := db.getUser(key)
	if err != nil {
		if err != badger.ErrKeyNotFound {
			return s, err
		}

		if err := db.setUser(key, DBUser{ID: uID}); err != nil {
			return s, err
		}

		s, err = db.getUser(key)
		if err != nil {
			return s, err
		}
	}
	return s, nil
}

func (db *DB) setUser(k []byte, s DBUser) error {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(s); err != nil {
		return err
	}
	if err := db.set(k, b.Bytes()); err != nil {
		return err
	}
	return nil
}

func (db *DB) getUser(key []byte) (DBUser, error) {
	var s DBUser
	b, err := db.get(key)
	if err != nil {
		return s, err
	}
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	if err := dec.Decode(&s); err != nil {
		return s, err
	}
	return s, nil
}

func (db *DB) set(k []byte, v []byte) error {
	err := db.i.Update(func(txn *badger.Txn) error {
		err := txn.Set(k, v)
		return err
	})

	return err
}

func (db *DB) get(k []byte) ([]byte, error) {
	var bytes []byte

	err := db.i.View(func(txn *badger.Txn) error {
		item, err := txn.Get(k)
		if err != nil {
			return err
		}

		v, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		bytes = v

		return err
	})

	return bytes, err
}

func (db *DB) del(k []byte) error {
	err := db.i.Update(func(txn *badger.Txn) error {
		err := txn.Delete(k)
		return err
	})

	return err
}

func (db *DB) roomIncrement() (uint64, error) {
	add := func(existing, new []byte) []byte {
		return db.uint64ToBytes(db.bytesToUint64(existing) + db.bytesToUint64(new))
	}

	key := []byte(DBPrefixes.RoomCount)

	m := db.i.GetMergeOperator(key, add, 200*time.Millisecond)
	defer m.Stop()

	m.Add(db.uint64ToBytes(1))

	res, err := m.Get()

	return db.bytesToUint64(res), err
}

func (db *DB) p(s ...string) []byte {
	strings.Join(s, "-")
	return []byte(strings.Join(s, "-"))
}

func (db *DB) uint64ToBytes(i uint64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], i)
	return buf[:]
}

func (db *DB) bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

func (db *DB) indexOfInt(s []int, q int) int {
	for i, v := range s {
		if v == q {
			return i
		}
	}

	return -1
}

func (db *DB) Close() {
	db.i.Close()
}
