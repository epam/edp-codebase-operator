package util

import (
	"encoding/json"
	"fmt"
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

	err := json.Unmarshal([]byte(payload), &requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json payload: %w", err)
	}

	for k := range requestPayload {
		if keysToDelete != nil && ContainsString(keysToDelete, k) {
			delete(requestPayload, k)
		}
	}

	return requestPayload, nil
}

func CheckElementInArray(array []string, element string) bool {
	for _, elementCandidate := range array {
		if elementCandidate == element {
			return true
		}
	}

	return false
}
