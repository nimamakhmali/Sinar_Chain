package core


import (
	"crypto/ecdsa"
	"fmt"
	"sync/atomic"
	
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
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
	//ID            string
	CreatorID     string    // ID or Address of the node
	Parents       []EventID // references to parent events
	Lamport       uint64    // Lamport timestamp (logical clock)
	Epoch         uint64    // epoch number
	ExtraData     []byte    // custom extra info
	TxHash        [32]byte  // hash of transactions
	TxCount       int
	Round         uint64       // برای الگوریتم اجماع DAG
	Frame         uint64       // (optional) برای دسته‌بندی منطقی
	Height        uint64       // بیشینه‌ی (height والدها) + 1
	RoundReceived uint64       // dar kodam round be ejmae reside?
	IsFamous      *bool        // is famous ?
	Atropos       EventID      // hash event nahaii ke ino taeed karde
	AtroposTime   uint64       // zaman ejmae nahaii
	MedianTime    uint64       // Average time of dag
	hash          atomic.Value // cached hash
	IsRoot        bool
	IsClotho      bool // ✅ نشان می‌دهد این رویداد Clotho شده

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

type DAG struct {
	Events map[EventID]*Event
}

func NewEvent(creatorID string, parents []EventID, epoch uint64, lamport uint64, txs []transaction, dag *DAG) *Event {
	height := uint64(0)
	for _, parentID := range parents {
		if parent, exists := dag.Events[parentID]; exists {
			if parent.Height > height {
				height = parent.Height
			}
		}
	}
	height += 1
	e := &Event{
		EventHeader: EventHeader{
			CreatorID: creatorID,
			Parents:   parents,
			Epoch:     epoch,
			Lamport:   lamport,
			TxCount:   len(txs),
			Height:    height,
		},
	}
	e.EventHeader.TxHash = e.CalculateTxHash()
	return e
}

func (e *Event) CalculateTxHash() [32]byte {
	hasher := sha3.NewLegacyKeccak256()
	txsBytes, err := rlp.EncodeToBytes(e.Transactions)
	if err != nil {
		panic(fmt.Errorf("Failed to encode transactions: %v", err))
	}
	hasher.Write(txsBytes)
	var txHash [32]byte
	copy(txHash[:], hasher.Sum(nil))
	return txHash
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
