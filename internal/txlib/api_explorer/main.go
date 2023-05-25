package api_explorer

import (
	"bufio"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
	"unicode/utf8"

	"os"
	"strings"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

type jsopenapi_t struct {
	Resources map[string]struct {
		SingularName string `json:"singular_name"`
		PluralName   string `json:"plural_name"`
		Upload       bool   `json:"upload"`
		Download     bool   `json:"download"`
		Operations   struct {
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
			CreateOne *struct {
				Summary    string `json:"summary"`
				Attributes *struct {
					Required []string `json:"required"`
					Optional []string `json:"optional"`
				} `json:"attributes"`
				Relationships *struct {
					Required map[string]string `json:"required"`
					Optional map[string]string `json:"optional"`
				} `json:"relationships"`
			} `json:"create_one"`
			EditOne *struct {
				Summary string   `json:"summary"`
				Fields  []string `json:"fields"`
			} `json:"edit_one"`
			Delete *struct {
				Summary string `json:"summary"`
			} `json:"delete"`
		} `json:"operations"`
		Relationships map[string]struct {
			Resource   string `json:"resource"`
			Operations struct {
				Change *struct {
					Summary string `json:"summary"`
				} `json:"change"`
				Get *struct {
					Summary string `json:"summary"`
				} `json:"get"`
				Add *struct {
					Summary string `json:"summary"`
				} `json:"add"`
				Remove *struct {
					Summary string `json:"summary"`
				} `json:"remove"`
				Reset *struct {
					Summary string `json:"summary"`
				} `json:"reset"`
			} `json:"operations"`
		} `json:"relationships"`
		Display string `json:"display"`
	} `json:"resources"`
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
		Name:  "api",
		Usage: "Transifex API explorer",
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
					{
						Name:  "session",
						Usage: "Get current session",
						Action: func(c *cli.Context) error {
							_, err := os.Stat(".tx/api_explorer_session.json")
							if err != nil {
								return err
							}
							body, err := os.ReadFile(".tx/api_explorer_session.json")
							if err != nil {
								return err
							}
							fmt.Println(string(body))
							return nil
						},
					},
				},
			},
			{
				Name: "clear",
				Subcommands: []*cli.Command{
					{
						Name:  "session",
						Usage: "Clear session file",
						Action: func(c *cli.Context) error {
							err := os.Remove(".tx/api_explorer_session.json")
							if err != nil {
								return err
							}
							fmt.Printf(
								"Removed .tx/api_explorer_session.json successfully\n",
							)
							return nil
						},
					},
				},
			},
		},
	}

	for resourceName := range jsopenapi.Resources {
		// In order to avoid the 'cannot assign to struct field in map' error,
		// we have to extract a copy of each resource, modify its PluralName
		// and SingularName and then put it back
		resource := jsopenapi.Resources[resourceName]
		if resource.PluralName == "" {
			resource.PluralName = resourceName
		}
		if resource.SingularName == "" {
			resource.SingularName = resource.PluralName[:len(resource.PluralName)-1]
		}
		jsopenapi.Resources[resourceName] = resource
	}

	for resourceName, resource := range jsopenapi.Resources {
		resourceNameCopy := resourceName

		if resource.Operations.GetMany != nil {
			subcommand := getOrCreateSubcommand(&result, "get")
			operation := cli.Command{
				Name:  resource.PluralName,
				Usage: resource.Operations.GetMany.Summary,
				Action: func(c *cli.Context) error {
					return cliCmdGetMany(c, resourceNameCopy, &jsopenapi)
				},
			}
			addFilterTags(&operation, resourceName, &jsopenapi, false)
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
		}

		if resource.Operations.GetOne != nil && !resource.Upload && !resource.Download {
			subcommand := getOrCreateSubcommand(&result, "get")
			operation := cli.Command{
				Name:  resource.SingularName,
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
			addFilterTags(&operation, resourceName, &jsopenapi, true)
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
		}

		if resource.Operations.EditOne != nil {
			subcommand := getOrCreateSubcommand(&result, "edit")
			operation := cli.Command{
				Name:  resource.SingularName,
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
			addFilterTags(&operation, resourceName, &jsopenapi, true)
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
		}

		if resource.Operations.CreateOne != nil &&
			!resource.Upload &&
			!resource.Download {
			subcommand := getOrCreateSubcommand(&result, "create")
			operation := cli.Command{
				Name:  resource.SingularName,
				Usage: resource.Operations.CreateOne.Summary,
				Action: func(c *cli.Context) error {
					return cliCmdCreateOne(c, resourceNameCopy, &jsopenapi)
				},
			}
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
		}

		if resource.Operations.Delete != nil {
			subcommand := getOrCreateSubcommand(&result, "delete")
			operation := cli.Command{
				Name:  resource.SingularName,
				Usage: resource.Operations.Delete.Summary,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id"},
				},
				Action: func(c *cli.Context) error {
					return cliCmdDelete(c, resourceNameCopy, &jsopenapi)
				},
			}
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
		}
		if resource.Download {
			subcommand := getOrCreateSubcommand(&result, "download")
			operation := cli.Command{
				Name:  resource.SingularName,
				Usage: resource.Operations.GetOne.Summary,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}},
					&cli.IntFlag{
						Name:    "interval",
						Aliases: []string{"t"},
						Value:   2,
					},
				},
				Action: func(c *cli.Context) error {
					return cliCmdDownload(c, resourceNameCopy, &jsopenapi)
				},
			}
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
		}
		if resource.Operations.GetMany != nil {
			subcommand := getOrCreateSubcommand(&result, "select")
			operation := cli.Command{
				Name:  resource.SingularName,
				Usage: fmt.Sprintf("Save %s to session file", resource.SingularName),
				Action: func(c *cli.Context) error {
					return cliCmdSelect(c, resourceNameCopy, &jsopenapi)
				},
			}
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
		}
		if resource.Operations.GetMany != nil {
			subcommand := getOrCreateSubcommand(&result, "clear")
			operation := cli.Command{
				Name:  resource.SingularName,
				Usage: fmt.Sprintf("Clear %s from session file", resource.SingularName),
				Action: func(c *cli.Context) error {
					return cliCmdClear(c, resourceNameCopy, &jsopenapi)
				},
			}
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
		}
		if resource.Upload {
			subcommand := getOrCreateSubcommand(&result, "upload")
			operation := cli.Command{
				Name:  resource.SingularName,
				Usage: resource.Operations.CreateOne.Summary,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "input",
						Aliases:  []string{"i"},
						Required: true,
					},
					&cli.IntFlag{
						Name:    "interval",
						Aliases: []string{"t"},
						Value:   2,
					},
				},
				Action: func(c *cli.Context) error {
					return cliCmdUpload(c, resourceNameCopy, &jsopenapi)
				},
			}
			subcommand.Subcommands = append(subcommand.Subcommands, &operation)
		}

		for relationshipName, relationship := range resource.Relationships {
			relationshipNameCopy := relationshipName
			if relationship.Operations.Change != nil {
				subcommand := getOrCreateSubcommand(&result, "change")
				parent := getOrCreateSubcommand(subcommand, resource.SingularName)
				if !flagExists(parent.Flags, "id") {
					parent.Flags = append(parent.Flags, &cli.StringFlag{
						Name: "id",
						// If we want to `get something` and the `somethings`
						// resource does not support `get_many`, then the user
						// won't be able to fuzzy-select the something and
						// `--id` should be required
						Required: resource.Operations.GetMany == nil,
					})
				}
				addFilterTags(parent, resourceName, &jsopenapi, true)
				operation := cli.Command{
					Name:  relationshipName,
					Usage: relationship.Operations.Change.Summary,
					Action: func(c *cli.Context) error {
						return cliCmdChange(
							c, resourceNameCopy, relationshipNameCopy, &jsopenapi,
						)
					},
				}
				addFilterTags(&operation, relationship.Resource, &jsopenapi, true)
				parent.Subcommands = append(parent.Subcommands, &operation)
			}

			if relationship.Operations.Get != nil {
				subcommand := getOrCreateSubcommand(&result, "get")
				parent := getOrCreateSubcommand(subcommand, resource.SingularName)
				addFilterTags(parent, resourceName, &jsopenapi, true)
				operation := cli.Command{
					Name:  relationshipName,
					Usage: relationship.Operations.Get.Summary,
					Action: func(c *cli.Context) error {
						return cliCmdGetRelated(
							c, resourceNameCopy, relationshipNameCopy, &jsopenapi,
						)
					},
				}
				parent.Subcommands = append(parent.Subcommands, &operation)
			}

			if relationship.Operations.Add != nil {
				subcommand := getOrCreateSubcommand(&result, "add")
				parent := getOrCreateSubcommand(subcommand, resource.SingularName)
				addFilterTags(parent, resourceName, &jsopenapi, true)
				operation := cli.Command{
					Name:  relationshipName,
					Usage: relationship.Operations.Add.Summary,
					Action: func(c *cli.Context) error {
						return cliCmdAdd(
							c, resourceNameCopy, relationshipNameCopy, &jsopenapi,
						)
					},
				}
				relatedResource := jsopenapi.Resources[relationship.Resource]
				if relatedResource.Operations.GetMany == nil {
					operation.Flags = []cli.Flag{
						&cli.StringFlag{
							Name:     "ids",
							Usage:    "Comma-separated IDs to use for the relationship",
							Required: true,
						},
					}
				}
				parent.Subcommands = append(parent.Subcommands, &operation)
			}

			if relationship.Operations.Remove != nil {
				subcommand := getOrCreateSubcommand(&result, "remove")
				parent := getOrCreateSubcommand(subcommand, resource.SingularName)
				addFilterTags(parent, resourceName, &jsopenapi, true)
				operation := cli.Command{
					Name:  relationshipName,
					Usage: relationship.Operations.Remove.Summary,
					Action: func(c *cli.Context) error {
						return cliCmdRemove(
							c, resourceNameCopy, relationshipNameCopy, &jsopenapi,
						)
					},
				}
				relatedResource := jsopenapi.Resources[relationship.Resource]
				if relatedResource.Operations.GetMany == nil {
					operation.Flags = []cli.Flag{
						&cli.StringFlag{
							Name:     "ids",
							Usage:    "Comma-separated IDs to use for the relationship",
							Required: true,
						},
					}
				}
				parent.Subcommands = append(parent.Subcommands, &operation)
			}

			if relationship.Operations.Reset != nil {
				subcommand := getOrCreateSubcommand(&result, "reset")
				parent := getOrCreateSubcommand(subcommand, resource.SingularName)
				addFilterTags(parent, resourceName, &jsopenapi, true)
				operation := cli.Command{
					Name:  relationshipName,
					Usage: relationship.Operations.Reset.Summary,
					Action: func(c *cli.Context) error {
						return cliCmdReset(
							c, resourceNameCopy, relationshipNameCopy, &jsopenapi,
						)
					},
				}
				relatedResource := jsopenapi.Resources[relationship.Resource]
				if relatedResource.Operations.GetMany == nil {
					operation.Flags = []cli.Flag{
						&cli.StringFlag{
							Name:     "ids",
							Usage:    "Comma-separated IDs to use for the relationship",
							Required: true,
						},
					}
				}
				parent.Subcommands = append(parent.Subcommands, &operation)
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

func cliCmdGetOne(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	resourceId := c.String("id")
	if resourceId == "" {
		resourceId, err = getResourceId(c, api, resourceName, jsopenapi, true)
		if err != nil {
			return err
		}
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
	resourceId := c.String("id")
	if resourceId == "" {
		resourceId, err = getResourceId(c, api, resourceName, jsopenapi, true)
		if err != nil {
			return err
		}
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

func cliCmdChange(
	c *cli.Context,
	resourceName,
	relationshipName string,
	jsopenapi *jsopenapi_t,
) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	parentId := c.String("id")
	if parentId == "" {
		parentId, err = getResourceId(c, api, resourceName, jsopenapi, true)
		if err != nil {
			return err
		}
	}
	childIds, err := selectResourceIds(
		c,
		api,
		jsopenapi.Resources[resourceName].Relationships[relationshipName].Resource,
		jsopenapi,
		true,
		false,
	)
	if err != nil {
		return err
	}
	childId := childIds[0]

	parent, err := api.Get(resourceName, parentId)
	if err != nil {
		return err
	}
	parent.Relationships[relationshipName].DataSingular.Id = childId
	err = parent.Save([]string{relationshipName})
	if err != nil {
		return err
	}
	return nil
}

func cliCmdDelete(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
	resource := jsopenapi.Resources[resourceName]
	api, err := getApi(c)
	if err != nil {
		return err
	}
	resourceId := c.String("id")
	if resourceId == "" {
		resourceIds, err := selectResourceIds(
			c, api, resourceName, jsopenapi, true, false,
		)
		if err != nil {
			return err
		}
		resourceId = resourceIds[0]
	}
	fmt.Printf(
		"About to delete %s: %s, are you sure (y/N)? ",
		resource.SingularName,
		resourceId,
	)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(strings.ToLower(answer)) == "y" {
		obj := jsonapi.Resource{API: api, Type: resourceName, Id: resourceId}
		err = obj.Delete()
		if err != nil {
			return err
		}
		fmt.Printf("Deleted %s: %s\n", resource.SingularName, resourceId)
	} else {
		fmt.Printf("Deletion aborted\n")
	}
	return nil
}

func cliCmdGetRelated(
	c *cli.Context, resourceName, relationshipName string, jsopenapi *jsopenapi_t,
) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	parentId := c.String("id")
	if parentId == "" {
		parentId, err = getResourceId(c, api, resourceName, jsopenapi, true)
		if err != nil {
			return err
		}
	}
	parent, err := api.Get(resourceName, parentId)
	if err != nil {
		return err
	}
	url := parent.Relationships[relationshipName].Links.Related
	body, err := api.ListBodyFromPath(url)
	if err != nil {
		return err
	}
	err = invokePager(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdSelect(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
	resource := jsopenapi.Resources[resourceName]
	api, err := getApi(c)
	if err != nil {
		return err
	}
	resourceIds, err := selectResourceIds(c, api, resourceName, jsopenapi, true, false)
	if err != nil {
		return err
	}
	resourceId := resourceIds[0]
	err = save(resource.SingularName, resourceId)
	if err != nil {
		return err
	}
	fmt.Printf("Saved %s: %s\n", resource.SingularName, resourceId)
	return nil
}

func cliCmdClear(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
	resource := jsopenapi.Resources[resourceName]
	resourceId, err := load(resource.SingularName)
	if err != nil {
		return err
	}
	if resourceId == "" {
		fmt.Printf(
			"Key %s has no entry in .tx/api_explorer_session.json\n",
			resource.SingularName,
		)
		return nil
	}

	return clear(resource.SingularName)
}

func cliCmdAdd(
	c *cli.Context, resourceName, relationshipName string, jsopenapi *jsopenapi_t,
) error {
	resource := jsopenapi.Resources[resourceName]
	relatedResourceName := resource.Relationships[relationshipName].Resource

	api, err := getApi(c)
	if err != nil {
		return err
	}
	parentId := c.String("id")
	if parentId == "" {
		parentId, err = getResourceId(c, api, resourceName, jsopenapi, true)
		if err != nil {
			return err
		}
	}
	parent, err := api.Get(resourceName, parentId)
	if err != nil {
		return err
	}
	var childIds []string
	if jsopenapi.Resources[relatedResourceName].Operations.GetMany != nil {
		childIds, err = selectResourceIds(
			c, api, relatedResourceName, jsopenapi, true, true,
		)
		if err != nil {
			return err
		}
	} else {
		childIds = strings.Split(c.String("ids"), ",")
	}
	var children []*jsonapi.Resource
	for _, childId := range childIds {
		children = append(children, &jsonapi.Resource{
			Type: relatedResourceName,
			Id:   childId,
		})
	}
	err = parent.Add(relationshipName, children)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdRemove(
	c *cli.Context, resourceName, relationshipName string, jsopenapi *jsopenapi_t,
) error {
	resource := jsopenapi.Resources[resourceName]
	relatedResourceName := resource.Relationships[relationshipName].Resource

	api, err := getApi(c)
	if err != nil {
		return err
	}
	parentId := c.String("id")
	if parentId == "" {
		parentId, err = getResourceId(c, api, resourceName, jsopenapi, true)
		if err != nil {
			return err
		}
	}
	parent, err := api.Get(resourceName, parentId)
	if err != nil {
		return err
	}

	var childIds []string
	if jsopenapi.Resources[relatedResourceName].Operations.GetMany != nil {
		url := parent.Relationships[relationshipName].Links.Related
		body, err := api.ListBodyFromPath(url)
		if err != nil {
			return err
		}
		childIds, err = fuzzy(
			api,
			body,
			fmt.Sprintf("Select %s to remove", relationshipName),
			jsopenapi.Resources[relatedResourceName].Display,
			false,
			true,
		)
		if err != nil {
			return err
		}
	} else {
		childIds = strings.Split(c.String("ids"), ",")
	}

	var children []*jsonapi.Resource
	for _, childId := range childIds {
		children = append(children, &jsonapi.Resource{
			Type: relatedResourceName,
			Id:   childId,
		})
	}
	err = parent.Remove(relationshipName, children)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdReset(
	c *cli.Context, resourceName, relationshipName string, jsopenapi *jsopenapi_t,
) error {
	resource := jsopenapi.Resources[resourceName]
	relatedResourceName := resource.Relationships[relationshipName].Resource

	api, err := getApi(c)
	if err != nil {
		return err
	}
	parentId := c.String("id")
	if parentId == "" {
		parentId, err = getResourceId(c, api, resourceName, jsopenapi, true)
		if err != nil {
			return err
		}
	}
	parent, err := api.Get(resourceName, parentId)
	if err != nil {
		return err
	}
	var childIds []string
	if jsopenapi.Resources[relatedResourceName].Operations.GetMany != nil {
		childIds, err = selectResourceIds(
			c, api, relatedResourceName, jsopenapi, true, true,
		)
		if err != nil {
			return err
		}
	} else {
		childIds = strings.Split(c.String("ids"), ",")
	}
	var children []*jsonapi.Resource
	for _, childId := range childIds {
		children = append(children, &jsonapi.Resource{
			Type: relatedResourceName,
			Id:   childId,
		})
	}
	err = parent.Reset(relationshipName, children)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdCreateOne(
	c *cli.Context, resourceName string, jsopenapi *jsopenapi_t,
) error {
	resource := jsopenapi.Resources[resourceName]
	type resourceInfo struct {
		id           string
		resourceName string
	}

	requiredRelationships := make(map[string]*resourceInfo)
	optionalRelationships := make(map[string]*resourceInfo)

	api, err := getApi(c)
	if err != nil {
		return err
	}

	operation := jsopenapi.Resources[resourceName].Operations.CreateOne
	for relationhipName, resourceName := range operation.Relationships.Required {
		resourceIds, err := selectResourceIds(
			c, api, resourceName, jsopenapi, true, false,
		)
		if err != nil {
			return err
		}
		resourceId := resourceIds[0]
		requiredRelationships[relationhipName] = &resourceInfo{
			id:           resourceId,
			resourceName: resourceName,
		}
	}
	for relationshipName, resourceName := range operation.Relationships.Optional {
		resourceIds, err := selectResourceIds(
			c, api, resourceName, jsopenapi, false, false,
		)
		if err != nil {
			return err
		}
		resourceId := resourceIds[0]
		if resourceId != "" {
			optionalRelationships[relationshipName] = &resourceInfo{
				id:           resourceId,
				resourceName: resourceName,
			}
		}
	}

	attributes, err := create(
		c.String("editor"),
		operation.Attributes.Required,
		operation.Attributes.Optional,
	)
	if err != nil {
		return err
	}
	obj := jsonapi.Resource{
		API:        api,
		Type:       resourceName,
		Attributes: attributes,
	}
	for relationshipName, resourceInfo := range requiredRelationships {
		obj.SetRelated(relationshipName, &jsonapi.Resource{
			Type: resourceInfo.resourceName,
			Id:   requiredRelationships[relationshipName].id,
		})
	}
	for relationshipName, resourceInfo := range optionalRelationships {
		obj.SetRelated(
			relationshipName,
			&jsonapi.Resource{Type: resourceInfo.resourceName, Id: resourceInfo.id},
		)
	}
	err = obj.Save(nil)
	if err != nil {
		return err
	}
	fmt.Printf("Created %s: %s\n", resource.SingularName, obj.Id)
	return nil
}

func cliCmdUpload(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
	type resourceInfo struct {
		id           string
		resourceName string
	}

	requiredRelationships := make(map[string]*resourceInfo)
	optionalRelationships := make(map[string]*resourceInfo)

	api, err := getApi(c)
	if err != nil {
		return err
	}

	operation := jsopenapi.Resources[resourceName].Operations.CreateOne
	for relationhipName, resourceName := range operation.Relationships.Required {
		resourceIds, err := selectResourceIds(
			c, api, resourceName, jsopenapi, true, false,
		)
		if err != nil {
			return err
		}
		resourceId := resourceIds[0]
		requiredRelationships[relationhipName] = &resourceInfo{
			id:           resourceId,
			resourceName: resourceName,
		}
	}

	for relationshipName, resourceName := range operation.Relationships.Optional {
		resourceIds, err := selectResourceIds(
			c, api, resourceName, jsopenapi, false, false,
		)
		if err != nil {
			return err
		}
		resourceId := resourceIds[0]
		if resourceId != "" {
			optionalRelationships[relationshipName] = &resourceInfo{
				id:           resourceId,
				resourceName: resourceName,
			}
		}
	}

	filteredRequiredAttrs := make([]string, 0, len(operation.Attributes.Required)-2)
	for _, required := range operation.Attributes.Required {
		if required != "content" && required != "content_encoding" {
			filteredRequiredAttrs = append(filteredRequiredAttrs, required)
		}
	}

	attributes, err := create(
		c.String("editor"),
		filteredRequiredAttrs,
		operation.Attributes.Optional,
	)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(c.String("input"))
	if err != nil {
		return err
	}
	if utf8.Valid(body) {
		attributes["content"] = string(body)
		attributes["content_encoding"] = "text"
	} else {
		attributes["content"] = base64.StdEncoding.EncodeToString(body)
		attributes["content_encoding"] = "base64"
	}
	upload := jsonapi.Resource{
		API:        api,
		Type:       resourceName,
		Attributes: attributes,
	}
	for relationshipName, resourceInfo := range requiredRelationships {
		upload.SetRelated(relationshipName, &jsonapi.Resource{
			Type: resourceInfo.resourceName,
			Id:   requiredRelationships[relationshipName].id,
		})
	}
	for relationshipName, resourceInfo := range optionalRelationships {
		upload.SetRelated(
			relationshipName,
			&jsonapi.Resource{Type: resourceInfo.resourceName, Id: resourceInfo.id},
		)
	}
	err = upload.Save(nil)
	if err != nil {
		return err
	}
	var uploadAttributes struct {
		Status  string `json:"status"`
		Details struct {
			StringsCreated int `json:"strings_created"`
			StringsDeleted int `json:"strings_deleted"`
			StringsSkipped int `json:"strings_skipped"`
			StringsUpdated int `json:"strings_updated"`
		} `json:"details"`
	}
	for {
		err = upload.MapAttributes(&uploadAttributes)
		if err != nil {
			return err
		}
		if uploadAttributes.Status == "failed" {
			var errorsMessages []string
			for _, err := range upload.Attributes["errors"].([]map[string]string) {
				errorsMessages = append(errorsMessages, err["detail"])
			}
			return fmt.Errorf("upload failed: %s", strings.Join(errorsMessages, ", "))
		} else if uploadAttributes.Status == "succeeded" {
			break
		}
		time.Sleep(time.Duration(c.Int("interval")) * time.Second)
		err = upload.Reload()
		if err != nil {
			return err
		}
	}
	fmt.Printf(
		"Upload succeeded; created: %d, deleted: %d, skipped: %d, updated: %d "+
			"strings\n",
		uploadAttributes.Details.StringsCreated,
		uploadAttributes.Details.StringsDeleted,
		uploadAttributes.Details.StringsSkipped,
		uploadAttributes.Details.StringsUpdated,
	)
	return nil
}

func cliCmdDownload(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
	type resourceInfo struct {
		id           string
		resourceName string
	}

	requiredRelationships := make(map[string]*resourceInfo)
	optionalRelationships := make(map[string]*resourceInfo)

	api, err := getApi(c)
	if err != nil {
		return err
	}

	operation := jsopenapi.Resources[resourceName].Operations.CreateOne
	for relationhipName, resourceName := range operation.Relationships.Required {
		resourceIds, err := selectResourceIds(
			c, api, resourceName, jsopenapi, true, false,
		)
		if err != nil {
			return err
		}
		resourceId := resourceIds[0]
		requiredRelationships[relationhipName] = &resourceInfo{
			id:           resourceId,
			resourceName: resourceName,
		}
	}
	for relationshipName, resourceName := range operation.Relationships.Optional {
		resourceIds, err := selectResourceIds(
			c, api, resourceName, jsopenapi, false, false,
		)
		if err != nil {
			return err
		}
		resourceId := resourceIds[0]
		if resourceId != "" {
			optionalRelationships[relationshipName] = &resourceInfo{
				id:           resourceId,
				resourceName: resourceName,
			}
		}
	}

	attributes, err := create(
		c.String("editor"),
		operation.Attributes.Required,
		operation.Attributes.Optional,
	)
	if err != nil {
		return err
	}
	download := &jsonapi.Resource{
		API:        api,
		Type:       resourceName,
		Attributes: attributes,
	}
	for relationshipName, resourceInfo := range requiredRelationships {
		download.SetRelated(relationshipName, &jsonapi.Resource{
			Type: resourceInfo.resourceName,
			Id:   requiredRelationships[relationshipName].id,
		})
	}
	for relationshipName, resourceInfo := range optionalRelationships {
		download.SetRelated(
			relationshipName,
			&jsonapi.Resource{Type: resourceInfo.resourceName, Id: resourceInfo.id},
		)
	}
	err = download.Save(nil)
	if err != nil {
		return err
	}
	for {
		var downloadAttributes struct {
			Status string `json:"status"`
			Errors []struct {
				Code   string `json:"code"`
				Detail string `json:"detail"`
			} `json:"errors"`
		}
		err = download.MapAttributes(&downloadAttributes)
		if err != nil {
			return err
		}
		if download.Redirect != "" {
			response, err := http.Get(download.Redirect)
			if err != nil {
				return err
			}
			if response.StatusCode != 200 {
				return errors.New("file download error")
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				return err
			}
			defer response.Body.Close()
			if c.String("output") != "" {
				os.WriteFile(c.String("output"), body, 0644)
			} else {
				os.Stdout.Write(body)
			}
			return nil
		} else if downloadAttributes.Status == "failed" {
			var errorsMessages []string
			for _, err := range downloadAttributes.Errors {
				errorsMessages = append(errorsMessages, err.Detail)
			}
			return fmt.Errorf("download failed: %s", strings.Join(errorsMessages, ", "))
		}
		time.Sleep(time.Duration(c.Int("interval")) * time.Second)
		err = download.Reload()
		if err != nil {
			return err
		}
	}
}
