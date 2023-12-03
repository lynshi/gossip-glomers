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

	broadcast_node := broadcast.NewMultiNodeNode(ctx, maelstrom_node)
	defer broadcast_node.ShutdownMultiNodeNode()

	if err := maelstrom_node.Run(); err != nil {
		log.Fatal(err)
	}
}
