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

	// Queues up messages yet to be sent to other nodes.
	queue chan int
}

func NewMultiNodeNode(ctx context.Context, mn *maelstrom.Node) *MultiNodeNode {
	messages := make(chan map[int]interface{}, 1)
	messages <- make(map[int]interface{})

	queue := make(chan int)

	n := &MultiNodeNode{
		messages: messages,
		queue:    queue,
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-n.queue:
				for _, neighbor := range mn.NodeIDs() {
					req := make(map[string]any)
					req["type"] = "broadcast_repeat"
					req["message"] = msg

					go mn.Send(neighbor, req)
				}
			}
		}
	}()

	return n
}

func (n *MultiNodeNode) ShutdownMultiNodeNode() {
	close(n.messages)
	close(n.queue)
}

func (n *MultiNodeNode) AddBroadcastHandle(mn *maelstrom.Node) {
	mn.Handle("broadcast", n.broadcastBuilder(mn))
}

func (n *MultiNodeNode) broadcastBuilder(mn *maelstrom.Node) maelstrom.HandlerFunc {
	broadcast := func(req maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return err
		}

		message, err := getMessage(body)
		if err != nil {
			return errors.Wrap(err, "could not get message")
		}

		messages := <-n.messages
		messages[message] = nil
		n.messages <- messages

		n.queue <- message

		resp := make(map[string]any)
		resp["type"] = "broadcast_ok"

		return mn.Reply(req, resp)
	}

	return broadcast
}

func (n *MultiNodeNode) AddBroadcastRepeatHandle(mn *maelstrom.Node) {
	mn.Handle("broadcast_repeat", n.broadcastRepeatBuilder(mn))
}

func (n *MultiNodeNode) broadcastRepeatBuilder(mn *maelstrom.Node) maelstrom.HandlerFunc {
	broadcast_repeat := func(req maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return err
		}

		message, err := getMessage(body)
		if err != nil {
			return errors.Wrap(err, "could not get message")
		}

		messages := <-n.messages
		messages[message] = nil
		n.messages <- messages

		// Don't respond since we use Send, which is fire-and-forget.
		return nil
	}

	return broadcast_repeat
}

func (n *MultiNodeNode) AddReadHandle(mn *maelstrom.Node) {
	mn.Handle("read", n.readBuilder(mn))
}

func (n *MultiNodeNode) readBuilder(mn *maelstrom.Node) maelstrom.HandlerFunc {
	read := func(req maelstrom.Message) error {
		messages := <-n.messages
		n.messages <- messages

		resp := make(map[string]any)
		resp["type"] = "read_ok"
		resp_messages := make([]int, 0, len(messages))

		for v, _ := range messages {
			resp_messages = append(resp_messages, v)
		}

		resp["messages"] = resp_messages

		return mn.Reply(req, resp)
	}

	return read
}

func (n *MultiNodeNode) AddTopologyHandle(mn *maelstrom.Node) {
	mn.Handle("topology", n.toplogyBuilder(mn))
}

func (n *MultiNodeNode) toplogyBuilder(mn *maelstrom.Node) maelstrom.HandlerFunc {
	topology := func(req maelstrom.Message) error {
		// Let's still ignore the topology as we'll send messages to every node.

		resp := make(map[string]any)
		resp["type"] = "topology_ok"

		return mn.Reply(req, resp)
	}

	return topology
}
