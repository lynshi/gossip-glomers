package unique_ids

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func AddUniqueIdsHandle(ctx context.Context, n *maelstrom.Node) {
	// Use a counter for keeping track of used node ids to easily allow for concurrent requests.
	counter := make(chan uint32, 1)
	i := uint32(0)

	go func() {
		for {
			select {
			case _, ok := <-ctx.Done():
				if !ok {
					close(counter)
					return
				}
			case counter <- i:
				i++
			}
		}
	}()

	n.Handle("generate", uniqueIdsBuilder(n, counter))
}

func uniqueIdsBuilder(n *maelstrom.Node, counter chan uint32) maelstrom.HandlerFunc {
	unique_ids := func(msg maelstrom.Message) error {
		node_id := n.ID()
		node_id_num, err := strconv.Atoi(node_id[1:])
		if err != nil {
			return fmt.Errorf("could not get integer from %s", node_id)
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
		node_id_num = node_id_num << 15

		resp := make(map[string]any)

		resp["type"] = "generate_ok"
		resp["id"] = uint32(node_id_num) + id

		return n.Reply(msg, resp)
	}

	return unique_ids
}
