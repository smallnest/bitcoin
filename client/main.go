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

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := scanner.Text()
		switch cmd {
		case "version":
			version()
		case "verack":
			verack()
		case "getaddr":
			getaddr()
		case "quit", "exit":
			return
		default:
			log.Errorf("unknown cmd: %s", cmd)
		}
	}

}

func version() {
	tcpAddrMe := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	me := wire.NewNetAddress(tcpAddrMe, wire.SFNodeGetUTXO)
	tcpAddrYou := &net.TCPAddr{IP: net.ParseIP(peer), Port: *port}
	you := wire.NewNetAddress(tcpAddrYou, wire.SFNodeNetwork)
	nonce := rand.Int63()
	lastBlock := int32(507742)
	msg1 := wire.NewMsgVersion(me, you, uint64(nonce), lastBlock)
	err := wire.WriteMessage(conn, msg1, pver, btcnet)
	if err != nil {
		log.Error(err)
	}

	msg2, rawPayload, err := wire.ReadMessage(conn, pver, btcnet)
	if err != nil {
		log.Error(err)
	}

	_ = rawPayload
	log.Infof("Received Type: %T", msg2)
	spew.Dump("Received:", msg2)
}

func verack() {
	msg1 := wire.NewMsgVerAck()
	err := wire.WriteMessage(conn, msg1, pver, btcnet)
	if err != nil {
		log.Error(err)
	}

	msg2, rawPayload, err := wire.ReadMessage(conn, pver, btcnet)
	if err != nil {
		log.Error(err)
	}

	_ = rawPayload
	log.Infof("Received Type: %T", msg2)
	spew.Dump("Received:", msg2)
}

func getaddr() {
	msg1 := wire.NewMsgGetAddr()
	err := wire.WriteMessage(conn, msg1, pver, btcnet)
	if err != nil {
		log.Error(err)
	}

	for {
		msg2, rawPayload, err := wire.ReadMessage(conn, pver, btcnet)
		if err != nil {
			log.Error(err)
		}

		if msg, ok := msg2.(*wire.MsgAddr); ok {
			_ = rawPayload
			spew.Dump("Received:", msg.AddrList)
			if len(msg.AddrList) == 0 {
				return
			}
		} else {
			log.Infof("Received unexpected Type: %T", msg2)
		}
	}

}
