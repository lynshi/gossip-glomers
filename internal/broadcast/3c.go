package broadcast

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"github.com/pkg/errors"
)

type FaultTolerantNode struct {
	mn *maelstrom.Node

	// Keeps track of received messages.
	messages chan map[int]interface{}
}

func (n *FaultTolerantNode) forward_to_all(message int, is_origin bool) {
	for _, neighbor := range n.mn.NodeIDs() {
		if neighbor == n.mn.ID() {
			continue
		}

		req := make(map[string]any)
		req["type"] = "broadcast_forward"
		req["message"] = message

		go n.forward(neighbor, req, is_origin)
	}
}

func (n *FaultTolerantNode) forward(neighbor string, body map[string]any, is_origin bool) {
	for {
		success := false
		err := n.mn.RPC(neighbor, body, func(resp maelstrom.Message) error {
			success = true
			return nil
		})
		if err == nil && success {
			return
		} else if !is_origin {
			// If we're not the origin (i.e. first node to receive the message), don't retry. This
			// avoids a sudden cascade of messages once the partition has recovered. Eventually, the
			// origin will succeed in connecting to this destination.
			return
		}

		// Let's not bother with fancy backoffs since we know the partition
		// heals eventually.
		time.Sleep(500 * time.Millisecond)
	}
}

func NewFaultTolerantNode(ctx context.Context, mn *maelstrom.Node) *FaultTolerantNode {
	messages := make(chan map[int]interface{}, 1)
	messages <- make(map[int]interface{})

	n := &FaultTolerantNode{
		mn:       mn,
		messages: messages,
	}

	n.addBroadcastHandle()
	n.addBroadcastForwardHandle()
	n.addReadHandle()
	n.addTopologyHandle()

	go func() {
		<-ctx.Done()
		close(messages)
	}()

	return n
}

func (n *FaultTolerantNode) ShutdownFaultTolerantNode() {
	close(n.messages)
}

func (n *FaultTolerantNode) addBroadcastHandle() {
	n.mn.Handle("broadcast", n.broadcastBuilder())
}

func (n *FaultTolerantNode) broadcastBuilder() maelstrom.HandlerFunc {
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
		_, val_exists := messages[message]
		messages[message] = nil
		n.messages <- messages

		source_is_node := strings.HasPrefix(req.Src, "n")
		// If it's a new value, continue to propagate it. This results in a lot of duplicate
		// messages but can help reduce latency depending on the shape of the partition. Since
		// there's no efficiency requirement yet, might as well!
		if !val_exists {
			go n.forward_to_all(message, !source_is_node)
		}

		if source_is_node {
			return nil
		}

		resp := make(map[string]any)
		resp["type"] = "broadcast_ok"

		return n.mn.Reply(req, resp)
	}

	return broadcast
}

func (n *FaultTolerantNode) addBroadcastForwardHandle() {
	n.mn.Handle("broadcast_forward", n.broadcastForwardBuilder())
}

func (n *FaultTolerantNode) broadcastForwardBuilder() maelstrom.HandlerFunc {
	broadcast_forward := func(req maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return err
		}

		message, err := getMessage(body)
		if err != nil {
			return errors.Wrap(err, "could not get message")
		}

		messages := <-n.messages
		_, val_exists := messages[message]
		if !val_exists {
			messages[message] = nil

			// If it's a new value, continue to propagate it. This results in a lot of duplicate
			// messages but can help reduce latency depending on the shape of the partition. Since
			// there's no efficiency requirement yet, might as well!
			go n.forward_to_all(message, false)
		}
		n.messages <- messages

		resp := make(map[string]any)

		return n.mn.Reply(req, resp)
	}

	return broadcast_forward
}

func (n *FaultTolerantNode) addReadHandle() {
	n.mn.Handle("read", n.readBuilder())
}

func (n *FaultTolerantNode) readBuilder() maelstrom.HandlerFunc {
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

func (n *FaultTolerantNode) addTopologyHandle() {
	n.mn.Handle("topology", n.toplogyBuilder())
}

func (n *FaultTolerantNode) toplogyBuilder() maelstrom.HandlerFunc {
	topology := func(req maelstrom.Message) error {
		// Let's still ignore the topology as we'll send messages to every node.

		resp := make(map[string]any)
		resp["type"] = "topology_ok"

		return n.mn.Reply(req, resp)
	}

	return topology
}
