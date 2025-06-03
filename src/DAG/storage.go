package dag

import (
	dag "_/D_/Sinar/Sinar_Chain/src/DAG"
	"fmt"
)

type Storage struct {
	db *leveldb.DB
}

func (s *Storage) SaveEvent(e *dag.Event) {
	key      := []byte("event_" + e.Hash)
	value, _ := gobEncode(e)
	s.db.Put(key, value, nil)
}