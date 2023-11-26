package main

import (
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()
	AddEchoHandle(n)

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
