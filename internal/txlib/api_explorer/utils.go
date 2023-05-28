package api_explorer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"text/template"
	"unicode/utf8"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/transifex/cli/internal/txlib"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

func getApi(c *cli.Context) (*jsonapi.Connection, error) {
	token := c.String("token")
	var hostname string
	if token != "" {
		hostname = c.String("hostname")
		if hostname == "" {
			hostname = "https://rest.api.transifex.com"
		}
	} else {
		cfg, err := config.LoadFromPaths(
			c.String("root-config"),
			c.String("config"),
		)
		if err != nil {
			return nil, fmt.Errorf("error loading configuration: %s", err)
		}
		hostname, token, err = txlib.GetHostAndToken(
			&cfg,
			c.String("hostname"),
			c.String("token"),
		)
		if err != nil {
			return nil, fmt.Errorf("error getting API token: %s", err)
		}
	}

	client, err := txlib.GetClient(c.String("cacert"))
	if err != nil {
		return nil, fmt.Errorf("error getting HTTP client configuration: %s", err)
	}

	return &jsonapi.Connection{
		Host:    hostname,
		Token:   token,
		Client:  client,
		Headers: map[string]string{"Integration": "txclient"},
	}, nil
}

func invokePager(pager string, body []byte) error {
	var unmarshalled map[string]interface{}
	err := json.Unmarshal(body, &unmarshalled)
	if err != nil {
		return err
	}
	output, err := json.MarshalIndent(unmarshalled, "", "  ")
	if err != nil {
		return err
	}
	if pager != "" {
		cmd := exec.Command(pager)
		cmd.Stdin = bytes.NewBuffer(output)
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			return err
		}
	} else {
		_, err = fmt.Fprintln(os.Stdout, bytes.NewBuffer(output))
		if err != nil {
			return err
		}
	}
	return nil
}

func fuzzy(
	api *jsonapi.Connection,
	body []byte,
	header string,
	display string,
	allowEmpty bool,
	multi bool,
) ([]string, error) {
	var payload map[string]interface{}
	err := json.Unmarshal(body, &payload)
	if err != nil {
		return nil, err
	}
	items, err := jsonapi.PostProcessListResponse(api, body)
	if err != nil {
		return nil, err
	}

	var data []jsonapi.Resource
	if allowEmpty {
		data = append([]jsonapi.Resource{{}}, items.Data...)
	} else {
		data = append([]jsonapi.Resource{}, items.Data...)
	}

	displayFunc := func(i int) string {
		if allowEmpty && i == 0 {
			return "<empty>"
		}
		obj := data[i]
		result, err := renderTemplate(display, obj)
		if err != nil {
			return obj.Id
		}
		return result
	}

	previewOption := fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
		if i == -1 {
			return ""
		}
		if allowEmpty && i == 0 {
			return "Empty selection"
		}
		var idx int
		if allowEmpty {
			idx = i - 1
		} else {
			idx = i
		}
		item, err := json.MarshalIndent(
			payload["data"].([]interface{})[idx],
			"",
			"  ",
		)
		if err != nil {
			return ""
		}
		return string(item)
	})

	var indices []int
	if multi {
		indices, err = fuzzyfinder.FindMulti(
			data, displayFunc, previewOption, fuzzyfinder.WithHeader(header),
		)
		if err != nil {
			return nil, err
		}
	} else {
		index, err := fuzzyfinder.Find(
			data, displayFunc, previewOption, fuzzyfinder.WithHeader(header),
		)
		if err != nil {
			return nil, err
		}
		indices = append(indices, index)
	}
	var ids []string
	for _, index := range indices {
		ids = append(ids, data[index].Id)
	}
	return ids, nil
}

func renderTemplate(templateString string, context interface{}) (string, error) {
	t := template.New("")
	t, err := t.Parse(templateString)
	if err != nil {
		return "", err
	}
	buf := bytes.NewBufferString("")
	err = t.Execute(buf, context)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func stringSliceContains(haystack []string, needle string) bool {
	for _, key := range haystack {
		if key == needle {
			return true
		}
	}
	return false
}

func createObject(
	c *cli.Context, resourceName string, jsopenapi *jsopenapi_t, hasContent bool,
) (*jsonapi.Resource, error) {
	api, err := getApi(c)
	if err != nil {
		return nil, err
	}

	requiredRelationships := make(map[string]*jsonapi.Resource)
	operation := jsopenapi.Resources[resourceName].Operations.CreateOne
	for relationshipName, resourceName := range operation.Relationships.Required {
		resourceId := c.String(fmt.Sprintf("%s-id", relationshipName))
		if resourceId == "" {
			resourceIds, err := selectResourceIds(
				c, api, resourceName, relationshipName, jsopenapi, true, false,
			)
			if err != nil {
				return nil, err
			}
			resourceId = resourceIds[0]
		}
		requiredRelationships[relationshipName] = &jsonapi.Resource{
			Id:   resourceId,
			Type: resourceName,
		}
	}

	optionalRelationships := make(map[string]*jsonapi.Resource)
	for relationshipName, resourceName := range operation.Relationships.Optional {
		resourceId := c.String(fmt.Sprintf("%s-id", relationshipName))
		if resourceId == "" {
			resourceIds, err := selectResourceIds(
				c, api, resourceName, relationshipName, jsopenapi, false, false,
			)
			if err != nil {
				return nil, err
			}
			resourceId = resourceIds[0]
		}
		if resourceId != "" {
			optionalRelationships[relationshipName] = &jsonapi.Resource{
				Id:   resourceId,
				Type: resourceName,
			}
		}
	}

	attributes := make(map[string]interface{})
	var requiredAttributeNames []string
	for _, attributeName := range operation.Attributes.Required {
		if hasContent && (attributeName == "content" || attributeName == "content_encoding") {
			continue
		}
		value := c.String(attributeName)
		if value != "" {
			attributes[attributeName] = value
		} else {
			requiredAttributeNames = append(requiredAttributeNames, attributeName)
		}
	}
	var optionalAttributeNames []string
	for _, attributeName := range operation.Attributes.Optional {
		if hasContent && (attributeName == "content" || attributeName == "content_encoding") {
			continue
		}
		value := c.String(attributeName)
		if value != "" {
			attributes[attributeName] = value
		} else {
			optionalAttributeNames = append(optionalAttributeNames, attributeName)
		}
	}

	userSuppliedAttributes, err := create(
		c.String("editor"), requiredAttributeNames, optionalAttributeNames,
	)
	if err != nil {
		return nil, err
	}

	for key, value := range userSuppliedAttributes {
		attributes[key] = value
	}

	if hasContent {
		body, err := os.ReadFile(c.String("input"))
		if err != nil {
			return nil, err
		}
		if utf8.Valid(body) {
			attributes["content"] = string(body)
			attributes["content_encoding"] = "text"
		} else {
			attributes["content"] = base64.StdEncoding.EncodeToString(body)
			attributes["content_encoding"] = "base64"
		}
	}

	obj := &jsonapi.Resource{
		API:        api,
		Type:       resourceName,
		Attributes: attributes,
	}
	for relationshipName, resourceInfo := range requiredRelationships {
		obj.SetRelated(relationshipName, resourceInfo)
	}
	for relationshipName, resourceInfo := range optionalRelationships {
		obj.SetRelated(relationshipName, resourceInfo)
	}
	err = obj.Save(nil)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
