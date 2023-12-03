package broadcast

import (
	"context"
	"encoding/json"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"github.com/pkg/errors"
)

type SingleNodeNode struct {
	mn *maelstrom.Node

	// messages is a 1 element channel containing the array of messages received. Reading from and
	// writing to the channel is analogous to acquiring and releasing a lock.
	messages chan []int
}

func NewSingleNodeNode(ctx context.Context, mn *maelstrom.Node) *SingleNodeNode {
	messages := make(chan []int, 1)
	messages <- make([]int, 0, 1)

	n := SingleNodeNode{
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

	return &n
}

func (n *SingleNodeNode) addBroadcastHandle() {
	n.mn.Handle("broadcast", n.broadcastSingleNodeBuilder())
}

func (n *SingleNodeNode) broadcastSingleNodeBuilder() maelstrom.HandlerFunc {
	broadcast := func(req maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return err
		}

		message, err := getMessage(body)
		if err != nil {
			return errors.Wrap(err, "could not get message")
		}

		msgs := <-n.messages
		msgs = append(msgs, int(message))
		n.messages <- msgs

		resp := make(map[string]any)
		resp["type"] = "broadcast_ok"

		return n.mn.Reply(req, resp)
	}

	return broadcast
}

func (n *SingleNodeNode) addReadHandle() {
	n.mn.Handle("read", n.readBuilder())
}

func (n *SingleNodeNode) readBuilder() maelstrom.HandlerFunc {
	read := func(req maelstrom.Message) error {
		msgs := <-n.messages
		// Now that we have a local copy, we can immediately return it to the channel so that other
		// goroutines are unblocked.
		n.messages <- msgs

		resp := make(map[string]any)
		resp["type"] = "read_ok"
		resp["messages"] = msgs

		return n.mn.Reply(req, resp)
	}

	return read
}

func (n *SingleNodeNode) addTopologyHandle() {
	n.mn.Handle("topology", n.topologyBuilder())
}

func (n *SingleNodeNode) topologyBuilder() maelstrom.HandlerFunc {
	topology := func(req maelstrom.Message) error {
		// Ignore for now as we don't do anything with the topology yet.

		resp := make(map[string]any)
		resp["type"] = "topology_ok"

		return n.mn.Reply(req, resp)
	}

	return topology
}
