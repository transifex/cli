package api_explorer_new

import (
	"encoding/json"
	"fmt"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

type resourceInfo struct {
	id   string
	path string
}

func cliCmdCreateOne(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
	requiredResourceInfo := make(map[string]*resourceInfo)
	optionalResourceInfo := make(map[string]*resourceInfo)

	api, err := getApi(c)
	if err != nil {
		return err
	}

	resourceData := jsopenapi.Resources[resourceName]
	for required, path := range resourceData.Operations.CreateOne.Relationships.Required {
		resourceIds, err := selectResourceIds(c, api, path, jsopenapi, true, false)
		if err != nil {
			return err
		}
		resourceId := resourceIds[0]
		requiredResourceInfo[required] = &resourceInfo{id: resourceId, path: path}
	}
	for optional, path := range resourceData.Operations.CreateOne.Relationships.Optional {
		resourceIds, err := selectResourceIds(c, api, path, jsopenapi, false, false)
		if err != nil {
			return err
		}
		resourceId := resourceIds[0]
		if resourceId != "<empty>" {
			optionalResourceInfo[optional] = &resourceInfo{id: resourceId, path: path}
		}
	}

	attributes, err := create(
		c.String("editor"),
		resourceData.Operations.CreateOne.Attributes.Required,
		resourceData.Operations.CreateOne.Attributes.Optional,
	)
	if err != nil {
		return err
	}
	resource := jsonapi.Resource{
		API:        api,
		Type:       resourceName,
		Attributes: attributes,
	}
	for required, resourceInfo := range requiredResourceInfo {
		resource.SetRelated(required, &jsonapi.Resource{
			Type: resourceInfo.path,
			Id:   requiredResourceInfo[required].id,
		})
	}
	for optional, resourceInfo := range optionalResourceInfo {
		resource.SetRelated(optional, &jsonapi.Resource{Type: resourceInfo.path, Id: resourceInfo.id})
	}
	err = resource.Save(nil)
	if err != nil {
		return err
	}
	fmt.Printf("Created %s: %s\n", resourceName[:len(resourceName)-1], resource.Id)
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
