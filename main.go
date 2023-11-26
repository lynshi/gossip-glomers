package main

import (
	"gossip-glomers/internal/echo"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()
	echo.AddEchoHandle(n)

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
