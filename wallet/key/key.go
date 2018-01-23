package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os/exec"
	"runtime"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	"github.com/smallnest/bitcoin/wallet/base58check"
	secp256k1 "github.com/toxeus/go-secp256k1"
	"golang.org/x/crypto/ripemd160"
)

var (
	testnet = flag.Bool("testnet", false, "Whether or not to use the bitcoin testnet. (optional, defaults false)")
)

// A Bitcoin wallet can refer to either a wallet program or a wallet file.
// Wallet programs create public keys to receive satoshis and use the corresponding private keys to spend those satoshis. // Wallet files store private keys and (optionally) other information related to transactions for the wallet program.
//
// This program is a part of wallet program: it generates private keys, derives the corresponding public keys.
func main() {
	flag.Parse()

	var privateKeyPrefix string
	var publicKeyPrefix string

	if *testnet {
		privateKeyPrefix = "EF"
		publicKeyPrefix = "6F"
	} else {
		privateKeyPrefix = "80"
		publicKeyPrefix = "00"
	}

	// Private keys are what are used to unlock satoshis from a particular address.
	// In Bitcoin, a private key in standard format is simply a 256-bit number, between the values:
	//
	// 0x01 and 0xFFFF FFFF FFFF FFFF FFFF FFFF FFFF FFFE BAAE DCE6 AF48 A03B BFD2 5E8C D036 4140,
	// representing nearly the entire range of 2256-1 values.
	// The range is governed by the secp256k1 ECDSA encryption standard used by Bitcoin.
	privateKey := generatePrivateKey()

	// 1. Take a private key.
	// 2. Add a 0x80 byte in front of it for mainnet addresses or 0xef for testnet addresses.
	// 3. Append a 0x01 byte after it if it should be used with compressed public keys. Nothing is appended if it is used with uncompressed public keys.
	// 4. Perform a SHA-256 hash on the extended key.
	// 5. Perform a SHA-256 hash on result of SHA-256 hash.
	// 6. Take the first four bytes of the second SHA-256 hash; this is the checksum.
	// 7. Add the four checksum bytes from point 5 at the end of the extended key from point 2.
	// 8. Convert the result from a byte string into a Base58 string using Base58Check encoding.
	privateKeyWif := base58check.Encode(privateKeyPrefix, privateKey)

	// Bitcoin addresses, which are base58-encoded strings containing an address version number, the hash,
	// and an error-detection checksum to catch typos.
	// A 20-byte hash formatted using base58check to produce either a P2PKH or P2SH Bitcoin address.
	// Currently the most common way users exchange payment information.

	publicKey := generatePublicKey(privateKey)
	//There is also a prefix on the public key
	//This is known as the Network ID Byte, or the version byte
	//6f is the testnet prefix
	//00 is the mainnet prefix
	publicKeyEncoded := base58check.Encode(publicKeyPrefix, publicKey)

	//Print the keys
	fmt.Println("Your private key is")
	fmt.Println(privateKeyWif)

	fmt.Println("Your address is")
	fmt.Println(publicKeyEncoded)

	// Display address info
	openbrowser("https://blockchain.info/address/" + publicKeyEncoded)

	// Print QRCode
	qrInTerminal("bitcoin:" + publicKeyEncoded)
}

func openbrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func qrInTerminal(content string) {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		fmt.Println(err)
		return
	}

	for ir, row := range qr.Bitmap() {
		lr := len(row)
		if ir == 0 || ir == 1 || ir == 2 ||
			ir == lr-1 || ir == lr-2 || ir == lr-3 {
			continue
		}
		for ic, col := range row {
			lc := len(qr.Bitmap())
			if ic == 0 || ic == 1 || ic == 2 ||
				ic == lc-1 || ic == lc-2 || ic == lc-3 {
				continue
			}
			if col {
				fmt.Print("\033[38;5;0m  \033[0m")
			} else {
				fmt.Print("\033[48;5;7m  \033[0m")
			}
		}
		fmt.Println()
	}
}
func generatePublicKey(privateKeyBytes []byte) []byte {
	//Generate the public key from the private key.
	//Unfortunately golang ecdsa package does not include a
	//secp256k1 curve as this is fairly specific to bitcoin so this uses wrapped the official bitcoin/c-secp256k1 with cgo.
	var privateKeyBytes32 [32]byte
	copy(privateKeyBytes32[:], privateKeyBytes)

	secp256k1.Start()
	publicKeyBytes, success := secp256k1.Pubkey_create(privateKeyBytes32, false)
	if !success {
		log.Fatal("Failed to create public key.")
	}

	secp256k1.Stop()

	//Next we get a sha256 hash of the public key generated
	//via ECDSA, and then get a ripemd160 hash of the sha256 hash.
	shaHash := sha256.New()
	shaHash.Write(publicKeyBytes)
	shadPublicKeyBytes := shaHash.Sum(nil)

	ripeHash := ripemd160.New()
	ripeHash.Write(shadPublicKeyBytes)
	ripeHashedBytes := ripeHash.Sum(nil)

	return ripeHashedBytes
}

// generate 256 bits of "random" data BTC for private key.
func generatePrivateKey() []byte {
	bytes := make([]byte, 32)
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
