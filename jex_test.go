package main

import (
	"strings"
	"testing"
)

func indexOrFail(t *testing.T, haystack, needle string) int {
	t.Helper()
	idx := strings.Index(haystack, needle)
	if idx < 0 {
		t.Fatalf("missing %q in output:\n%s", needle, haystack)
	}
	return idx
}

func TestJSONOrderPreservedAfterEdit(t *testing.T) {
	input := []byte(`{"first":true,"second":2,"third":{"x":1,"y":2}}`)
	data, err := parseJSON(input)
	if err != nil {
		t.Fatalf("parseJSON failed: %v", err)
	}

	root := data
	if err := setValueAtPath(&root, []pathToken{{Key: "first"}}, false); err != nil {
		t.Fatalf("setValueAtPath failed: %v", err)
	}

	out, err := encodeOrderedJSON(root)
	if err != nil {
		t.Fatalf("encodeOrderedJSON failed: %v", err)
	}

	text := string(out)
	firstIdx := indexOrFail(t, text, `"first"`)
	secondIdx := indexOrFail(t, text, `"second"`)
	thirdIdx := indexOrFail(t, text, `"third"`)
	if !(firstIdx < secondIdx && secondIdx < thirdIdx) {
		t.Fatalf("top-level key order changed:\n%s", text)
	}

	xIdx := indexOrFail(t, text, `"x"`)
	yIdx := indexOrFail(t, text, `"y"`)
	if !(xIdx < yIdx) {
		t.Fatalf("nested key order changed:\n%s", text)
	}
}

func TestYAMLOrderPreservedAfterEdit(t *testing.T) {
	input := []byte("first: true\nsecond: 2\nthird:\n  x: 1\n  y: 2\n")
	data, err := parseYAML(input)
	if err != nil {
		t.Fatalf("parseYAML failed: %v", err)
	}

	root := data
	if err := setValueAtPath(&root, []pathToken{{Key: "first"}}, false); err != nil {
		t.Fatalf("setValueAtPath failed: %v", err)
	}

	out, err := encodeOrderedYAML(root)
	if err != nil {
		t.Fatalf("encodeOrderedYAML failed: %v", err)
	}

	text := string(out)
	firstIdx := indexOrFail(t, text, "first:")
	secondIdx := indexOrFail(t, text, "second:")
	thirdIdx := indexOrFail(t, text, "third:")
	if !(firstIdx < secondIdx && secondIdx < thirdIdx) {
		t.Fatalf("top-level key order changed:\n%s", text)
	}

	xIdx := indexOrFail(t, text, "x:")
	yIdx := indexOrFail(t, text, "y:")
	if !(xIdx < yIdx) {
		t.Fatalf("nested key order changed:\n%s", text)
	}
}
