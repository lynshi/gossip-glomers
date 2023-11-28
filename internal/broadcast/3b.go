package broadcast

import (
	"context"
	"encoding/json"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"github.com/pkg/errors"
)

type MultiNodeNode struct {
	messages  chan map[int]interface{}
	neighbors map[string]chan int
}

func NewMultiNodeNode() *MultiNodeNode {
	// Keeps track of received messages.
	messages := make(chan map[int]interface{}, 1)
	messages <- make(map[int]interface{})

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
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		message, err := getMessage(body)
		if err != nil {
			return errors.Wrap(err, "could not get message")
		}

		msgs := <-messages
		msgs = append(msgs, int(message))
		messages <- msgs

		resp := make(map[string]any)
		resp["type"] = "broadcast_ok"

		return n.Reply(msg, resp)
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

		node_id := mn.ID()
		topology := (body["topology"]).(map[string][]string)
		neighbors := topology[node_id]
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
