package test

import (
	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/Bedrock-OSS/regolith/regolith"
	"testing"
)

func TestSimplePath(t *testing.T) {
	obj := map[string]any{
		"foo": "bar",
	}
	expected := "bar"
	actual, err := regolith.FindByJSONPath[string](obj, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

func TestSimplePath2(t *testing.T) {
	obj := map[string]any{
		"foo": []any{"bar"},
	}
	expected := "bar"
	actual, err := regolith.FindByJSONPath[string](obj, "foo/0")
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

func TestSimplePath3(t *testing.T) {
	obj := map[string]any{
		"foo": map[string]any{
			"bar": "baz",
		},
	}
	expected := "baz"
	actual, err := regolith.FindByJSONPath[string](obj, "foo/bar")
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

func TestEscapedPath(t *testing.T) {
	obj := map[string]any{
		"fo/o": "bar",
	}
	expected := "bar"
	actual, err := regolith.FindByJSONPath[string](obj, "fo\\/o")
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

func TestEscapedPath2(t *testing.T) {
	obj := map[string]any{
		"fo/o": "bar",
	}
	expected := "bar"
	actual, err := regolith.FindByJSONPath[string](obj, regolith.EscapePathPart("fo/o"))
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

func TestInvalidPath(t *testing.T) {
	obj := map[string]any{
		"foo": map[string]any{
			"bar": "baz",
		},
	}
	expected := "Invalid data type.\nJSON Path: foo->bar->baz\nExpected type: object or array"
	_, err := regolith.FindByJSONPath[string](obj, "foo/bar/baz")
	if err == nil {
		t.Fatal("Expected an error, got nil")
	}
	if burrito.GetAllMessages(err)[0] != expected {
		t.Fatalf("Expected error %v, got %v", expected, burrito.GetAllMessages(err)[0])
	}
}

func TestInvalidPath2(t *testing.T) {
	obj := map[string]any{
		"foo": map[string]any{
			"bar": "baz",
		},
	}
	expected := "Required JSON path is missing.\nJSON Path: foo->0"
	_, err := regolith.FindByJSONPath[string](obj, "foo/0/baz")
	if err == nil {
		t.Fatal("Expected an error, got nil")
	}
	if burrito.GetAllMessages(err)[0] != expected {
		t.Fatalf("Expected error %v, got %v", expected, burrito.GetAllMessages(err)[0])
	}
}

func TestNullObject(t *testing.T) {
	expected := "Object is empty"
	_, err := regolith.FindByJSONPath[string](nil, "foo/bar/baz")
	if err == nil {
		t.Fatal("Expected an error, got nil")
	}
	if burrito.GetAllMessages(err)[0] != expected {
		t.Fatalf("Expected error %v, got %v", expected, burrito.GetAllMessages(err)[0])
	}
}
