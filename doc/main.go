package main

import (
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"sinar_chain/src/event"
)

func main() {
	// ساخت کلید خصوصی نود
	privKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	pubKey := &privKey.PublicKey

	// شبیه‌سازی تراکنش‌ها
	txs := types.Transactions{
		types.NewTransaction(0, crypto.PubkeyToAddress(*pubKey), nil, 21000, nil, nil),
		types.NewTransaction(1, crypto.PubkeyToAddress(*pubKey), nil, 21000, nil, nil),
	}

	// ساخت DAG خالی
	dag := &event.DAG{Events: make(map[event.EventID]*event.Event)}

	// ساخت Event
	e := event.NewEvent("node-1", []event.EventID{}, 0, 1, nil, dag)
	e.Transactions = txs
	e.EventHeader.TxHash = e.CalculateTxHash()

	// امضا
	err = e.Sign(privKey)
	if err != nil {
		log.Fatal("sign failed:", err)
	}

	// بررسی امضا
	if e.VerifySignature(pubKey) {
		fmt.Println("Signature valid.")
	} else {
		fmt.Println("Signature invalid.")
	}

	// چاپ هش
	fmt.Printf("Event Hash: %x\n", e.Hash())
	fmt.Printf("TxHash:     %x\n", e.TxHash)
	fmt.Printf("TxCount:    %d\n", e.TxCount)
}
