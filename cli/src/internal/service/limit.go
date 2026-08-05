package service

import "encoding/json"

func limitJSONBody(body []byte, limit int) ([]byte, bool, error) {
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return body, false, err
	}

	switch value := data.(type) {
	case []any:
		if len(value) <= limit {
			return body, false, nil
		}
		out, err := json.Marshal(value[:limit])
		return out, err == nil, err
	case map[string]any:
		items, ok := value["value"].([]any)
		if !ok || len(items) <= limit {
			return body, false, nil
		}
		value["value"] = items[:limit]
		out, err := json.Marshal(value)
		return out, err == nil, err
	default:
		return body, false, nil
	}
}
