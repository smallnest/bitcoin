package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/smallnest/bitcoin/wallet/base58check"
	secp256k1 "github.com/toxeus/go-secp256k1"
)

var (
	privateKey       = flag.String("private-key", "", "The private key of the bitcoin wallet which contains the bitcoins you wish to send.")
	publicKey        = flag.String("public-key", "", "The public address of the bitcoin wallet which contains the bitcoins you wish to send.")
	destination      = flag.String("destination", "", "The public address of the bitcoin wallet to which you wish to send the bitcoins.")
	inputTransaction = flag.String("input-transaction", "", "An unspent input transaction hash which contains the bitcoins you wish to send. (Note: This program assumes a single input transaction, and a single output transaction for simplicity.)")
	inputIndex       = flag.Int("input-index", 0, "The output index of the unspent input transaction which contains the bitcoins you wish to send. Defaults to 0 (first index).")
	satoshis         = flag.Int("satoshis", 0, "The number of bitcoins you wish to send as represented in satoshis (100,000,000 satoshis = 1 bitcoin). (Important note: the number of satoshis left unspent in your input transaction will be spent as the transaction fee.)")
)

// https://zh-cn.bitcoin.it/wiki/Transactions
// go run transaction.go --private-key  5K5ib2WaTvqs4n3r1bMJLhDXg4CnV1We995UyECmbHLbzNnoTft --public-key 1K6KHeR4pRJLMcgb82Hmrg4RDhUZ2CaL2p -destination 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa --input-transaction  61ad94e4ad3b0cef86bbab2742f6946534ecbfd82153ce396c723cbbaa2a40fb -satoshis 1000

// https://bitcoin.org/en/developer-reference#raw-transaction-format
func main() {
	flag.Parse()

	tempScriptSig := createScriptPubKey(*publicKey)

	rawTransaction := createRawTransaction(*inputTransaction, *inputIndex, *destination, *satoshis, tempScriptSig)

	//After completing the raw transaction, we append
	//SIGHASH_ALL in little-endian format to the end of the raw transaction.
	hashCodeType, _ := hex.DecodeString("01000000")

	var rawTransBuffer bytes.Buffer
	rawTransBuffer.Write(rawTransaction)
	rawTransBuffer.Write(hashCodeType)
	rawTransWithHashCodeType := rawTransBuffer.Bytes()

	//Sign the raw transaction, and output it to the console.
	finalTransaction := signRawTransaction(rawTransWithHashCodeType, *privateKey)
	finalTransactionHex := hex.EncodeToString(finalTransaction)

	fmt.Println("Your final transaction is: ", finalTransactionHex)
}

func createScriptPubKey(publicKeyBase58 string) []byte {
	publicKeyBytes := base58check.Decode(publicKeyBase58)

	var scriptPubKey bytes.Buffer
	scriptPubKey.WriteByte(byte(118))                 //OP_DUP
	scriptPubKey.WriteByte(byte(169))                 //OP_HASH160
	scriptPubKey.WriteByte(byte(len(publicKeyBytes))) //PUSH
	scriptPubKey.Write(publicKeyBytes)
	scriptPubKey.WriteByte(byte(136)) //OP_EQUALVERIFY
	scriptPubKey.WriteByte(byte(172)) //OP_CHECKSIG
	return scriptPubKey.Bytes()
}

