package jsonapi

import (
	"testing"
)

func TestJsonEqual(t *testing.T) {
	left := `{
        "aaa": "bbb",
        "ccc": "ddd"
    }`
	// Change formatting and order
	right := `{"ccc": "ddd", "aaa": "bbb"}`

	equal, err := jsonEqual([]byte(left), []byte(right))
	if err != nil {
		t.Error(err)
	}
	if !equal {
		t.Error("JSON appears not equal")
	}
}
