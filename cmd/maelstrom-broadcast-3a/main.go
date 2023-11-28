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

	n := maelstrom.NewNode()

	broadcast.AddSingleNodeBroadcastHandle(ctx, n)
	broadcast.AddSingleNodeReadHandle(n)
	broadcast.AddSingleNodeTopologyHandle(n)

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
