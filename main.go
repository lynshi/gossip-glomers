package main

import (
	"context"
	"gossip-glomers/internal/broadcast"
	"gossip-glomers/internal/echo"
	"gossip-glomers/internal/unique_ids"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	n := maelstrom.NewNode()
	echo.AddEchoHandle(n)

	unique_ids.AddUniqueIdsHandle(ctx, n)

	broadcast.AddBroadcastHandle(ctx, n)
	broadcast.AddReadHandle(n)
	broadcast.AddTopologyHandle(n)

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
