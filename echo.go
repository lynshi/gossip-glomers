package main

import (
	"encoding/json"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func AddEchoHandle(n *maelstrom.Node) {
	n.Handle("echo", echoBuilder(n))
}

func echoBuilder(n *maelstrom.Node) maelstrom.HandlerFunc {
	echo := func(msg maelstrom.Message) error {
		// Unmarshal the message body as an loosely-typed map.
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		// Update the message type to return back.
		body["type"] = "echo_ok"

		// Echo the original message back with the updated message type.
		return n.Reply(msg, body)
	}

	return echo
}
