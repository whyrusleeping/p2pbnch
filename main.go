package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p"
	net "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

func main() {
	listenF := flag.Int("l", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	//secio := flag.Bool("secio", false, "enable secio")

	flag.Parse()

	listenaddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", *listenF)

	ctx := context.Background()

	h, err := libp2p.New(ctx, libp2p.ListenAddrStrings(listenaddr))
	if err != nil {
		log.Fatal(err)
	}

	if *target == "" {
		for _, a := range h.Addrs() {
			fmt.Printf("%s/p2p/%s\n", a, h.ID())
		}
		done := make(chan struct{})
		h.SetStreamHandler("/xfertest", func(s net.Stream) {
			defer func() {
				close(done)
			}()

			start := time.Now()
			defer s.Close()
			n, err := io.Copy(ioutil.Discard, s)
			if err != nil {
				log.Println("COPY ERROR:", err)
				return
			}
			took := time.Since(start)
			fmt.Println("Read bytes: ", n, took)
		})

		log.Println("listening for connections")
		<-done
		return
	}
	// This is where the listener code ends

	ipfsaddr, err := ma.NewMultiaddr(*target)
	if err != nil {
		log.Fatalln(err)
	}

	p2pa, err := peer.AddrInfoFromP2pAddr(ipfsaddr)
	if err != nil {
		log.Fatalln(err)
	}

	// We need to add the target to our peerstore, so we know how we can
	// contact it
	h.Peerstore().AddAddr(p2pa.ID, p2pa.Addrs[0], peerstore.PermanentAddrTTL)

	log.Println("opening stream")
	// make a new stream from host B to host A
	// it should be handled on host A by the handler we set above
	s, err := h.NewStream(context.Background(), p2pa.ID, "/xfertest")
	if err != nil {
		log.Fatalln("opening new stream: ", err)
	}

	r := rand.New(rand.NewSource(42))
	beg := time.Now()
	lr := io.LimitReader(r, 1024*1024*1024)

	nc, err := io.Copy(s, lr)
	if err != nil {
		log.Println("failed to write out bytes: ", err)
		return
	}

	took := time.Since(beg)
	log.Printf("wrote %d bytes in %s", nc, took)
	s.Close()
	time.Sleep(time.Second / 10)
}
