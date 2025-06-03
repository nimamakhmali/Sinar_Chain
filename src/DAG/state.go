type State struct {
	db *leveldb.DB
}

func (s *State) ApplyTransaction(tx []byte) {
	// vm - evm
}
