package main

import (
	"context"
	"gossip-glomers/internal/broadcast"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	maelstrom_node := maelstrom.NewNode()

	broadcast_node := broadcast.NewEfficient1Node(ctx, maelstrom_node)
	defer broadcast_node.ShutdownEfficient1Node()

	if err := maelstrom_node.Run(); err != nil {
		log.Fatal(err)
	}
}
