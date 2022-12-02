package util

import (
	"encoding/json"
	"fmt"
)

func SearchVersion(a []string, b string) bool {
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

	for _, key := range keysToDelete {
		delete(requestPayload, key)
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
