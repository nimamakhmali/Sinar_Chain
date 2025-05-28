package dag

import "crypto/sha256"
import "encoding/hex"
import "time"

type Event struct {
	Creator   string // hash
	Index     uint64   // number for builders node
	Parents   []string // hash parent
	Timestamp time.Time
	TxData    [][]byte
	Signature []byte
	Hash      string
}

func (e *Event) ComputeHash() string {
	hasher := sha256.New()
	hasher.Write([]byte(e.Creator))
	hasher.Write([]byte(string(e.Index)))
	for _, p := range e.Parents {
		hasher.Write([]byte(p))
	}
	hasher.Write([]byte(e.Timestamp.String()))
	for _, tx := range e.TxData {
		hasher.Write(tx)
	}
	sum := hasher.Sum(nil)
	return hex.EncodeToString(sum)
}