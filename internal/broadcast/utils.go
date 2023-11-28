package broadcast

import (
	"reflect"

	"github.com/pkg/errors"
)

func getMessage(body map[string]any) (int, error) {
	message, ok := (body["message"]).(float64)
	if !ok {
		return 0, errors.Errorf(
			"could not convert message %v to int. Has type %v", body["message"],
			reflect.TypeOf(body["message"]))
	}

	return int(message), nil
}

func getNeighbors(nid string, body map[string]any) []string {
	topology := (body["topology"]).(map[string]interface{})
	neighbors := (topology[nid]).([]interface{})

	s := make([]string, len(neighbors))
	for i, v := range neighbors {
		s[i] = v.(string)
	}

	return s
}
