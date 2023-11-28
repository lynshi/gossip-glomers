package broadcast

import (
	"context"
	"encoding/json"
	"reflect"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"github.com/pkg/errors"
)

var messages chan []int

func AddSingleNodeBroadcastHandle(ctx context.Context, n *maelstrom.Node) {
	// messages is a 1 element channel containing the array of messages received. Reading from and
	// writing to the channel is analogous to acquiring and releasing a lock.
	messages = make(chan []int, 1)
	messages <- make([]int, 0, 1)

	go func() {
		<-ctx.Done()
		close(messages)
	}()

	n.Handle("broadcast", broadcastSingleNodeBuilder(n))
}

func broadcastSingleNodeBuilder(n *maelstrom.Node) maelstrom.HandlerFunc {
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

		msgs := <-messages
		msgs = append(msgs, int(message))
		messages <- msgs

		resp := make(map[string]any)
		resp["type"] = "broadcast_ok"

		return n.Reply(msg, resp)
	}

	return broadcast
}

func AddSingleNodeReadHandle(n *maelstrom.Node) {
	n.Handle("read", readSingleNodeBuilder(n))
}

func readSingleNodeBuilder(n *maelstrom.Node) maelstrom.HandlerFunc {
	read := func(msg maelstrom.Message) error {
		msgs := <-messages
		// Now that we have a local copy, we can immediately restore it so that other goroutines are unblocked.
		messages <- msgs

		resp := make(map[string]any)
		resp["type"] = "read_ok"
		resp["messages"] = msgs

		return n.Reply(msg, resp)
	}

	return read
}

func AddSingleNodeTopologyHandle(n *maelstrom.Node) {
	n.Handle("topology", topologySingleNodeBuilder(n))
}

func topologySingleNodeBuilder(n *maelstrom.Node) maelstrom.HandlerFunc {
	topology := func(msg maelstrom.Message) error {
		// Ignore for now as we don't do anything with the topology yet.

		resp := make(map[string]any)
		resp["type"] = "topology_ok"

		return n.Reply(msg, resp)
	}

	return topology
}
