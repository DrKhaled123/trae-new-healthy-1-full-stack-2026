package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LoadJSONFile loads JSON that may be:
// - a single object
// - an array of objects
// - a stream of concatenated JSON objects (object}{object}{...)
// - concatenated objects with newlines ({...}\n{...})
// It returns either a single object or a []interface{} of objects.
func LoadJSONFile(filePath string) (interface{}, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Read entire file content to detect concatenated objects
	content, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	// Check if this contains concatenated objects by looking for any closing brace followed by optional whitespace and an opening brace
	trim := strings.TrimSpace(string(content))
	if strings.Contains(trim, "}{") || strings.Contains(trim, "}\n{") || strings.Contains(trim, "\r\n{") || strings.Contains(trim, "\t{") || strings.Contains(trim, "} {") || strings.Contains(trim, "\r{") {
		return parseConcatenatedObjects(content)
	}

	// Use standard JSON parsing for single objects/arrays
	dec := json.NewDecoder(bufio.NewReader(bytes.NewReader(content)))
	dec.UseNumber()

	// Peek first non-whitespace token
	t, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to read first token: %w", err)
	}

	switch delim := t.(type) {
	case json.Delim:
		// Array or object
		if delim == '[' {
			var arr []interface{}
			for dec.More() {
				var v interface{}
				if err := dec.Decode(&v); err != nil {
					return nil, fmt.Errorf("failed to decode array element: %w", err)
				}
				arr = append(arr, v)
			}
			// consume closing ']'
			if _, err := dec.Token(); err != nil {
				return nil, fmt.Errorf("failed to consume array end: %w", err)
			}
			return arr, nil
		}
		if delim == '{' {
			// Single object - reconstruct decoder and parse
			dec = json.NewDecoder(bufio.NewReader(bytes.NewReader(content)))
			dec.UseNumber()

			var obj interface{}
			if err := dec.Decode(&obj); err != nil {
				return nil, fmt.Errorf("failed to decode object: %w", err)
			}
			return obj, nil
		}
	default:
		// Primitive first token is unexpected for our data files
		return nil, fmt.Errorf("unexpected JSON token at start: %v", t)
	}
	// Defensive fallback: should not reach here
	return nil, fmt.Errorf("unexpected end of JSON parsing")
}

// parseConcatenatedObjects parses files with concatenated JSON objects
// Handles patterns like: {...}{...} or {...}\n{...}\n{...}
// Uses brace counting to properly handle nested objects
func parseConcatenatedObjects(content []byte) (interface{}, error) {
	objects := make([]interface{}, 0)
	contentStr := string(content)

	// Use a more robust approach: find object boundaries by counting braces
	// This handles nested objects correctly
	var startPos int = -1
	braceDepth := 0
	inString := false
	escapeNext := false

	for i := 0; i < len(contentStr); i++ {
		char := contentStr[i]

		// Handle string literals (braces inside strings don't count)
		if escapeNext {
			escapeNext = false
			continue
		}
		if char == '\\' {
			escapeNext = true
			continue
		}
		if char == '"' {
			inString = !inString
			continue
		}

		// Only count braces outside of strings
		if !inString {
			if char == '{' {
				if braceDepth == 0 {
					startPos = i // Start of a new object
				}
				braceDepth++
			} else if char == '}' {
				braceDepth--
				if braceDepth == 0 && startPos >= 0 {
					// Found complete object from startPos to i+1
					objStr := contentStr[startPos : i+1]

					// Try to parse this object
					var obj interface{}
					if err := json.Unmarshal([]byte(objStr), &obj); err != nil {
						// If parsing fails, try with trimmed whitespace
						objStr = strings.TrimSpace(objStr)
						if err := json.Unmarshal([]byte(objStr), &obj); err != nil {
							// Skip malformed objects but continue parsing others
							continue
						}
					}

					if obj != nil {
						objects = append(objects, obj)
					}
					startPos = -1
				}
			}
		}
	}

	// If we found objects, return them
	if len(objects) > 0 {
		if len(objects) == 1 {
			return objects[0], nil
		}
		return objects, nil
	}

	// Enhanced approach: try streaming method first for better reliability
	return parseConcatenatedObjectsStreaming(content)
}

