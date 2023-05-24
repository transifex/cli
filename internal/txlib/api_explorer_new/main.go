package api_explorer_new

import (
	_ "embed"
	"encoding/json"
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
				} `json:"filters"`
			} `json:"get_many"`
		} `json:"operations"`
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
		subcommand := findSubcommand(result.Subcommands, "get")
		if subcommand == nil {
			subcommand = &cli.Command{Name: "get"}
			result.Subcommands = append(result.Subcommands, subcommand)
		}

		if resource.Operations.GetMany != nil {
			operation := cli.Command{
				Name:  resourceName,
				Usage: resource.Description,
				Action: func(c *cli.Context) error {
					return cliCmdGetMany(c, resourceNameCopy, &jsopenapi)
				},
			}
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
			for filterName, filter := range resource.Operations.GetMany.Filters {
				operation.Flags = append(
					operation.Flags,
					&cli.StringFlag{
						Name:  strings.ReplaceAll(filterName, "__", "-"),
						Usage: filter.Description,
					},
				)
			}
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
	for filterName := range filters {
		filterValue := c.String(strings.ReplaceAll(filterName, "__", "-"))
		if filterValue != "" {
			query.Filters[filterName] = filterValue
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
