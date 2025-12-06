package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// FormatJSONField formats a JSON field for display
func FormatJSONField(data []byte, field string) (interface{}, error) {
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, err
	}
	return jsonData[field], nil
}

// ExtractField extracts a field from a JSON object
func ExtractField(jsonData []byte, fieldPath string) (interface{}, error) {
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, err
	}

	fields := strings.Split(fieldPath, ".")
	current := data

	for _, field := range fields {
		if currentMap, ok := current.(map[string]interface{}); ok {
			if val, exists := currentMap[field]; exists {
				current = val
			} else {
				return nil, fmt.Errorf("field '%s' not found in path '%s'", field, fieldPath)
			}
		} else {
			return nil, fmt.Errorf("cannot traverse field '%s' in path '%s'", field, fieldPath)
		}
	}

	return current, nil
}

// MergeMaps merges multiple maps into one
func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// Contains checks if a slice contains a value
func Contains(slice interface{}, value interface{}) bool {
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice {
		return false
	}

	for i := 0; i < sliceValue.Len(); i++ {
		if reflect.DeepEqual(sliceValue.Index(i).Interface(), value) {
			return true
		}
	}
	return false
}

// UniqueStrings returns unique strings from a slice
func UniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	for _, str := range slice {
		if !keys[str] {
			keys[str] = true
			result = append(result, str)
		}
	}
	return result
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// FormatBilingualText formats bilingual text for display
func FormatBilingualText(text map[string]interface{}, lang string) string {
	if text == nil {
		return ""
	}

	// Try requested language first
	if val, ok := text[lang].(string); ok && val != "" {
		return val
	}

	// Fallback to English
	if val, ok := text["en"].(string); ok && val != "" {
		return val
	}

	// Fallback to Arabic
	if val, ok := text["ar"].(string); ok && val != "" {
		return val
	}

	// Return first available value
	for _, val := range text {
		if str, ok := val.(string); ok && str != "" {
			return str
		}
	}

	return ""
}

