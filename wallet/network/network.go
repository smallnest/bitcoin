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
	"net"
	"strconv"
	"strings"
	"time"
)

// https://blockexplorer.com

var (
	transaction    = flag.String("transaction", "", "")
	networkAddress = flag.String("network-address", "127.0.0.1", "")
	nodeAddress    = flag.String("node-address", "seed.bitcoinstats.com", "")
	testnet        = flag.Bool("testnet", false, "Whether or not to use the bitcoin testnet. Defaults to false")
)

var magicBytes string

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var port string

	if *testnet {
		magicBytes = "0b110907"
		port = ":18333"
	} else {
		magicBytes = "f9beb4d9"
		port = ":8333"
	}

	ips, err := net.LookupHost(*nodeAddress)
	*nodeAddress = ips[0]

	//Attempt to connect to the node
	servAddr := *nodeAddress + port

	conn, err := net.DialTimeout("tcp", servAddr, 10*time.Second)
	if err != nil {
		log.Fatalf("failed to dail %s : %v", servAddr, err)
	}

	//Send a version message to the node.
	versionMessage := makeMessage(magicBytes, "version", getVersionMessage())

	n, err := conn.Write(versionMessage)
	if err != nil {
		log.Fatal("failed to versionMessage: ", err, n)
	}

	reply := make([]byte, 1024)

	n, err = conn.Read(reply)
	if err != nil {
		log.Fatal("Write to server failed:", err.Error())
	}

	log.Println("reply from server=", string(reply[n:]))

	reply2 := make([]byte, 1024)
	n, err = conn.Read(reply2)
	if err != nil {
		log.Fatal("Write to server failed:", err.Error())
	}

	log.Println("reply from server=", string(reply2[:n]))

	rawTransaction, err := hex.DecodeString(*transaction)
	if err != nil {
		log.Fatal("Write of rawTransaction fails", err.Error())
	}

	//Send the transaction message to the node
	txMessage := makeMessage(magicBytes, "tx", rawTransaction)

	n, err = conn.Write(txMessage)
	if err != nil {
		log.Fatal("failed to write txMessage: ", err, n)
	}

	for {
		reply3 := make([]byte, 1024)

		n, err := conn.Read(reply3)
		if err != nil {
			//do nothing
		}

		if n > 0 {
			log.Println("reply from server=")
			log.Println(string(reply3[:n]))
			log.Println(hex.EncodeToString(reply3[:n]))
		}
	}

	conn.Close()
}

func makeMessage(magic string, command string, payload []byte) []byte {
	//Messages on the bitcoin protocol consist of
	//4 bytes magic value indicating the origin network.
	//12 bytes which contains the command you're sending.
	//4 bytes which represent the length of your payload
	//4 byte checksum which is the first 4 bytes of sha256(sha256(payload))
	//your payload

	magicBytes, err := hex.DecodeString(magic)
	if err != nil {
		fmt.Println(err)
	}

	shaHash := sha256.New()
	shaHash.Write(payload)
	shaHashFirst := shaHash.Sum(nil)

	shaHash2 := sha256.New()
	shaHash2.Write(shaHashFirst)
	hashedPayload := shaHash2.Sum(nil)

	checksum := hashedPayload[0:4]

	length := uint32(len(payload))
	lengthBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthBytes, length)

	commandBytes := make([]byte, 12)
	for i := 0; i < 12; i++ {
		if i >= len(command) {
			commandBytes[i] = 0
		} else {
			commandBytes[i] = command[i]
		}
	}

	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.LittleEndian, magicBytes)
	binary.Write(buffer, binary.LittleEndian, commandBytes)
	buffer.Write(lengthBytes)

	binary.Write(buffer, binary.LittleEndian, checksum)
	buffer.Write(payload)

	return buffer.Bytes()
}
func getNetworkAddress(ip string) []byte {
	//Network addresses in the bitcoin protocol are represented with
	//8 bytes services
	//16 bytes IPv6
	//2 bytes port

	services, _ := hex.DecodeString("0100000000000000")

	ipv4Strings := strings.Split(ip, ".")
	ipv4Bytes := make([]byte, 4)

	for i := 0; i < 4; i++ {
		ipByte, err := strconv.Atoi(ipv4Strings[i])
		if err != nil {
			log.Fatal(err)
		}

		ipv4Bytes[i] = byte(ipByte)
	}

	ipv64 := new(bytes.Buffer)
	prefix, err := hex.DecodeString("00000000000000000000FFFF")
	if err != nil {
		log.Fatal(err)
	}
	ipv64.Write(prefix)
	binary.Write(ipv64, binary.BigEndian, ipv4Bytes)

	port := make([]byte, 2)
	binary.BigEndian.PutUint16(port, uint16(8333))

	networkAddressBuffer := new(bytes.Buffer)
	binary.Write(networkAddressBuffer, binary.LittleEndian, services)
	networkAddressBuffer.Write(ipv64.Bytes())
	networkAddressBuffer.Write(port)

	return networkAddressBuffer.Bytes()
}

func getVersionMessage() []byte {
	//Version messages is the initial message we send
	//to the node after the TCP handshake has completed
	//and we are connected.

	//It consists of
	//4 bytes protocol version
	//8 bytes services (same as network address)
	//8 bytes unix timestamp
	//26 bytes addr_recv
	//26 bytes addr_from
	//8 bytes nonce
	//1 byte user agent
	//4 bytes start_height

	version, err := hex.DecodeString("62EA0000")
	if err != nil {
		log.Fatal(err)
	}

	services, err := hex.DecodeString("0100000000000000")
	if err != nil {
		log.Fatal(err)
	}

	timestamp := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestamp, uint64(time.Now().Unix()))

	addrRecv := getNetworkAddress(*nodeAddress)
	addrFrom := getNetworkAddress(*networkAddress) //me

	nonce := make([]byte, 8)
	for i := 0; i < 8; i++ {
		nonce[i] = byte(randInt(0, math.MaxUint8))
	}

	userAgent, _ := hex.DecodeString("00")
	startHeight, _ := hex.DecodeString("00000000")

	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.LittleEndian, version)
	binary.Write(buffer, binary.LittleEndian, services)
	buffer.Write(timestamp)
	buffer.Write(addrRecv)
	buffer.Write(addrFrom)
	binary.Write(buffer, binary.LittleEndian, nonce)
	buffer.Write(userAgent)
	buffer.Write(startHeight)

	return buffer.Bytes()

}

func randInt(min int, max int) uint8 {
	rand.Seed(time.Now().UTC().UnixNano())
	return uint8(min + rand.Intn(max-min))
}
