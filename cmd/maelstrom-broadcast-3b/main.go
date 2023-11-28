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

	broadcast_node.AddBroadcastHandle(maelstrom_node)
	broadcast_node.AddBroadcastRepeatHandle(maelstrom_node)
	broadcast_node.AddReadHandle(maelstrom_node)
	broadcast_node.AddTopologyHandle(maelstrom_node)

	if err := maelstrom_node.Run(); err != nil {
		log.Fatal(err)
	}
}
