package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

func PrintResponse(resp interface{}) error {
	outType := "json"
	switch outType {
	case "json":
		JSONOutput(resp)
	default:
		return fmt.Errorf("Unrecognized output format")
	}
	return nil
}

func JSONOutput(t interface{}) {
	output, _ := JSONMarshal(t)
	fmt.Println(string(output))
}
