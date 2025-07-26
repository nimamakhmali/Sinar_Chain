package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	// ساخت DAG جدید
	dag := NewDAG()

	// کلید خصوصی و عمومی تولید کن
	privKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	pubKey := &privKey.PublicKey

	// رویداد genesis (بدون والد)
	event1, err := dag.CreateAndAddEvent("NodeA", nil, 0, 1, types.Transactions{})
	if err != nil {
		log.Fatal(err)
	}
	event1.Sign(privKey)

	// رویداد دوم که والدش event1 هست
	event2, err := dag.CreateAndAddEvent("NodeB", []EventID{event1.Hash()}, 0, 2, types.Transactions{})
	if err != nil {
		log.Fatal(err)
	}
	event2.Sign(privKey)

	// نمایش خروجی
	fmt.Println("Event1 Hash:", event1.Hash())
	fmt.Println("Event2 Hash:", event2.Hash())

	// بررسی امضای دیجیتال
	fmt.Println("Event1 Valid Signature:", event1.VerifySignature(pubKey))
	fmt.Println("Event2 Valid Signature:", event2.VerifySignature(pubKey))
}
