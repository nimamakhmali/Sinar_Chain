package block

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"time"
)

type Block struct{
	Index        int
	Timestamp    int64
	Data         string
	PrevHash     string
	Hash         string
	Nonce        int
	Difficulity  int
}

func (b *Block) CalculateHash() string {
	record := strconv.Itoa(b,Index) + b.Timestamp + b.Data + b.PrevHash + strconv.Itoa((b, Nonce))
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

func GenerateBlock(prevBlock Block, data string) Block {
	newBlock := Block {
		Index: PrevBlock.Index + 1,
		Timestamp: time.Now().String(),
		Data: data,
		PrevHash: prevBlock.Hash,
		Nonce: 0,
	}
	newBlock.Hash = newBlock.CalculateHash()
	return newBlock
}

func CreateGenesisBlock(difficulity int) Block {
	genesis:= Block{
		Index: 0,
		Timestamp: time.Now().Unix(),
		Data: "Genesis Block",
		PrevHash: "",
		Nonce: 0,
		Difficulity: difficulity,
	}
	genesis.Hash = genesis.CalculateHash()
	return genesis
}

func generateNewBlockWithPow(PrevHash Block, data string, difficulity int) Block{
	var nonce int 
	timestamp := time.Now().Unix()
	newBlock := Block{
		Index: prevBlock.Index +1,
		Timestamp: timestamp,
		PrevHash: PrevHash.Hash,
		Data: data,
		Nonce: 0,
		Difficulity: difficulity,
	}

	target := big.NewInt(1)
	target.Lsh(target, uint(256-difficulity))
	for nonce < math.MaxInt64 {
		newBlock.Nonce = nonce
		hash := CalculateHash(newBlock)
		hashInt := new(big.Int)
		hashInt.SetString(hash, 16)
		if hashInt.Cmp(target) == -1 {
			newBlock.Hash = hash
			break
		} else {
			nonce++
		}
	}
	return newBlock
}

func CreateGenesisBlockForPos(difficulity int) Block {
	timestamp := time.Now().Unix()
	genesisBlock := Block {
		Index: 0,
		Timestamp: timestamp,
		PrevHash: 0,
		Data: "Genesis Block",
		Nonce: 0,
		Difficulity: difficulity,
	}
	genesisBlock.Hash = CalculateHash(genesisBlock)
	return genesisBlock
}

func generateNewBlockWithPos(prevBlock Block, data string, difficulity int, validators []string) Block {
	timestamp := time.Now().Unix()
	newBlock := Block{
		Index: prevBlock.Index +1,
		Timestamp: timestamp,
		PrevHash: prevBlock.Hash,
		Data: data,
		Nonce: 0,
		Difficulity: difficulity,
	}
	rand.Seed(time.Now().UnixNano())
	validatorsIndex := rand.Intn(len(validators))
	validators.Hash = CalculateHash(newBlock + validator)
	return newBlock
}