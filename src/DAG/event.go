package dag

import (
	"crypto"
	"crypto/rand"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"time"
)

type Event struct {
	Creator   string // hash
	Index     uint64   // number for builders node
	Parents   []string // hash parent
	Timestamp time.Time
	Payload   []byte
	Signature []byte
	Hash      string
}

func (e *Event) ComputeHash() string {
	hasher := sha256.New()
	binary.Write(hasher, binary.BigEndian, e.Index)
	for _, p := range e.Parents {
		hasher.Write([]byte(p))
	}
	hasher.Write([]byte(e.Timestamp.Format(time.RFC3339Nano)))
	hasher.Write(e.Payload)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (e * Event) Sign(privKey *ecdsa.PrivateKey) error {
	hash, _ := hex.DecodeString(e.ComputeHash())
	sig, err := crypto.Sign(hash, privKey)
	if err == nil {
		e.Signature = sig
	}
	return err
}