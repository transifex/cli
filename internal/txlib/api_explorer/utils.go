package api_explorer

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
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

	resource := jsopenapi.Resources[resourceName]
	operation := resource.Operations.CreateOne

	// What we'll end up prompting the user to edit
	editPayload := make(map[string]interface{})

	// What we'll end up sending to the API
	attributes := make(map[string]interface{})
	relationships := make(map[string]*jsonapi.Resource)

	for _, field := range operation.RequiredFields {
		if hasContent && (field == "content" || field == "content_encoding") {
			continue
		}
		_, isAttribute := resource.RequestAttributes[field]
		_, isRelationship := resource.RequestRelationships[field]
		if isAttribute {
			if c.String(field) != "" {
				value, err := intepretFlag(c, field, resource.RequestAttributes[field])
				if err != nil {
					return nil, err
				}
				attributes[field] = value
			} else {
				if c.Bool("no-interactive") {
					return nil, fmt.Errorf("%s not set, use the --%s flag", field, field)
				}
				editPayload[field] = prepareEditPayload(
					field, resource.RequestAttributes[field],
				)
			}
		} else if isRelationship {
			resourceId := c.String(fmt.Sprintf("%s-id", field))
			if resourceId == "" {
				resourceIds, err := selectResourceIds(
					c,
					api,
					resource.RequestRelationships[field].Resource,
					field,
					jsopenapi,
					true,
					false,
					nil,
				)
				if err != nil {
					return nil, err
				}
				resourceId = resourceIds[0]
			}
			relationships[field] = &jsonapi.Resource{
				Id:   resourceId,
				Type: resource.RequestRelationships[field].Resource,
			}
		} else {
			return nil, fmt.Errorf("unknown field %s of %s", field, resourceName)
		}
	}

	for _, field := range operation.OptionalFields {
		if hasContent && (field == "content" || field == "content_encoding") {
			continue
		}
		_, isAttribute := resource.RequestAttributes[field]
		_, isRelationship := resource.RequestRelationships[field]
		if isAttribute {
			if c.String(field) != "" {
				value, err := intepretFlag(c, field, resource.RequestAttributes[field])
				if err != nil {
					return nil, err
				}
				attributes[field] = value
			} else {
				editPayload[fmt.Sprintf("//%s", field)] = prepareEditPayload(
					field, resource.RequestAttributes[field],
				)
			}
		} else if isRelationship {
			resourceId := c.String(fmt.Sprintf("%s-id", field))
			if resourceId == "" && !c.Bool("no-interactive") {
				resourceIds, err := selectResourceIds(
					c,
					api,
					resource.RequestRelationships[field].Resource,
					field,
					jsopenapi,
					false,
					false,
					nil,
				)
				if err != nil {
					return nil, err
				}
				resourceId = resourceIds[0]
			}
			if resourceId != "" {
				relationships[field] = &jsonapi.Resource{
					Id:   resourceId,
					Type: resource.RequestRelationships[field].Resource,
				}
			}
		} else {
			return nil, fmt.Errorf("unknown field %s of %s", field, resourceName)
		}
	}

	if !c.Bool("no-interactive") && len(editPayload) > 0 {
		var fields []string
		fields = append(fields, operation.RequiredFields...)
		fields = append(fields, operation.OptionalFields...)
		userSuppliedAttributes, err := create(c.String("editor"), editPayload, fields)
		if err != nil {
			return nil, err
		}
		for key, value := range userSuppliedAttributes {
			attributes[key] = value
		}
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
	for relationshipName, relationship := range relationships {
		obj.SetRelated(relationshipName, relationship)
	}
	err = obj.Save(nil)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func input(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	return reader.ReadString('\n')
}

func intepretFlag(
	c *cli.Context, field string, jsonschema *jsonschema_t,
) (interface{}, error) {
	stringValue := c.String(field)
	if jsonschema.Type == "number" {
		return c.Int(field), nil
	} else if jsonschema.Type == "array" ||
		jsonschema.Type == "object" {
		var value interface{}
		json.Unmarshal([]byte(stringValue), &value)
		return value, nil
	} else if jsonschema.Type == "boolean" {
		if stringValue == "true" {
			return true, nil
		} else if stringValue == "false" {
			return false, nil
		} else {
			return nil, fmt.Errorf("--%s must either be 'true' or 'false'", field)
		}
	} else {
		return stringValue, nil
	}
}

func prepareEditPayload(field string, jsonschema *jsonschema_t) interface{} {
	if jsonschema.Type == "number" {
		return 0
	} else if jsonschema.Type == "boolean" {
		return false
	} else if jsonschema.Type == "array" {
		return []string{}
	} else if jsonschema.Type == "object" {
		obj := make(map[string]interface{})
		for _, objectField := range jsonschema.Required {
			obj[objectField] = ""
		}
		for objectField := range jsonschema.Properties {
			obj[fmt.Sprintf("//%s", objectField)] = ""
		}
		return obj
	} else if len(jsonschema.Enum) > 0 {
		return strings.Join(jsonschema.Enum, "/")
	} else {
		return ""
	}
}

// Turns 'a[b][c]' to a-b-c
func getFlagName(parameterName string) (string, error) {
	re, err := regexp.Compile(`[^\[\]]+`)
	if err != nil {
		return "", err
	}
	parts := re.FindAllString(parameterName, -1)
	return strings.Join(parts, "-"), nil
}

func getQuery(
	c *cli.Context, resourceName string, jsopenapi *jsopenapi_t,
) (*jsonapi.Query, error) {
	api, err := getApi(c)
	if err != nil {
		return nil, err
	}
	query := jsonapi.Query{Extras: make(map[string]string)}
	parameters := jsopenapi.Resources[resourceName].Operations.GetMany.Parameters
	for _, parameter := range parameters {
		flagName, err := getFlagName(parameter.Name)
		if err != nil {
			return nil, err
		}
		parameterValue := c.String(flagName)
		if parameterValue == "" && parameter.Resource != "" {
			parameterValue, err = getResourceId(
				c, api, parameter.Resource, jsopenapi, parameter.Required, nil,
			)
			if err != nil {
				return nil, err
			}
		}
		if parameterValue != "" {
			query.Extras[parameter.Name] = parameterValue
		}
	}
	return &query, nil
}