// parseConcatenatedObjectsStreaming uses json.Decoder to parse concatenated objects
// This is more robust for complex concatenated JSON files
func parseConcatenatedObjectsStreaming(content []byte) (interface{}, error) {
	objects := make([]interface{}, 0)
	contentStr := string(content)

	// Normalize various separator patterns to a consistent format
	normalized := strings.ReplaceAll(contentStr, "}\n{", "}{")
	normalized = strings.ReplaceAll(normalized, "\r\n{", "}{")
	normalized = strings.ReplaceAll(normalized, "\t{", "}{")
	normalized = strings.ReplaceAll(normalized, "} {", "}{")
	normalized = strings.ReplaceAll(normalized, "\r{", "}{")

	// If no concatenation patterns found, try parsing as single object
	if !strings.Contains(normalized, "}{") {
		var obj interface{}
		if err := json.Unmarshal(content, &obj); err == nil {
			return obj, nil
		}
	}

	// Split by normalized separator and parse each object
	objectStrings := strings.Split(normalized, "}{")

	for i, objStr := range objectStrings {
		// Add back braces that were removed by splitting
		if i > 0 {
			objStr = "{" + objStr
		}
		if i < len(objectStrings)-1 {
			objStr = objStr + "}"
		}

		var obj interface{}
		if err := json.Unmarshal([]byte(objStr), &obj); err != nil {
			// Try trimming whitespace
			objStr = strings.TrimSpace(objStr)
			if err := json.Unmarshal([]byte(objStr), &obj); err != nil {
				// Skip malformed objects but continue parsing others
				continue
			}
		}

		if obj != nil {
			objects = append(objects, obj)
		}
	}

	// If we found objects, return them
	if len(objects) > 0 {
		if len(objects) == 1 {
			return objects[0], nil
		}
		return objects, nil
	}

	// Final fallback: try original method
	return parseConcatenatedObjectsFallback(content)
}

// parseConcatenatedObjectsFallback is the original implementation
// Used as fallback when brace counting doesn't work
func parseConcatenatedObjectsFallback(content []byte) (interface{}, error) {
	contentStr := string(content)
	objects := make([]interface{}, 0)

	// Normalize separators like '}\n{' or '} \t{' to plain '}{'
	contentStr = strings.ReplaceAll(contentStr, "}\n{", "}{")
	contentStr = strings.ReplaceAll(contentStr, "\r\n{", "}{")
	contentStr = strings.ReplaceAll(contentStr, "\t{", "}{")
	contentStr = strings.ReplaceAll(contentStr, "} {", "}{")
	contentStr = strings.ReplaceAll(contentStr, "\r{", "}{")

	// Use json.Decoder for more robust parsing
	dec := json.NewDecoder(strings.NewReader(contentStr))
	dec.UseNumber()

	for {
		var obj interface{}
		err := dec.Decode(&obj)
		if err == io.EOF {
			break
		}
		if err != nil {
			// Try to continue with next object
			continue
		}
		if obj != nil {
			objects = append(objects, obj)
		}
	}

	// If we found objects, return them
	if len(objects) > 0 {
		if len(objects) == 1 {
			return objects[0], nil
		}
		return objects, nil
	}

	// Last resort: try parsing as single object
	var obj interface{}
	if err := json.Unmarshal(content, &obj); err == nil {
		return obj, nil
	}

	return nil, fmt.Errorf("failed to parse concatenated JSON objects")
}

// LoadJSONFileFromDir loads a JSON file from a directory
func LoadJSONFileFromDir(dataDir, filename string) (interface{}, error) {
	filePath := filepath.Join(dataDir, filename)
	return LoadJSONFile(filePath)
}
