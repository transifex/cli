package api_explorer_new

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

type jsopenapi_t struct {
	Resources map[string]struct {
		Description string `json:"description"`
		Operations  struct {
			GetMany *struct {
				Summary string `json:"summary"`
				Filters map[string]struct {
					Description string `json:"description"`
					Resource    string `json:"resource"`
					Required    bool   `json:"required"`
				} `json:"filters"`
			} `json:"get_many"`
			GetOne *struct {
				Summary string `json:"summary"`
			} `json:"get_one"`
		} `json:"operations"`
		Display string `json:"display"`
	} `json:"resources"`
}

func findSubcommand(subcommands []*cli.Command, name string) *cli.Command {
	for _, subcommand := range subcommands {
		if subcommand.Name == name {
			return subcommand
		}
	}
	return nil
}

//go:embed jsopenapi.json
var jsopenapi_bytes []byte

func Cmd() *cli.Command {
	var jsopenapi jsopenapi_t
	err := json.Unmarshal(jsopenapi_bytes, &jsopenapi)
	if err != nil {
		panic(err)
	}

	result := cli.Command{
		Name: "api_new",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "pager", EnvVars: []string{"PAGER"}},
			&cli.StringFlag{Name: "editor", EnvVars: []string{"EDITOR"}},
		},
	}

	for resourceName, resource := range jsopenapi.Resources {
		resourceNameCopy := resourceName

		if resource.Operations.GetMany != nil {
			subcommand := findSubcommand(result.Subcommands, "get")
			if subcommand == nil {
				subcommand = &cli.Command{Name: "get"}
				result.Subcommands = append(result.Subcommands, subcommand)
			}
			operation := cli.Command{
				Name:  resourceName,
				Usage: resource.Description,
				Action: func(c *cli.Context) error {
					return cliCmdGetMany(c, resourceNameCopy, &jsopenapi)
				},
			}
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
			for filterName, filter := range resource.Operations.GetMany.Filters {
				if filter.Resource != "" {
					operation.Flags = append(
						operation.Flags,
						&cli.StringFlag{
							Name: fmt.Sprintf(
								"%s-id",
								filter.Resource[:len(filter.Resource)-1],
							),
							Usage: filter.Description,
						},
					)
				} else {
					operation.Flags = append(
						operation.Flags,
						&cli.StringFlag{
							Name:     strings.ReplaceAll(filterName, "__", "-"),
							Usage:    filter.Description,
							Required: filter.Required,
						},
					)
				}
			}
		}

		if resource.Operations.GetOne != nil {
			subcommand := findSubcommand(result.Subcommands, "get")
			if subcommand == nil {
				subcommand = &cli.Command{Name: "get"}
				result.Subcommands = append(result.Subcommands, subcommand)
			}
			operation := cli.Command{
				Name:  resourceName[:len(resourceName)-1],
				Usage: resource.Description,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name: "id",
						// If we want to `get something` and the `somethings`
						// resource does not support `get_many`, then the user
						// won't be able to fuzzy-select the something and
						// `--id` should be required
						Required: resource.Operations.GetMany == nil,
					},
				},
				Action: func(c *cli.Context) error {
					return cliCmdGetOne(c, resourceNameCopy, &jsopenapi)
				},
			}
			if resource.Operations.GetMany != nil {
				for filterName, filter := range resource.Operations.GetMany.Filters {
					if filter.Resource != "" {
						operation.Flags = append(operation.Flags, &cli.StringFlag{
							Name:  fmt.Sprintf("%s-id", filterName),
							Usage: filter.Description,
						})
					} else {
						operation.Flags = append(
							operation.Flags,
							&cli.StringFlag{
								Name:     strings.ReplaceAll(filterName, "__", "-"),
								Usage:    filter.Description,
								Required: filter.Required,
							},
						)
					}
				}
			}
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
		}
	}

	return &result
}

func cliCmdGetMany(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	query := jsonapi.Query{Filters: make(map[string]string)}
	filters := jsopenapi.Resources[resourceName].Operations.GetMany.Filters
	for filterName, filter := range filters {
		if filter.Resource != "" {
			filterValue, err := getResourceId(
				c, api, filter.Resource, jsopenapi, filter.Required,
			)
			if err != nil {
				return err
			}
			if filterValue != "" {
				query.Filters[filterName] = filterValue
			}
		} else {
			filterValue := c.String(strings.ReplaceAll(filterName, "__", "-"))
			if filterValue != "" {
				query.Filters[filterName] = filterValue
			}
		}
	}
	body, err := api.ListBody(resourceName, query.Encode())
	if err != nil {
		return err
	}
	err = handlePagination(body)
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func selectResourceId(
	c *cli.Context,
	api *jsonapi.Connection,
	resourceName string,
	jsopenapi *jsopenapi_t,
	required bool,
) (string, error) {
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
					return "", err
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
		return "", err
	}
	body, err = joinPages(api, body)
	if err != nil {
		return "", err
	}

	isEmpty, err := getIsEmpty(body)
	if err != nil {
		return "", err
	}
	if isEmpty && required {
		return "", fmt.Errorf("%s not found", resourceName[:len(resourceName)-1])
	}
	resourceId, err := getIfOnlyOne(body)
	if err != nil {
		return "", err
	}
	if resourceId != "" {
		return resourceId, nil
	}

	resourceId, err = fuzzy(
		api,
		body,
		fmt.Sprintf("Select %s", resourceName[:len(resourceName)-1]),
		jsopenapi.Resources[resourceName].Display,
		!required,
	)
	if err != nil {
		return "", err
	}
	return resourceId, nil
}

func getResourceId(
	c *cli.Context,
	api *jsonapi.Connection,
	resourceName string,
	jsopenapi *jsopenapi_t,
	required bool,
) (string, error) {
	resourceId := c.String("id")
	if resourceId != "" {
		return resourceId, nil
	}
	resourceId = c.String(fmt.Sprintf("%s-id", resourceName[:len(resourceName)-1]))
	if resourceId != "" {
		return resourceId, nil
	}
	resourceId, err := load(resourceName[:len(resourceName)-1])
	if err != nil {
		return "", err
	}
	if resourceId == "" {
		resourceId, err = selectResourceId(c, api, resourceName, jsopenapi, required)
		if err != nil {
			return "", err
		}
	}
	return resourceId, nil
}

func cliCmdGetOne(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	resourceId, err := getResourceId(c, api, resourceName, jsopenapi, true)
	if err != nil {
		return err
	}
	body, err := api.GetBody(resourceName, resourceId)
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}
