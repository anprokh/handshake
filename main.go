package main

import (
	"fmt"
	"handshake/node"
	"time"
)

// https://en.bitcoin.it/wiki/Protocol_documentation
// https://developer.bitcoin.org/reference/p2p_networking.html
func main() {

	done := make(chan struct{})
	var p node.Peer
	go p.Handshake(done)

	// ожидаем завершения handshake, либо завершения по таймауту
	select {
	case <-done:
		// handshake успешно выполнен
	case <-time.After(time.Minute * 5):
		fmt.Println("Work stopped due to timeout...")
	}

	p.Disconnect()
	fmt.Println("ALL DONE")
}
