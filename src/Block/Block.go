package block

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"math/big"

	"../core"


)

type Block struct {
	Index     uint64
	Timestamp time.Time
	Events    []*core.Event
	TxRoot    []byte
	PrevHash  string
	Hash      string
	Signature []byte
	Validator string
}

func signBlock(b Block, privateKey *ecdsa.PrivateKey) []byte {
	hash := sha256.Sum256([]byte(b.Hash))
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil
	}
	return append(r.Bytes(), s.Bytes()...)
}

// BuildBlockFromEvents creates a block from a list of finalized events
func BuildBlockFromEvents(events []*core.Event, prevHash string, validator string, index uint64, privateKey []byte) Block {
	txs := collectAllTransactions(events)  // TODO: implement this function
	txRoot := hashTransactions(txs)        // TODO: implement this function

	block := Block{
		Index:     index,
		Timestamp: time.Now(),
		Events:    events,
		TxRoot:    txRoot,
		PrevHash:  prevHash,
		Validator: validator,
	}

	block.Hash = block.CalculateHash()
	block.Signature = signBlock(block, privateKey) // TODO: implement this function
	return block
}

// CalculateHash generates the hash of the block's metadata
func (b *Block) CalculateHash() string {
	record := strconv.FormatUint(b.Index, 10) +
		b.Timestamp.String() +
		string(b.TxRoot) +
		b.PrevHash +
		b.Validator

	hash := sha256.New()
	hash.Write([]byte(record))
	return hex.EncodeToString(hash.Sum(nil))
}

func collectAllTransactions(events []*core.Event) [][]byte {
	var txs [][]byte
	for _, ev := range events {
		for _, tx := range ev.Transactions {
			// فرض بر اینکه هر tx خودش []byte است
			txs = append(txs, tx)
		}
	}
	return txs
}

func hashTransactions(txs [][]byte) []byte {
	h := sha256.New()
	for _, tx := range txs {
		h.Write(tx)
	}
	return h.Sum(nil)
}