func signRawTransaction(rawTransaction []byte, privateKeyBase58 string) []byte {
	//Here we start the process of signing the raw transaction.

	secp256k1.Start()
	privateKeyBytes := base58check.Decode(privateKeyBase58)
	var privateKeyBytes32 [32]byte
	copy(privateKeyBytes32[:], privateKeyBytes)

	//Get the raw public key
	publicKeyBytes, success := secp256k1.Pubkey_create(privateKeyBytes32, false)
	if !success {
		log.Fatal("Failed to convert private key to public key")
	}

	//Hash the raw transaction twice before the signing
	shaHash := sha256.New()
	shaHash.Write(rawTransaction)
	var hash = shaHash.Sum(nil)

	shaHash2 := sha256.New()
	shaHash2.Write(hash)
	rawTransactionHashed := shaHash2.Sum(nil)

	var rawTransHashed32 [32]byte
	copy(rawTransHashed32[:], rawTransactionHashed)

	//Sign the raw transaction
	nonce := generateNonce()
	signedTransaction, success := secp256k1.Sign(rawTransHashed32, privateKeyBytes32, &nonce)
	if !success {
		log.Fatal("Failed to sign transaction")
	}

	//Verify that it worked.
	verified := secp256k1.Verify(rawTransHashed32, signedTransaction, publicKeyBytes)
	if !verified {
		log.Fatal("Failed to sign transaction")
	}

	secp256k1.Stop()

	hashCodeType, _ := hex.DecodeString("01")

	//+1 for hashCodeType
	signedTransactionLength := byte(len(signedTransaction) + 1)

	var publicKeyBuffer bytes.Buffer
	publicKeyBuffer.Write(publicKeyBytes)
	pubKeyLength := byte(len(publicKeyBuffer.Bytes()))

	var buffer bytes.Buffer
	buffer.WriteByte(signedTransactionLength)
	buffer.Write(signedTransaction)
	buffer.WriteByte(hashCodeType[0])
	buffer.WriteByte(pubKeyLength)
	buffer.Write(publicKeyBuffer.Bytes())

	scriptSig := buffer.Bytes()

	//Return the final transaction
	return createRawTransaction(*inputTransaction, *inputIndex, *destination, *satoshis, scriptSig)
}

func generateNonce() [32]byte {
	var bytes [32]byte
	for i := 0; i < 32; i++ {
		//This is not "cryptographically random"
		bytes[i] = byte(randInt(0, math.MaxUint8))
	}
	return bytes
}

func randInt(min int, max int) uint8 {
	rand.Seed(time.Now().UTC().UnixNano())
	return uint8(min + rand.Intn(max-min))
}

func createRawTransaction(inputTransactionHash string, inputTransactionIndex int, publicKeyBase58Destination string, satoshis int, scriptSig []byte) []byte {
	//Create the raw transaction.

	//Version field
	version, _ := hex.DecodeString("01000000")

	//# of inputs (always 1 in our case)
	inputs, _ := hex.DecodeString("01")

	//Input transaction hash
	inputTransactionBytes, err := hex.DecodeString(inputTransactionHash)
	if err != nil {
		log.Fatal(err)
	}

	//Convert input transaction hash to little-endian form
	inputTransactionBytesReversed := make([]byte, len(inputTransactionBytes))
	for i := 0; i < len(inputTransactionBytes); i++ {
		inputTransactionBytesReversed[i] = inputTransactionBytes[len(inputTransactionBytes)-i-1]
	}

	//Output index of input transaction
	outputIndexBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(outputIndexBytes, uint32(inputTransactionIndex))

	//Script sig length
	scriptSigLength := len(scriptSig)

	//sequence_no. Normally 0xFFFFFFFF. Always in this case.
	sequence, _ := hex.DecodeString("ffffffff")

	//Numbers of outputs for the transaction being created. Always one in this example.
	numOutputs, _ := hex.DecodeString("01")

	//Satoshis to send.
	satoshiBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(satoshiBytes, uint64(satoshis))

	//Script pub key
	scriptPubKey := createScriptPubKey(publicKeyBase58Destination)
	scriptPubKeyLength := len(scriptPubKey)

	//Lock time field
	lockTimeField, err := hex.DecodeString("00000000")
	if err != nil {
		log.Fatal(err)
	}

	var buffer bytes.Buffer
	buffer.Write(version)
	buffer.Write(inputs)
	buffer.Write(inputTransactionBytesReversed)
	buffer.Write(outputIndexBytes)
	buffer.WriteByte(byte(scriptSigLength))
	buffer.Write(scriptSig)
	buffer.Write(sequence)
	buffer.Write(numOutputs)
	buffer.Write(satoshiBytes)
	buffer.WriteByte(byte(scriptPubKeyLength))
	buffer.Write(scriptPubKey)
	buffer.Write(lockTimeField)

	return buffer.Bytes()
}
