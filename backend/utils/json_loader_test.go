package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTmp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return p
}

func TestLoadJSONFile_SingleObject(t *testing.T) {
	p := writeTmp(t, "single.json", `{"key":"value"}`)
	v, err := LoadJSONFile(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := v.(map[string]interface{})
	if !ok || m["key"].(string) != "value" {
		t.Fatalf("unexpected value: %#v", v)
	}
}

func TestLoadJSONFile_Array(t *testing.T) {
	p := writeTmp(t, "arr.json", `[{"a":1},{"b":2}]`)
	v, err := LoadJSONFile(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := v.([]interface{})
	if !ok || len(arr) != 2 {
		t.Fatalf("expected 2 elements: %#v", v)
	}
}

func TestLoadJSONFile_TwoConcatenated(t *testing.T) {
	p := writeTmp(t, "two.json", `{"a":1}{"b":2}`)
	v, err := LoadJSONFile(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := v.([]interface{})
	if !ok || len(arr) != 2 {
		t.Fatalf("expected 2 elements: %#v", v)
	}
}

func TestLoadJSONFile_FourConcatenated(t *testing.T) {
	p := writeTmp(t, "four.json", `{"a":1}{"b":2}{"c":3}{"d":4}`)
	v, err := LoadJSONFile(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := v.([]interface{})
	if !ok || len(arr) != 4 {
		t.Fatalf("expected 4 elements: %#v", v)
	}
}

func TestLoadJSONFile_WithNewlines(t *testing.T) {
	p := writeTmp(t, "nl.json", "{\"a\":1}\n{\"b\":2}")
	v, err := LoadJSONFile(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := v.([]interface{})
	if !ok || len(arr) != 2 {
		t.Fatalf("expected 2 elements: %#v", v)
	}
}

func TestLoadJSONFile_WithWhitespace(t *testing.T) {
	p := writeTmp(t, "ws.json", "{\"a\":1} {\"b\":2}")
	v, err := LoadJSONFile(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := v.([]interface{})
	if !ok || len(arr) != 2 {
		t.Fatalf("expected 2 elements: %#v", v)
	}
}

func TestLoadJSONFile_Empty(t *testing.T) {
	p := writeTmp(t, "empty.json", "")
	if _, err := LoadJSONFile(p); err == nil {
		t.Fatalf("expected error for empty file")
	}
}

func TestLoadJSONFile_InvalidJSON(t *testing.T) {
	p := writeTmp(t, "bad.json", "{invalid}")
	if _, err := LoadJSONFile(p); err == nil {
		t.Fatalf("expected error for invalid json")
	}
}

func TestLoadJSONFile_MixedValidInvalid(t *testing.T) {
	// Decoder stops at first invalid object; treat as error
	p := writeTmp(t, "mixed.json", `{"a":1}{invalid}{"c":3}`)
	if _, err := LoadJSONFile(p); err == nil {
		t.Fatalf("expected error for mixed valid/invalid")
	}
}
