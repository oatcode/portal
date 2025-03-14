package main

import (
	"context"
	"log"
	"net"

	"github.com/oatcode/portal"
)

func tunnelClient() {
	log.Printf("Tunnel client...")
	c, err := net.Dial("tcp", tunnelAddress)
	if err != nil {
		log.Fatalf("Tunnel client dial error: %v", err)
	}
	defer c.Close()
	log.Print("Tunnel client connected")

	tunnel := &portal.Tunnel{}
	tunnel.Serve(context.Background(), NewNetConnFramer(c))
}
