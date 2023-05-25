package api_explorer_new

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
	jsopenapi *jsopenapi_t,
	required bool,
	multi bool,
) ([]string, error) {
	// Before we show a list of options, we need to fetch it. In order to do
	// so, we need to see if there are any filters
	query := jsonapi.Query{Filters: make(map[string]string)}
	if jsopenapi.Resources[resourceName].Operations.GetMany != nil {
		filters := jsopenapi.Resources[resourceName].Operations.GetMany.Filters
		for filterName, filter := range filters {
			if filter.Resource != "" {
				filterValue, err := getResourceId(
					c, api, filter.Resource, jsopenapi, filter.Required,
				)
				if err != nil {
					return nil, err
				}
				if filterValue != "" {
					query.Filters[filterName] = filterValue
				}
			} else {
				filterValue := c.String(
					strings.ReplaceAll(filterName, "__", "-"),
				)
				if filterValue != "" {
					query.Filters[filterName] = filterValue
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
		return nil, fmt.Errorf("%s not found", resourceName[:len(resourceName)-1])
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

	return fuzzy(
		api,
		body,
		fmt.Sprintf("Select %s", resourceName[:len(resourceName)-1]),
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
	resourceId := c.String(fmt.Sprintf("%s-id", resourceName[:len(resourceName)-1]))
	if resourceId != "" {
		return resourceId, nil
	}
	resourceId, err := load(resourceName[:len(resourceName)-1])
	if err != nil {
		return "", err
	}
	if resourceId == "" {
		resourceIds, err := selectResourceIds(
			c, api, resourceName, jsopenapi, required, false,
		)
		if err != nil {
			return "", err
		}
		resourceId = resourceIds[0]
	}
	return resourceId, nil
}
