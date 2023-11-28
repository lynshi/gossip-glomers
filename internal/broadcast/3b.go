package broadcast

import (
	"context"
	"encoding/json"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"github.com/pkg/errors"
)

type MultiNodeNode struct {
	// Keeps track of received messages.
	messages chan map[int]interface{}

	// Queues up messages yet to be sent to each neighbor.
	neighbors map[string]chan int
}

func NewMultiNodeNode() *MultiNodeNode {
	messages := make(chan map[int]interface{}, 1)
	messages <- make(map[int]interface{})

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
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		message, err := getMessage(body)
		if err != nil {
			return errors.Wrap(err, "could not get message")
		}

		messages := <-n.messages
		if _, ok := messages[message]; ok {
			// We've received this message before, so do nothing.
			resp := make(map[string]any)
			resp["type"] = "broadcast_ok"

			return mn.Reply(msg, resp)
		}

		messages[message] = nil
		n.messages <- messages

		// Ensure the message is sent to neighbors.
		for neighbor, c := range n.neighbors {
			if neighbor == msg.Src {
				// Don't send a message back to the source as it already knows about this message.
				continue
			}

			c <- message
		}

		resp := make(map[string]any)
		resp["type"] = "broadcast_ok"

		return mn.Reply(msg, resp)
	}

	return broadcast
}

func (n *MultiNodeNode) AddReadHandle(ctx context.Context, mn *maelstrom.Node) {
	mn.Handle("read", n.readBuilder(mn))
}

func (n *MultiNodeNode) readBuilder(mn *maelstrom.Node) maelstrom.HandlerFunc {
	read := func(msg maelstrom.Message) error {
		messages := <-n.messages
		n.messages <- messages

		resp := make(map[string]any)
		resp["type"] = "read_ok"
		resp_messages := make([]int, 0, len(messages))

		for v, _ := range messages {
			resp_messages = append(resp_messages, v)
		}

		resp["messages"] = resp_messages

		return mn.Reply(msg, resp)
	}

	return read
}

func (n *MultiNodeNode) AddTopologyHandle(ctx context.Context, mn *maelstrom.Node) {
	mn.Handle("topology", n.toplogyBuilder(ctx, mn))
}

func (n *MultiNodeNode) toplogyBuilder(ctx context.Context, mn *maelstrom.Node) maelstrom.HandlerFunc {
	topology := func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		neighbors := getNeighbors(mn.ID(), body)
		for _, neighbor := range neighbors {
			if _, ok := n.neighbors[neighbor]; ok {
				// Ignore known neighbors (though I think this message is only sent once anyway).
				continue
			}

			n.neighbors[neighbor] = make(chan int)

			// Avoid capturing the loop variable in the nested method.
			neighbor_id := neighbor
			go func() {
				for {
					select {
					case <-ctx.Done():
						close(n.neighbors[neighbor_id])
						return
					case v := <-n.neighbors[neighbor_id]:
						req := make(map[string]any)
						req["type"] = "broadcast"
						req["message"] = v

						mn.Send(neighbor_id, req)
					}
				}
			}()
		}

		resp := make(map[string]any)
		resp["type"] = "topology_ok"

		return mn.Reply(msg, resp)
	}

	return topology
}
