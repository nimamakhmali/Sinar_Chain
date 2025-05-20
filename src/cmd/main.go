package main

import  "fmt"
import  "Sinar_Chain/src/Block"
import "github.com/nimamakhmali/Sinar_Chain/core"


func main() {
    genesis := Block.CreateGenesisBlock()
    fmt.Println("Genesis Block:", genesis)

    nextBlock := Block.GenerateBlock(genesis, "My first transaction")
    fmt.Println("Next Block:", nextBlock)
}
