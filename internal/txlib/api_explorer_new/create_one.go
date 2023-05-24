package api_explorer_new

import (
	"encoding/json"
	"fmt"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

func cliCmdCreateOne(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
  requiredResourceIds := make(map[string]string)
  optionalResourceIds := make(map[string]string)

	api, err := getApi(c)
	if err != nil {
		return err
	}

	resourceData := jsopenapi.Resources[resourceName]
  for required, path := range resourceData.Operations.CreateOne.Relationships.Required {
    resourceId, err := selectResourceId(c, api, path, jsopenapi, true)
    if err != nil {
      return err
    }
    requiredResourceIds[required] = resourceId
  }
  for optional, path := range resourceData.Operations.CreateOne.Relationships.Optional {
    resourceId, err := selectResourceId(c, api, path, jsopenapi, false)
    if err != nil {
      return err
    }
    if resourceId != "<empty>" {
      requiredResourceIds[optional] = resourceId
    }
  }

  validAttributes := resourceData.Operations.CreateOne.Attributes.Required
  validAttributes = append(validAttributes, resourceData.Operations.CreateOne.Attributes.Optional...)

	attributes, err := create(
		CREATE_ONE_STRING,
		c.String("editor"),
    validAttributes,
	)
	if err != nil {
		return err
	}
	resource := jsonapi.Resource{
		API:        api,
		Type:       resourceName,
		Attributes: attributes,
	}
  for required, resourceName := range resourceData.Operations.CreateOne.Relationships.Required {
    resource.SetRelated(required, &jsonapi.Resource{
      Type: resourceName,
      Id:   requiredResourceIds[required],
    })
  }
  for optional, resourceName := range resourceData.Operations.CreateOne.Relationships.Optional {
    if resourceId, ok := optionalResourceIds[optional]; ok {
      resource.SetRelated(optional, &jsonapi.Resource{Type: resourceName, Id: resourceId})
    }
  }
	err = resource.Save(nil)
	if err != nil {
		return err
	}
	fmt.Printf("Created %s: %s\n",resourceName[:len(resourceName)-1], resource.Id)
	return nil
}

func create(
	create_string,
	editor string,
	fields []string,
) (map[string]interface{}, error) {
	body, err := invokeEditor([]byte(create_string), editor)
	if err != nil {
		return nil, err
	}
	var attributes map[string]interface{}
	err = json.Unmarshal(body, &attributes)
	if err != nil {
		return nil, err
	}
	for field := range attributes {
		if !stringSliceContains(fields, field) {
			delete(attributes, field)
		}
	}
	return attributes, nil
}

