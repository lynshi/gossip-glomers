package broadcast

import (
	"context"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type MultiNodeNode struct {
	messages  map[int]interface{}
	neighbors map[string]chan int
}

func NewMultiNodeNode(ctx context.Context, n *maelstrom.Node) *MultiNodeNode {
	messages := make(map[int]interface{})
	neighbors := make(map[string]chan int)

	node := &MultiNodeNode{
		messages:  messages,
		neighbors: neighbors,
	}

	return node
}
