package block

import "crypto/sha256"
import "encoding/hex"
import "strconv"
import "time"

type Block struct{
	Index       int
	Timestamp   string
	Data        string
	PrevHash    string
	Hash        string
	Nonce       int
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

func CreateGenesisBlock() Block {
	genesis:= Block{
		Index: 0,
		Timestamp: time.Now().String(),
		Data: "Genesis Block",
		PrevHash: "",
		Nonce: 0,
	}
	genesis.Hash = genesis.CalculateHash()
	return genesis
}


func main() {
    genesis := block.CreateGenesisBlock()
    fmt.Println("Genesis Block:", genesis)

    nextBlock := block.GenerateBlock(genesis, "My first transaction")
    fmt.Println("Next Block:", nextBlock)
}