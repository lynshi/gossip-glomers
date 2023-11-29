package broadcast

import (
	"context"
	"encoding/json"
	"log"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"github.com/pkg/errors"
)

type Efficient1Node struct {
	mn *maelstrom.Node

	// Keeps track of received messages.
	messages chan map[int]interface{}

	// Queues up messages yet to be sent to other nodes.
	queue chan int
}

func (n *Efficient1Node) forward_to_all(message int, is_origin bool) {
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

func (n *Efficient1Node) forward(neighbor string, body map[string]any, is_origin bool) {
	for {
		success := false
		err := n.mn.RPC(neighbor, body, func(resp maelstrom.Message) error {
			success = true
			return nil
		})
		if (err == nil && success) || !is_origin {
			// If we're not the origin (i.e. first node to receive the message), don't retry. This
			// avoids a sudden cascade of messages once the partition has recovered. Eventually, the
			// origin will succeed in connecting to this destination.
			return
		}

		// Let's not bother with fancy backoffs since we know the partition
		// heals eventually.
		time.Sleep(2 * time.Second)
		log.Printf("Error making RPC to %s: %v", neighbor, err)
	}
}

func NewEfficient1Node(ctx context.Context, mn *maelstrom.Node) *Efficient1Node {
	messages := make(chan map[int]interface{}, 1)
	messages <- make(map[int]interface{})

	queue := make(chan int)

	n := &Efficient1Node{
		mn:       mn,
		messages: messages,
		queue:    queue,
	}

	n.addBroadcastHandle()
	n.addBroadcastForwardHandle()
	n.addReadHandle()
	n.addTopologyHandle()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-n.queue:
				go n.forward_to_all(msg, true)
			}
		}
	}()

	return n
}

func (n *Efficient1Node) ShutdownEfficient1Node() {
	close(n.messages)
	close(n.queue)
}

func (n *Efficient1Node) addBroadcastHandle() {
	n.mn.Handle("broadcast", n.broadcastBuilder())
}

func (n *Efficient1Node) broadcastBuilder() maelstrom.HandlerFunc {
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

		return n.mn.Reply(req, resp)
	}

	return broadcast
}

func (n *Efficient1Node) addBroadcastForwardHandle() {
	n.mn.Handle("broadcast_forward", n.broadcastForwardBuilder())
}

func (n *Efficient1Node) broadcastForwardBuilder() maelstrom.HandlerFunc {
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
		_, is_new_value := messages[message]
		if is_new_value {
			messages[message] = nil
		}
		n.messages <- messages

		// If it's a new value, continue to propagate it. This results in a lot of duplicate
		// messages but greatly reduces latency. Since there's no efficiency requirement yet, might
		// as well!
		if is_new_value {
			go n.forward_to_all(message, false)
		}

		return nil
	}

	return broadcast_forward
}

func (n *Efficient1Node) addReadHandle() {
	n.mn.Handle("read", n.readBuilder())
}

func (n *Efficient1Node) readBuilder() maelstrom.HandlerFunc {
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

func (n *Efficient1Node) addTopologyHandle() {
	n.mn.Handle("topology", n.toplogyBuilder())
}

func (n *Efficient1Node) toplogyBuilder() maelstrom.HandlerFunc {
	topology := func(req maelstrom.Message) error {
		// Let's still ignore the topology as we'll send messages to every node.

		resp := make(map[string]any)
		resp["type"] = "topology_ok"

		return n.mn.Reply(req, resp)
	}

	return topology
}
