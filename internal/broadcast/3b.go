package broadcast

import (
	"context"
	"encoding/json"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type MultiNodeNode struct {
	messages  map[int]interface{}
	neighbors map[string]chan int
}

func NewMultiNodeNode() *MultiNodeNode {
	// Keeps track of received messages.
	messages := make(map[int]interface{})

	// Queues up messages yet to be sent to each neighbor.
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
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		node_id := mn.ID()
		topology := (body["topology"]).(map[string][]string)
		neighbors := topology[node_id]
		for _, neighbor := range neighbors {
			if _, ok := n.neighbors[neighbor]; ok {
				// Ignore known neighbors (though I think this message is only sent once anyway).
				continue
			}

			n.neighbors[neighbor] = make(chan int)
		}

		resp := make(map[string]any)
		resp["type"] = "topology_ok"

		return mn.Reply(msg, resp)
	}

	return topology
}
