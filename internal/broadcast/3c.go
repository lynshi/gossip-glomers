package broadcast

import (
	"context"
	"encoding/json"
	"log"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"github.com/pkg/errors"
)

type FaultTolerantNode struct {
	// Keeps track of received messages.
	messages chan map[int]interface{}

	// Queues up messages yet to be sent to other nodes.
	queue chan int
}

func NewFaultTolerantNode(ctx context.Context, mn *maelstrom.Node) *FaultTolerantNode {
	messages := make(chan map[int]interface{}, 1)
	messages <- make(map[int]interface{})

	queue := make(chan int)

	n := &FaultTolerantNode{
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
					req["type"] = "broadcastF_orward"
					req["message"] = msg

					neighbor_id := neighbor
					go func() {
						for {
							success := false
							err := mn.RPC(neighbor_id, req, func(resp maelstrom.Message) error {
								success = true
								return nil
							})
							if err == nil && success {
								break
							}

							// Let's not bother with fancy backoffs since we know the partition
							// heals eventually.
							time.Sleep(1 * time.Second)
							log.Printf("Error making RPC to %s: %v", neighbor_id, err)
						}
					}()
				}
			}
		}
	}()

	return n
}

func (n *FaultTolerantNode) ShutdownFaultTolerantNode() {
	close(n.messages)
	close(n.queue)
}

func (n *FaultTolerantNode) AddBroadcastHandle(mn *maelstrom.Node) {
	mn.Handle("broadcast", n.broadcastBuilder(mn))
}

func (n *FaultTolerantNode) broadcastBuilder(mn *maelstrom.Node) maelstrom.HandlerFunc {
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

func (n *FaultTolerantNode) AddBroadcastForwardHandle(mn *maelstrom.Node) {
	mn.Handle("broadcastF_orward", n.broadcastForwardBuilder(mn))
}

func (n *FaultTolerantNode) broadcastForwardBuilder(mn *maelstrom.Node) maelstrom.HandlerFunc {
	broadcastF_orward := func(req maelstrom.Message) error {
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

		resp := make(map[string]any)
		resp["type"] = "broadcastF_orward_ok"

		return mn.Reply(req, resp)
	}

	return broadcastF_orward
}

func (n *FaultTolerantNode) AddReadHandle(mn *maelstrom.Node) {
	mn.Handle("read", n.readBuilder(mn))
}

func (n *FaultTolerantNode) readBuilder(mn *maelstrom.Node) maelstrom.HandlerFunc {
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

func (n *FaultTolerantNode) AddTopologyHandle(mn *maelstrom.Node) {
	mn.Handle("topology", n.toplogyBuilder(mn))
}

func (n *FaultTolerantNode) toplogyBuilder(mn *maelstrom.Node) maelstrom.HandlerFunc {
	topology := func(req maelstrom.Message) error {
		// Let's still ignore the topology as we'll send messages to every node.

		resp := make(map[string]any)
		resp["type"] = "topology_ok"

		return mn.Reply(req, resp)
	}

	return topology
}
