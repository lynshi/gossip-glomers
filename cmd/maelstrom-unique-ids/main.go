package main

import (
	"context"
	"gossip-glomers/internal/unique_ids"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	n := maelstrom.NewNode()

	unique_ids.AddUniqueIdsHandle(ctx, n)

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
