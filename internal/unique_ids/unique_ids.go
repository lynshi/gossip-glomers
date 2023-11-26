package unique_ids

import (
	"context"
	"errors"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func AddUniqueIdsHandle(ctx context.Context, n *maelstrom.Node) {
	counter := make(chan int, 1)

	go func() {
		i := 0
		select {
		case _, ok := <-ctx.Done():
			if !ok {
				close(counter)
				return
			}
		case counter <- i:
			i++
		}
	}()

	n.Handle("generate", uniqueIdsBuilder(n, counter))
}

func uniqueIdsBuilder(n *maelstrom.Node, counter chan int) maelstrom.HandlerFunc {
	unique_ids := func(msg maelstrom.Message) error {
		node_id := n.ID()
		id, ok := <-counter
		if !ok {
			return errors.New("Counter channel closed")
		}

		// Unmarshal the message body as an loosely-typed map.
		resp := make(map[string]any)

		// Update the message type to return back.
		resp["type"] = "generate_ok"

		// Echo the original message back with the updated message type.
		return n.Reply(msg, resp)
	}

	return unique_ids
}
