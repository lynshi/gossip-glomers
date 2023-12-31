package unique_ids

import (
	"context"
	"strconv"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"github.com/pkg/errors"
)

func AddUniqueIdsHandle(ctx context.Context, n *maelstrom.Node) {
	// Use a counter to allocate node ids to easily support concurrent requests.
	counter := make(chan uint32, 1)
	i := uint32(0)

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(counter)
				return
			case counter <- i:
				i++
			}
		}
	}()

	n.Handle("generate", uniqueIdsBuilder(n, counter))
}

func uniqueIdsBuilder(n *maelstrom.Node, counter chan uint32) maelstrom.HandlerFunc {
	unique_ids := func(msg maelstrom.Message) error {
		// Node IDs have an 'n' in front (e.g. 'n1').
		node_id, err := strconv.Atoi(n.ID()[1:])
		if err != nil {
			return errors.Wrapf(err, "could not get integer from %s", n.ID())
		}

		id, ok := <-counter
		if !ok {
			return errors.New("counter channel closed")
		}

		// 1000 requests per second for 30 seconds = 30,000 ids needed (2^15).
		// So, we'll just left shift by 15. Note that we have to use a 32 bit int because the node
		// id has 2 bits as there are 3 nodes.
		//
		// This ensures a unique prefix per node, and each node just needs to create locally unique
		// ids.
		node_id = node_id << 15

		resp := make(map[string]any)

		resp["type"] = "generate_ok"
		resp["id"] = uint32(node_id) + id

		return n.Reply(msg, resp)
	}

	return unique_ids
}
