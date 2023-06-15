package api_explorer

import (
	"fmt"

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
	query *jsonapi.Query,
) ([]string, error) {
	resource := jsopenapi.Resources[resourceName]

	if !required && resource.Operations.GetMany == nil {
		return []string{""}, nil
	}

	if query == nil {
		query = &jsonapi.Query{Extras: make(map[string]string)}
	}
	parameters := jsopenapi.Resources[resourceName].Operations.GetMany.Parameters
	for _, parameter := range parameters {
		if !parameter.Required || parameter.Resource == "" {
			continue
		}
		_, exists := query.Extras[parameter.Name]
		if exists {
			continue
		}
		filterValue, err := getResourceId(
			c, api, parameter.Resource, jsopenapi, true, nil,
		)
		if err != nil {
			return nil, err
		}
		query.Extras[parameter.Name] = filterValue
	}

	queryString := ""
	if query != nil {
		queryString = query.Encode()
	}
	body, err := api.ListBody(resourceName, queryString)
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
	query *jsonapi.Query,
) (string, error) {
	resource := jsopenapi.Resources[resourceName]
	resourceId, err := load(resource.SingularName)
	if err != nil {
		return "", err
	}
	if resourceId == "" {
		resourceIds, err := selectResourceIds(
			c, api, resourceName, "", jsopenapi, required, false, query,
		)
		if err != nil {
			return "", err
		}
		resourceId = resourceIds[0]
	}
	return resourceId, nil
}
