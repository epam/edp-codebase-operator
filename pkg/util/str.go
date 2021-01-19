package util

import (
	"encoding/json"
)

func SearchVersion(a []string, b string) bool {
	if len(a) == 0 {
		return false
	}

	for _, v := range a {
		if v == b {
			return true
		}
	}

	return false
}

func GetFieldsMap(payload string, keysToDelete []string) (map[string]interface{}, error) {
	requestPayload := make(map[string]interface{})
	if err := json.Unmarshal([]byte(payload), &requestPayload); err != nil {
		return nil, err
	}
	for k := range requestPayload {
		if keysToDelete != nil && ContainsString(keysToDelete, k) {
			delete(requestPayload, k)
		}
	}
	return requestPayload, nil
}
