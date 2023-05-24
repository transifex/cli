package api_explorer_new

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
			EditOne *struct {
				Summary string   `json:"summary"`
				Fields  []string `json:"fields"`
			} `json:"edit_one"`
			Delete *struct {
				Summary string `json:"summary"`
			} `json:"delete"`
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
		Subcommands: []*cli.Command{
			{
				Name: "get",
				Subcommands: []*cli.Command{
					{
						Name:  "next",
						Usage: "Get the next page of the last request",
						Action: func(c *cli.Context) error {
							api, err := getApi(c)
							if err != nil {
								return err
							}
							url, err := load("next")
							if err != nil {
								return err
							}
							if url == "" {
								return errors.New(
									"last request did not have a next page",
								)
							}
							body, err := api.ListBodyFromPath(url)
							if err != nil {
								return err
							}
							err = handlePagination(body)
							if err != nil {
								return err
							}
							err = invokePager(c.String("pager"), body)
							if err != nil {
								return err
							}
							return nil
						},
					},
					{
						Name:  "previous",
						Usage: "Get the previous page of the last request",
						Action: func(c *cli.Context) error {
							api, err := getApi(c)
							if err != nil {
								return err
							}
							url, err := load("previous")
							if err != nil {
								return err
							}
							if url == "" {
								return errors.New(
									"last request did not have a previous page",
								)
							}
							body, err := api.ListBodyFromPath(url)
							if err != nil {
								return err
							}
							err = handlePagination(body)
							if err != nil {
								return err
							}
							err = invokePager(c.String("pager"), body)
							if err != nil {
								return err
							}
							return nil
						},
					},
				},
			},
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
			operation.Flags = append(
				operation.Flags, getFilterFlags(resourceName, &jsopenapi)...,
			)
		}

		if resource.Operations.GetOne != nil {
			subcommand := findSubcommand(result.Subcommands, "get")
			if subcommand == nil {
				subcommand = &cli.Command{Name: "get"}
				result.Subcommands = append(result.Subcommands, subcommand)
			}
			operation := cli.Command{
				Name:  resourceName[:len(resourceName)-1],
				Usage: resource.Operations.GetOne.Summary,
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
			operation.Flags = append(
				operation.Flags, getFilterFlags(resourceName, &jsopenapi)...,
			)
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
		}

		if resource.Operations.EditOne != nil {
			subcommand := findSubcommand(result.Subcommands, "edit")
			if subcommand == nil {
				subcommand = &cli.Command{Name: "edit"}
				result.Subcommands = append(result.Subcommands, subcommand)
			}
			operation := cli.Command{
				Name:  resourceName[:len(resourceName)-1],
				Usage: resource.Operations.EditOne.Summary,
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
					return cliCmdEditOne(c, resourceNameCopy, &jsopenapi)
				},
			}
			operation.Flags = append(
				operation.Flags, getFilterFlags(resourceName, &jsopenapi)...,
			)
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
		}

		if resource.Operations.Delete != nil {
			subcommand := findSubcommand(result.Subcommands, "delete")
			if subcommand == nil {
				subcommand = &cli.Command{Name: "delete"}
				result.Subcommands = append(result.Subcommands, subcommand)
			}
			operation := cli.Command{
				Name:  resourceName[:len(resourceName)-1],
				Usage: resource.Description,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id"},
				},
				Action: func(c *cli.Context) error {
					return cliCmdDelete(c, resourceNameCopy, &jsopenapi)
				},
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
	err = invokePager(c.String("pager"), body)
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
	err = invokePager(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdEditOne(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	resourceId, err := getResourceId(c, api, resourceName, jsopenapi, true)
	if err != nil {
		return err
	}
	resource, err := api.Get(resourceName, resourceId)
	if err != nil {
		return err
	}
	err = edit(
		c.String("editor"),
		&resource,
		jsopenapi.Resources[resourceName].Operations.EditOne.Fields,
	)
	if err != nil {
		return err
	}
	return nil
}

func getFilterFlags(resourceName string, jsopenapi *jsopenapi_t) []cli.Flag {
	var result []cli.Flag
	resource := jsopenapi.Resources[resourceName]
	if resource.Operations.GetMany == nil {
		return result
	}
	for filterName, filter := range resource.Operations.GetMany.Filters {
		if filter.Resource != "" {
			result = append(
				result,
				&cli.StringFlag{
					Name:  fmt.Sprintf("%s-id", filterName),
					Usage: filter.Description,
				},
			)
		} else {
			result = append(
				result,
				&cli.StringFlag{
					Name:     strings.ReplaceAll(filterName, "__", "-"),
					Usage:    filter.Description,
					Required: filter.Required,
				},
			)
		}
	}
	return result
}

func cliCmdDelete(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	resourceId, err := getResourceId(c, api, resourceName, jsopenapi, true)
	if err != nil {
		return err
	}
	fmt.Printf("About to delete %s: %s, are you sure (y/N)? ", resourceName[:len(resourceName)-1], resourceId)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(strings.ToLower(answer)) == "y" {
		resource := jsonapi.Resource{API: api, Type: resourceName, Id: resourceId}
		err = resource.Delete()
		if err != nil {
			return err
		}
		fmt.Printf("Deleted %s: %s\n", resourceName[:len(resourceName)-1], resourceId)
	} else {
		fmt.Printf("Deletion aborted\n")
	}
	return nil
}
