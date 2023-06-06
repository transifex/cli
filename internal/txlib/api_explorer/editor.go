package api_explorer

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"reflect"

	"github.com/google/shlex"
)

func invokeEditor(input []byte, editor string) ([]byte, error) {
	if editor == "" {
		return nil, errors.New(
			"no editor specified, use the --editor flag or set the EDITOR environment " +
				"variable",
		)
	}
	tempFile, err := os.CreateTemp("", "*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tempFile.Name())
	_, err = tempFile.Write(input)
	if err != nil {
		return nil, err
	}
	editorArgs, err := shlex.Split(editor)
	if err != nil {
		return nil, err
	}
	editorArgs = append(editorArgs, tempFile.Name())
	cmd := exec.Command(editorArgs[0], editorArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		return nil, err
	}
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	output, err := io.ReadAll(tempFile)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func edit(
	editor string, preAttributes map[string]interface{}, editable_fields []string,
) (map[string]interface{}, error) {
	for field := range preAttributes {
		if !stringSliceContains(editable_fields, field) {
			delete(preAttributes, field)
		}
	}
	body, err := json.MarshalIndent(preAttributes, "", "  ")
	if err != nil {
		return nil, err
	}
	body, err = invokeEditor(body, editor)
	if err != nil {
		return nil, err
	}
	var postAttributes map[string]interface{}
	err = json.Unmarshal(body, &postAttributes)
	if err != nil {
		return nil, err
	}
	for field, postValue := range postAttributes {
		preValue, exists := preAttributes[field]
		if !exists || reflect.DeepEqual(preValue, postValue) {
			delete(postAttributes, field)
		}
	}
	return postAttributes, nil
}

func create(
	editor string, editPayload map[string]interface{}, fields []string,
) (map[string]interface{}, error) {
	editPayload[""] = "A '//' in front of the attribute name implies that the key is " +
		"optional. Remove the slashes to include the field in the request payload."

	body, err := json.MarshalIndent(editPayload, "", "  ")
	if err != nil {
		return nil, err
	}
	body, err = invokeEditor(body, editor)
	if err != nil {
		return nil, err
	}
	var attributes map[string]interface{}
	err = json.Unmarshal(body, &attributes)
	if err != nil {
		return nil, err
	}
	for attr := range attributes {
		if !stringSliceContains(fields, attr) {
			delete(attributes, attr)
		}
	}
	return attributes, nil
}
