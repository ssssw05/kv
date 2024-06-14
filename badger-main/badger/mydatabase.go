package main

import (
	"github.com/dgraph-io/badger/v4"
)

type MyDatabase struct {
	*badger.DB
}

func (db *MyDatabase) Set(key, value []byte) error {
	return db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func (db *MyDatabase) Get(key []byte) ([]byte, error) {
	var val []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			val = v
			return nil
		})
	})
	return val, err
}

func (db *MyDatabase) Delete(key []byte) error {
	return db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func NewMyDatabase(db *badger.DB) *MyDatabase {
	return &MyDatabase{DB: db}
}
