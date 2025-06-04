package core

import (
	"bytes"
	"time"
	"crypto/ecdsa"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
	"github.com/ethereum/go-ethereum/core/types"

)

// Event Id is a 31-byte hash of the event
type EventID [32]byte

type transaction struct {
	From   string
	To     string
	Amount uint64
	Data   []byte
	// And more ...
}

// EventHeader contains metadata of the event
type EventHeader struct {
	Creator     string        // ID or Address of the node
	Parents     []EventID     // references to parent events
	Lamport     uint64        // Lamport timestamp (logical clock)
	Epoch       uint64        // epoch number
	ExtraData   []byte        // custom extra info
	TxHash      []byte        // hash of transactions
	TxCount     int
	Round       uint64        // برای الگوریتم اجماع DAG
	Frame       uint64        // (optional) برای دسته‌بندی منطقی
	Height      uint64        // بیشینه‌ی (height والدها) + 1
	hash        atomic.Value  // cached hash
}


// Event is a full structure containing all necessary data for DAG
type Event struct {
	EventHeader
	Transactions types.Transactions
	Signature    []byte
	Payload      [][]byte
	size         atomic.Value
}

type rlpEvent struct {
	Header       EventHeader
	Transactions types.Transactions
	Signature    []byte
	Payload      [][]byte
}


func NewEvent(creator string, parents []EventID, epoch uint64, lamport uint64, txs []transaction) *Event {
	height := uint64(0)
	return &Event {
		EventHeader: EventHeader{
			Creator:  creator,
			Parents:  parents,
			Epoch:    epoch,
			Lamport:  lamport,
			TxCount:  len(txs),
			Height: height,
		},
		Transactions: txs,
		Signature:    nil,
	}
}

func (e *Event) Hash() EventID {
	if h := e.hash.Load(); h != nil {
		return h.(EventID)
	}

	var id EventID
	hasher := sha3.NewLegacyKeccak256()

	// rlp with encoding header
	headerBytes, err := rlp.EncodeToBytes(e.EventHeader)
	if err != nil {
		panic(fmt.Errorf("Failed to RLP encode header: %v", err))
	}
	hasher.Write(headerBytes)

	// encoding transactions
	txsBytes, err := rlp.EncodeToBytes(e.Transactions)
	if err != nil {
		panic(fmt.Errorf("Failed to RLP encode transactions: %v", err))
	}
	hasher.Write(txsBytes)

	copy(id[:], hasher.Sum(nil))
	e.hash.Store(id)
	return id
}

func (e *Event) DataToSign() []byte {
	hash := e.Hash()
	return hash[:]
}


func (e *Event) Sign(priv *ecdsa.PrivateKey) error {

	data := crypto.Keccak256(e.DataToSign())
	sig, err := crypto.Sign(data, priv)
	if err != nil {
		return err
	}
	e.Signature = sig
	return nil
}

func (e *Event) VerifySignature(pub *ecdsa.PublicKey) bool {
	data := crypto.Keccak256(e.DataToSign())
	recovered, err := crypto.SigToPub(data, e.Signature)
	if err != nil {
		return false
	}
	return recovered.X.Cmp(pub.X) == 0 && recovered.Y.Cmp(pub.Y) == 0
}

func (e *Event) EncodeRLP() ([]byte, error) {
	enc := rlpEvent{
		Header:       e.EventHeader,
		Transactions: e.Transactions,
		Signature:    e.Signature,
		Payload:      e.Payload,
	}
	return rlp.EncodeToBytes(enc)
}

func DecodeRLP(data []byte) (*Event, error) {
	var dec rlpEvent
	err := rlp.DecodeBytes(data, &dec)
	if err != nil {
		return nil, err
	}

	return &Event{
		EventHeader:  dec.Header,
		Transactions: dec.Transactions,
		Signature:    dec.Signature,
		Payload:      dec.Payload,
	}, nil
}

