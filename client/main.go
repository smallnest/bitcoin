package main

import (
	"bufio"
	"flag"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"

	"github.com/btcsuite/btcd/wire"
	"github.com/davecgh/go-spew/spew"
	"github.com/smallnest/log"
)

var (
	peers = flag.String("peers", "seed.bitcoin.sipa.be,dnsseed.bluematt.me,dnsseed.bitcoin.dashjr.org,seed.bitcoinstats.com,seed.bitcoin.jonasschnelli.ch,seed.btc.petertodd.org", "main net")
	port  = flag.Int("port", 8333, "port")
	cmd   = flag.String("cmd", "version", "version")
)

var (
	conn net.Conn
	peer string
	addr string

	pver   = wire.ProtocolVersion
	btcnet = wire.MainNet
)

// https://en.bitcoin.it/wiki/Protocol_documentation
//
// https://blockchain.info
// https://blockexplorer.com
// https://live.blockcypher.com
// https://www.blocktrail.com/BTC
func main() {
	nodes := strings.Split(*peers, ",")
	i := rand.Intn(len(nodes))
	peer = nodes[i]

	addr = peer + ":" + strconv.Itoa(*port)

	var err error
	conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	go readMessages()

	version()
	verack()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		t := scanner.Text()
		cmd := strings.SplitN(t, " ", 2)

		switch cmd[0] {
		// case "version":
		// 	version()
		// case "verack":
		// 	verack()
		case "getaddr":
			go getaddr()
		case "getheaders":
			blockhash := ""
			if len(cmd) > 1 {
				blockhash = cmd[1]
			}
			go getheaders(blockhash)
		case "ping":
			go ping()
		case "getblocks":
			blockhash := ""
			if len(cmd) > 1 {
				blockhash = cmd[1]
			}
			go getblocks(blockhash)
		case "quit", "exit":
			return
		default:
			log.Errorf("unknown cmd: %s", cmd)
		}
	}

}

func readMessages() {
	for {
		msg, rawPayload, err := wire.ReadMessage(conn, pver, btcnet)
		if err != nil {
			log.Fatal(err)
		}

		_ = rawPayload

		// ignore *wire.MsgInv because there are a lot of such messages
		switch v := msg.(type) {
		case *wire.MsgInv:
			go getdata(v)
		case *wire.MsgPing:
			log.Infof("Received: %s\n", spew.Sdump(msg))
			go pong(v)
		case *wire.MsgBlock:
			log.Infof("Received: %x\n", rawPayload)
		default:
			log.Infof("Received: %s\n", spew.Sdump(msg))
		}
	}
}

func version() {
	tcpAddrMe := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	me := wire.NewNetAddress(tcpAddrMe, wire.SFNodeNetwork)
	tcpAddrYou := &net.TCPAddr{IP: net.ParseIP(peer), Port: *port}
	you := wire.NewNetAddress(tcpAddrYou, wire.SFNodeNetwork)
	nonce := rand.Int63()
	lastBlock := int32(507742)
	msg1 := wire.NewMsgVersion(me, you, uint64(nonce), lastBlock)
	err := wire.WriteMessage(conn, msg1, pver, btcnet)
	if err != nil {
		log.Fatal(err)
	}
}

func verack() {
	msg1 := wire.NewMsgVerAck()
	err := wire.WriteMessage(conn, msg1, pver, btcnet)
	if err != nil {
		log.Fatal(err)
	}
}

func getaddr() {
	msg1 := wire.NewMsgGetAddr()
	err := wire.WriteMessage(conn, msg1, pver, btcnet)
	if err != nil {
		log.Fatal(err)
	}
}

func getheaders(blockhash string) {
	msg1 := wire.NewMsgGetHeaders()

	if blockhash == "" {
		blockhash = "000000000000000000648fd2fa6ccfba1b60441f5958f81594817398ece0a1fd"
	}
	h, _ := chainhash.NewHashFromStr(blockhash)

	msg1.AddBlockLocatorHash(h)
	err := wire.WriteMessage(conn, msg1, pver, btcnet)
	if err != nil {
		log.Fatal(err)
	}
}

func ping() {
	nonce := rand.Int63()
	msg1 := wire.NewMsgPing(uint64(nonce))
	err := wire.WriteMessage(conn, msg1, pver, btcnet)
	if err != nil {
		log.Fatal(err)
	}
}

func pong(msg *wire.MsgPing) {
	msg1 := wire.NewMsgPong(msg.Nonce)
	err := wire.WriteMessage(conn, msg1, pver, btcnet)
	if err != nil {
		log.Fatal(err)
	}
}

func getblocks(blockhash string) {
	if blockhash == "" {
		blockhash = "000000000000000000648fd2fa6ccfba1b60441f5958f81594817398ece0a1fd"
	}
	h, _ := chainhash.NewHashFromStr(blockhash)

	msg1 := wire.NewMsgGetBlocks(h)

	err := wire.WriteMessage(conn, msg1, pver, btcnet)
	if err != nil {
		log.Fatal(err)
	}
}

func getdata(msg *wire.MsgInv) {
	var invList []*wire.InvVect

	for _, m := range msg.InvList {
		if m.Type == wire.InvTypeBlock {
			invList = append(invList, m)
		}
	}

	if len(invList) == 0 {
		return
	}

	log.Infof("Received: %s\n", spew.Sdump(invList))
	msg1 := wire.NewMsgGetData()
	msg1.InvList = invList

	err := wire.WriteMessage(conn, msg1, pver, btcnet)
	if err != nil {
		log.Fatal(err)
	}
}
