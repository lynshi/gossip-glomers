package broadcast

import (
	"context"
	"encoding/json"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"github.com/pkg/errors"
)

var messages chan []int

func AddBroadcastHandle(ctx context.Context, n *maelstrom.Node) {
	messages = make(chan []int, 1)

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

		message, ok := (body["message"]).(int)
		if !ok {
			return errors.Errorf("could not convert message %v to int", body["message"])
		}

		queue <- message

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
