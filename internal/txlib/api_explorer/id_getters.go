package api_explorer

import (
	"fmt"
	"strings"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

func selectResourceIds(
	c *cli.Context,
	api *jsonapi.Connection,
	resourceName string,
	relationshipName string,
	jsopenapi *jsopenapi_t,
	required bool,
	multi bool,
) ([]string, error) {
	resource := jsopenapi.Resources[resourceName]

	if !required && resource.Operations.GetMany == nil {
		return []string{""}, nil
	}

	// Before we show a list of options, we need to fetch it. In order to do
	// so, we need to see if there are any filters
	query := jsonapi.Query{
		Filters: make(map[string]string),
		Extras:  make(map[string]string),
	}
	if resource.Operations.GetMany != nil {
		parameters := resource.Operations.GetMany.Parameters
		for _, parameter := range parameters {
			if parameter.Resource != "" {
				filterValue, err := getResourceId(
					c, api, parameter.Resource, jsopenapi, parameter.Required,
				)
				if err != nil {
					return nil, err
				}
				if filterValue != "" {
					queryName, err := getQueryName(parameter.Name)
					if err != nil {
						return nil, err
					}
					query.Filters[queryName] = filterValue
				}
			} else {
				flagName, err := getFlagName(parameter.Name)
				if err != nil {
					return nil, err
				}
				filterValue := c.String(flagName)
				if filterValue != "" {
					if strings.HasPrefix(parameter.Name, "filter[") {
						queryName, err := getQueryName(parameter.Name)
						if err != nil {
							return nil, err
						}
						query.Filters[queryName] = filterValue
					} else {
						query.Extras[parameter.Name] = filterValue
					}
				}
			}
		}
	}
	body, err := api.ListBody(resourceName, query.Encode())
	if err != nil {
		return nil, err
	}
	body, err = joinPages(api, body)
	if err != nil {
		return nil, err
	}

	isEmpty, err := getIsEmpty(body)
	if err != nil {
		return nil, err
	}
	if isEmpty && required {
		return nil, fmt.Errorf("%s not found", resource.SingularName)
	}
	if !multi && required {
		resourceId, err := getIfOnlyOne(body)
		if err != nil {
			return nil, err
		}
		if resourceId != "" {
			return []string{resourceId}, nil
		}
	}

	if c.Bool("no-interactive") {
		if required {
			return nil, fmt.Errorf(
				"more than one %s found, cannot proceed with --no-interactive",
				resource.PluralName,
			)
		} else {
			return []string{""}, nil
		}
	}

	var header string
	if relationshipName != "" {
		header = fmt.Sprintf("Select %s", relationshipName)
	} else if multi {
		header = fmt.Sprintf("Select %s", resource.PluralName)
	} else {
		header = fmt.Sprintf("Select %s", resource.SingularName)
	}

	return fuzzy(
		api,
		body,
		header,
		jsopenapi.Resources[resourceName].Display,
		!required,
		multi,
	)
}

func getResourceId(
	c *cli.Context,
	api *jsonapi.Connection,
	resourceName string,
	jsopenapi *jsopenapi_t,
	required bool,
) (string, error) {
	resource := jsopenapi.Resources[resourceName]
	resourceId := c.String(fmt.Sprintf("%s-id", resource.SingularName))
	if resourceId != "" {
		return resourceId, nil
	}
	resourceId, err := load(resource.SingularName)
	if err != nil {
		return "", err
	}
	if resourceId == "" {
		resourceIds, err := selectResourceIds(
			c, api, resourceName, "", jsopenapi, required, false,
		)
		if err != nil {
			return "", err
		}
		resourceId = resourceIds[0]
	}
	return resourceId, nil
}
