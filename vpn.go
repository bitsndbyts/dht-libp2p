package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	discovery "github.com/libp2p/go-libp2p-discovery"
	protocol "github.com/libp2p/go-libp2p-protocol"
	"os"
	"github.com/ipfs/go-log"
	inet "github.com/libp2p/go-libp2p-net"

	"github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/multiformats/go-multiaddr"
)

var logger = log.Logger("rendezvous")

func handleStream(stream inet.Stream) {
	logger.Info("Got a new stream!")

	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go readData(rw)
	go writeData(rw)

	// 'stream' will stay open until you close it (or the other side closes it).
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}

		if str == "" {
			return
		}
		if str != "\n" {
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}

	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}

		_, err = rw.WriteString(fmt.Sprintf("%s\n", sendData))
		if err != nil {
			fmt.Println("Error writing to buffer")
			panic(err)
		}
		err = rw.Flush()
		if err != nil {
			fmt.Println("Error flushing buffer")
			panic(err)
		}
	}
}
func main() {

	RendezvousString:="testing"
	ProtoID:= protocol.ID("VPN")
	port := 8000
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
		r := rand.Reader
		prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
		if err != nil {
			panic(err)
		}

		// 0.0.0.0 will listen on any interface device.
		sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", int64(port)))
		port = port + 1

		host, err := libp2p.New(
			ctx,
			libp2p.ListenAddrs(sourceMultiAddr),
			libp2p.Identity(prvKey),
		)
		if err != nil {
			panic(err)
		}

	host.SetStreamHandler(ProtoID, handleStream)

		kdht,_:= dht.New(ctx,host)
		bootstapers := dht.DefaultBootstrapPeers


	for _,addr:= range bootstapers {
		peerinfo, err := pstore.InfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}
		err = host.Connect(ctx, *peerinfo)
		if err != nil {
			fmt.Println("Not connected bootstarpnode",peerinfo)
		} else {
			fmt.Println("connected",peerinfo)
		}
	}

	logger.Info("Announcing ourselves...")
	routingDiscovery := discovery.NewRoutingDiscovery(kdht)
	discovery.Advertise(ctx, routingDiscovery,RendezvousString)
	logger.Debug("Successfully announced!")

	// Now, look for others who have announced
	// This is like your friend telling you the location to meet you.
	logger.Debug("Searching for other peers...")
	peerChan, err := routingDiscovery.FindPeers(ctx, RendezvousString)
	if err != nil {
		panic(err)
	}

	for peer := range peerChan {
		if peer.ID == host.ID() {
			continue
		}
		logger.Debug("Found peer:", peer)

		logger.Debug("Connecting to:", peer)
		stream, err := host.NewStream(ctx, peer.ID, ProtoID)

		if err != nil {
			logger.Warning("Connection failed:", err)
			continue
		} else {
			rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

			go writeData(rw)
			go readData(rw)
		}

		logger.Info("Connected to:", peer)
	}

	select {}

}
