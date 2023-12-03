package broadcast

import (
	"context"
	"encoding/json"
	"strings"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"github.com/pkg/errors"
)

type MultiNodeNode struct {
	mn *maelstrom.Node

	// Keeps track of received messages.
	messages chan map[int]interface{}
}

func NewMultiNodeNode(ctx context.Context, mn *maelstrom.Node) *MultiNodeNode {
	messages := make(chan map[int]interface{}, 1)
	messages <- make(map[int]interface{})

	n := &MultiNodeNode{
		mn:       mn,
		messages: messages,
	}

	go func() {
		<-ctx.Done()
		close(messages)
	}()

	n.addBroadcastHandle()
	n.addReadHandle()
	n.addTopologyHandle()

	return n
}

func (n *MultiNodeNode) addBroadcastHandle() {
	n.mn.Handle("broadcast", n.broadcastBuilder())
}

func (n *MultiNodeNode) broadcastBuilder() maelstrom.HandlerFunc {
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

		// Only forward if the message did not come from another node.
		if strings.HasPrefix(req.Src, "n") {
			return nil
		}

		go func() {
			for _, neighbor := range n.mn.NodeIDs() {
				req := make(map[string]any)
				req["type"] = "broadcast"
				req["message"] = message

				go n.mn.Send(neighbor, req)
			}
		}()

		resp := make(map[string]any)
		resp["type"] = "broadcast_ok"

		return n.mn.Reply(req, resp)
	}

	return broadcast
}

func (n *MultiNodeNode) addReadHandle() {
	n.mn.Handle("read", n.readBuilder())
}

func (n *MultiNodeNode) readBuilder() maelstrom.HandlerFunc {
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

		return n.mn.Reply(req, resp)
	}

	return read
}

func (n *MultiNodeNode) addTopologyHandle() {
	n.mn.Handle("topology", n.toplogyBuilder())
}

func (n *MultiNodeNode) toplogyBuilder() maelstrom.HandlerFunc {
	topology := func(req maelstrom.Message) error {
		// Let's still ignore the topology as we'll send messages to every node.

		resp := make(map[string]any)
		resp["type"] = "topology_ok"

		return n.mn.Reply(req, resp)
	}

	return topology
}
