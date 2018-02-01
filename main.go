package main

import (
	"bufio"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/ipfs/go-datastore"

	golog "github.com/ipfs/go-log"
	net "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	gologging "github.com/whyrusleeping/go-logging"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/keep-network/go-experiments/p2ptest/p2p"
)

var (
	store          = datastore.NewMapDatastore()
	ps             = pstore.NewPeerstore()
)



func main() {

	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	// Parse options from the command line
	listenF := flag.Int("l", 0, "wait for incoming connections")
	//target := flag.String("d", "", "target peer to dial")
	secio := flag.Bool("secio", false, "enable secio")
	seed := flag.Int64("seed", 0, "set random seed for id generation")
	flag.Parse()

	if *listenF == 0 {
		log.Fatal("Please provide a port to bind on with -l")
	}

	// Make a host that listens on the given multiaddress
	ha, err := p2p.MakeBasicHost(ps, *listenF, *secio, *seed)
	if err != nil {
		log.Fatal(err)
	}
	//ht := dht.NewDHT(context.Background(), ha, store)
	//ht.Bootstrap(context.Background())
	ha.SetStreamHandler("/test/1.0.0", func(s net.Stream) {
		log.Println("test")
	})

	ha.SetStreamHandler("/get/1.0.0", func(s net.Stream) {
		log.Println("Get peers")
		peers := ha.Peerstore().Peers()
		peerStrings := []string{}
		for _, peer := range peers {
			addrs := ha.Peerstore().Addrs(peer)
			for _, addr := range addrs {
				if peer.Pretty() != "" && addr.String() != "" {
					peerStrings = append(peerStrings, addr.String()+"/ipfs/"+peer.Pretty())
				}
			}
		}
		data := strings.Join(peerStrings, ",") + "\n"
		log.Println(data)
		s.Write([]byte(data))
	})

	//Adds the current peerlist from the connecting peer and sends back this host's peerlist.
	ha.SetStreamHandler("/add/1.0.0", func(s net.Stream) {
		buf := bufio.NewReader(s)
		str, err := buf.ReadString('\n')
		// The following code extracts target's the peer ID from the
		// given multiaddress
		t := strings.TrimSpace(str)

		ipfsaddr, err := ma.NewMultiaddr(t)
		if err != nil {
			log.Println(err)
		}
		pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS)
		if err != nil {
			log.Println(err)
		}

		peerid, err := peer.IDB58Decode(pid)
		if err != nil {
			log.Println(err)
		}
		// Decapsulate the /ipfs/<peerID> part from the target
		// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
		targetAddr := ipfsaddr.Decapsulate(ipfsaddr)
		if peerid.String() != ha.ID().String() {
			if peerid.String() != "" {
				ps.AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)
			}
		}

		peers := ps.Peers()
		peerStrings := []string{}
		for _, peer := range peers {
			addrs := ps.Addrs(peer)
			for _, addr := range addrs {
				if peer.Pretty() != "" && addr.String() != "" {
					peerStrings = append(peerStrings, addr.String()+"/ipfs/"+peer.Pretty())
				}
			}
		}
		data := strings.Join(peerStrings, ",") + "\n"
		s.Write([]byte(data))
	})

	p2p.BootstrapConnect(ha)

	go func() {
		for {
			log.Println("Host ID:", ha.ID(), ha.Peerstore().Peers())
			time.Sleep(time.Duration(5) * time.Second)
			p2p.AddPeers(ha)
		}
	}()

	//sleep long enough for peerlists to get built.
	time.Sleep(time.Duration(6) * time.Second)
	p2p.Test(ha)
	select {}
}
