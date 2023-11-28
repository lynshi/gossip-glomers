package broadcast

import (
	"context"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type MultiNodeNode struct {
	messages  map[int]interface{}
	neighbors map[string]chan int
}

func NewMultiNodeNode() *MultiNodeNode {
	messages := make(map[int]interface{})
	neighbors := make(map[string]chan int)

	node := &MultiNodeNode{
		messages:  messages,
		neighbors: neighbors,
	}

	return node
}

func (n *MultiNodeNode) AddBroadcastHandle(ctx context.Context, mn *maelstrom.Node) {
	mn.Handle("broadcast", n.broadcastBuilder(mn))
}

func (n *MultiNodeNode) broadcastBuilder(mn *maelstrom.Node) maelstrom.HandlerFunc {
	broadcast := func(msg maelstrom.Message) error {
		return nil
	}

	return broadcast
}

func (n *MultiNodeNode) AddReadHandle(ctx context.Context, mn *maelstrom.Node) {
	mn.Handle("read", n.readBuilder(mn))
}

func (n *MultiNodeNode) readBuilder(mn *maelstrom.Node) maelstrom.HandlerFunc {
	read := func(msg maelstrom.Message) error {
		return nil
	}

	return read
}

func (n *MultiNodeNode) AddTopologyHandle(ctx context.Context, mn *maelstrom.Node) {
	mn.Handle("topology", n.toplogyBuilder(mn))
}

func (n *MultiNodeNode) toplogyBuilder(mn *maelstrom.Node) maelstrom.HandlerFunc {
	topology := func(msg maelstrom.Message) error {
		return nil
	}

	return topology
}
