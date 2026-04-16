package timeline

import "encoding/json"

// UnmarshalPayloads takes a slice of Events and unmarshals their JSON payloads into a slice of type T.
func UnmarshalPayloads[T any](events []*Event) ([]T, error) {
	if len(events) == 0 {
		return nil, nil
	}

	payloadsJSON := make([]byte, 0, len(events)*128)
	payloadsJSON = append(payloadsJSON, '[')
	for i, event := range events {
		if i > 0 {
			payloadsJSON = append(payloadsJSON, ',')
		}
		payloadsJSON = append(payloadsJSON, event.Payload...)
	}
	payloadsJSON = append(payloadsJSON, ']')

	var results []T
	if err := json.Unmarshal(payloadsJSON, &results); err != nil {
		return nil, err
	}

	return results, nil
}

func UnmarshalPayload[T any](event *Event) (T, error) {
	var result T
	err := json.Unmarshal([]byte(event.Payload), &result)
	return result, err
}
