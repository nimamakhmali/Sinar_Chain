// Lachesis Module 1: Event Handling
package main

import (
	"crypto/ecdsa"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

type EventID [32]byte

type EventHeader struct {
	CreatorID     string
	Parents       []EventID
	Lamport       uint64
	Epoch         uint64
	ExtraData     []byte
	TxHash        [32]byte
	TxCount       int
	Round         uint64
	Frame         uint64
	Height        uint64
	RoundReceived uint64
	IsFamous      *bool
	Atropos       EventID
	AtroposTime   uint64
	MedianTime    uint64
	IsRoot        bool
	IsClotho      bool
	hash          atomic.Value
}

type Event struct {
	EventHeader
	Transactions types.Transactions
	Signature    []byte
	Payload      [][]byte
}

type rlpEvent struct {
	Header       EventHeader
	Transactions types.Transactions
	Signature    []byte
	Payload      [][]byte
}

func NewEvent(creatorID string, parents []EventID, epoch, lamport uint64, txs types.Transactions, height uint64) *Event {
	e := &Event{
		EventHeader: EventHeader{
			CreatorID: creatorID,
			Parents:   parents,
			Epoch:     epoch,
			Lamport:   lamport,
			TxCount:   len(txs),
			Height:    height,
		},
		Transactions: txs,
	}
	e.TxHash = e.CalculateTxHash()
	return e
}

func (e *Event) CalculateTxHash() [32]byte {
	hasher := sha3.NewLegacyKeccak256()
	txsBytes, err := rlp.EncodeToBytes(e.Transactions)
	if err != nil {
		panic(fmt.Errorf("tx encoding failed: %v", err))
	}
	hasher.Write(txsBytes)
	var hash [32]byte
	copy(hash[:], hasher.Sum(nil))
	return hash
}

func (e *Event) Hash() EventID {
	if h := e.hash.Load(); h != nil {
		return h.(EventID)
	}
	var id EventID
	headerBytes, _ := rlp.EncodeToBytes(e.EventHeader)
	txsBytes, _ := rlp.EncodeToBytes(e.Transactions)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(headerBytes)
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
	re := rlpEvent{
		Header:       e.EventHeader,
		Transactions: e.Transactions,
		Signature:    e.Signature,
		Payload:      e.Payload,
	}
	return rlp.EncodeToBytes(re)
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

// Reset بازنشانی event برای استفاده مجدد در pool
func (e *Event) Reset() {
	// پاک کردن تمام فیلدها
	e.CreatorID = ""
	e.Parents = nil
	e.Lamport = 0
	e.Epoch = 0
	e.ExtraData = nil
	e.TxHash = [32]byte{}
	e.TxCount = 0
	e.Round = 0
	e.Frame = 0
	e.Height = 0
	e.RoundReceived = 0
	e.IsFamous = nil
	e.Atropos = EventID{}
	e.AtroposTime = 0
	e.MedianTime = 0
	e.IsRoot = false
	e.IsClotho = false
	e.Transactions = nil
	e.Signature = nil
	e.Payload = nil

	// پاک کردن hash cache
	e.hash.Store(EventID{})
}

// Note: This is only the first module. After you review it, I will continue to generate the next 8 modules (DAG, Round Assignment, Fame Voting, Clotho Selection, Atropos Finality, Time Consensus, Block Packaging, Serialization).
