package api_explorer_new

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"

	"github.com/transifex/cli/pkg/jsonapi"
)

func invokeEditor(input []byte, editor string) ([]byte, error) {
	tempFile, err := os.CreateTemp("", "*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tempFile.Name())
	_, err = tempFile.Write(input)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(editor, tempFile.Name())
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

func edit(editor string, item *jsonapi.Resource, editable_fields []string) error {
	var preAttributes map[string]interface{}
	err := item.MapAttributes(&preAttributes)
	if err != nil {
		return err
	}
	for field := range preAttributes {
		if !stringSliceContains(editable_fields, field) {
			delete(preAttributes, field)
		}
	}
	body, err := json.MarshalIndent(preAttributes, "", "  ")
	if err != nil {
		return err
	}
	body, err = invokeEditor(body, editor)
	if err != nil {
		return err
	}
	var postAttributes map[string]interface{}
	err = json.Unmarshal(body, &postAttributes)
	if err != nil {
		return err
	}
	var finalFields []string
	for field, postValue := range postAttributes {
		preValue, exists := preAttributes[field]
		if !exists || reflect.DeepEqual(preValue, postValue) {
			delete(postAttributes, field)
		} else {
			finalFields = append(finalFields, field)
		}
	}
	if len(finalFields) == 0 {
		return errors.New("nothing changed")
	}
	item.Attributes = postAttributes
	err = item.Save(finalFields)
	if err != nil {
		return err
	}
	return nil
}

func create(editor string, required_attrs, optional_attrs []string) (map[string]interface{}, error) {
	preAttributes := map[string]interface{}{
		" ": "A '//' infront of the attribute name implies that the key is optional. Remove the slashes to include the field in the request payload.",
	}

	for _, attr := range optional_attrs {
		preAttributes[fmt.Sprintf("//%s", attr)] = ""
	}

	for _, attr := range required_attrs {
		preAttributes[attr] = ""
	}

	body, err := json.MarshalIndent(preAttributes, "", "  ")
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
	validAttributes := required_attrs
	validAttributes = append(validAttributes, optional_attrs...)
	for attr := range attributes {
		if !stringSliceContains(validAttributes, attr) {
			delete(attributes, attr)
		}
	}
	return attributes, nil
}
