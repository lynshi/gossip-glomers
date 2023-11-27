package broadcast

import (
	"context"
	"encoding/json"
	"reflect"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"github.com/pkg/errors"
)

var messages chan []int

func AddBroadcastHandle(ctx context.Context, n *maelstrom.Node) {
	// messages is a 1 element channel containing the array of messages received. Reading from and
	// writing to the channel is analogous to acquiring and releasing a lock.
	messages = make(chan []int, 1)
	messages <- make([]int, 0, 1)

	// Use a queue for processing received broadcasts to ensure that there is only one writer to
	// messages.
	queue := make(chan int)

	go func() {
		for {
			select {
			case m := <-queue:
				msgs := <-messages
				msgs = append(msgs, m)
				messages <- msgs
			case <-ctx.Done():
				close(messages)
				close(queue)
				return
			}
		}
	}()

	n.Handle("broadcast", broadcastBuilder(n, queue))
}

func broadcastBuilder(n *maelstrom.Node, queue chan int) maelstrom.HandlerFunc {
	broadcast := func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		message, ok := (body["message"]).(float64)
		if !ok {
			return errors.Errorf(
				"could not convert message %v to int. Has type %v", body["message"],
				reflect.TypeOf(body["message"]))
		}

		queue <- int(message)

		resp := make(map[string]any)
		resp["type"] = "broadcast_ok"

		return n.Reply(msg, resp)
	}

	return broadcast
}

func AddReadHandle(n *maelstrom.Node) {
	n.Handle("read", readBuilder(n))
}

func readBuilder(n *maelstrom.Node) maelstrom.HandlerFunc {
	read := func(msg maelstrom.Message) error {
		msgs := <-messages
		defer func() {
			messages <- msgs
		}()

		resp := make(map[string]any)
		resp["type"] = "read_ok"
		resp["messages"] = msgs

		return n.Reply(msg, resp)
	}

	return read
}

func AddTopologyHandle(n *maelstrom.Node) {
	n.Handle("topology", topologyBuilder(n))
}

func topologyBuilder(n *maelstrom.Node) maelstrom.HandlerFunc {
	topology := func(msg maelstrom.Message) error {
		// Ignore for now as we don't do anything with the topology yet.

		resp := make(map[string]any)
		resp["type"] = "topology_ok"

		return n.Reply(msg, resp)
	}

	return topology
}
