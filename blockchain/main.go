package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
)

var (
	addr = flag.String("addr", ":9981", "listened address")
)

// Block 代表一个区块
type Block struct {
	Index     int    // 区块高度
	Timestamp string // 时间戳
	Item      string // 数据项
	Hash      string // 区块的hash
	PrevHash  string // 前一个区块的hash
}

// FullBlockchain 合法的区块链
var FullBlockchain []Block

// Message 从http post body解析出要写入区块链的数据
type Message struct {
	Item string
}

func main() {
	go func() {
		t := time.Now()
		// 创世块
		genesisBlock := Block{0, t.String(), "0", "", ""}
		spew.Dump(genesisBlock)

		FullBlockchain = append(FullBlockchain, genesisBlock)
	}()
	log.Fatal(run())
}

// 检查区块是否合法，检查高度和哈希
func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

// 总是使用最长的区块链作为合法的区块链
func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(FullBlockchain) {
		FullBlockchain = newBlocks
	}
}

// 计算区块的 SHA256 hasing
func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + block.Item + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// 创建一个区块
func generateBlock(oldBlock Block, Item string) (Block, error) {

	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.Item = Item
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calculateHash(newBlock)

	return newBlock, nil
}
