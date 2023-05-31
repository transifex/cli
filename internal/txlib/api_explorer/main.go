package api_explorer

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

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
				Summary       string   `json:"summary"`
				Attributes    []string `json:"attributes"`
				Relationships []string `json:"relationships"`
			} `json:"edit_one"`
			DeleteOne *struct {
				Summary string `json:"summary"`
			} `json:"delete_one"`
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
			&cli.BoolFlag{Name: "no-interactive", Aliases: []string{"y"}},
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
							sessionPath, err := getSessionPath()
							if err != nil {
								return err
							}
							_, err = os.Stat(sessionPath)
							if err != nil {
								return err
							}
							body, err := os.ReadFile(sessionPath)
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
							sessionPath, err := getSessionPath()
							if err != nil {
								return err
							}
							err = os.Remove(sessionPath)
							if err != nil {
								return err
							}
							fmt.Printf("Removed %s successfully\n", sessionPath)
							return nil
						},
					},
				},
			},
			{
				Name: "has",
				Subcommands: []*cli.Command{
					{
						Name: "next",
						Action: func(c *cli.Context) error {
							value, err := load("next")
							if err != nil {
								return err
							}
							if value != "" {
								return nil
							} else {
								return cli.Exit("", 1)
							}
						},
					},
					{
						Name: "previous",
						Action: func(c *cli.Context) error {
							value, err := load("previous")
							if err != nil {
								return err
							}
							if value != "" {
								return nil
							} else {
								return cli.Exit("", 1)
							}
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
		// Make sure the closure functions have access to the correct variable
		resourceNameCopy := resourceName

		if resource.Upload {
			subcommand := getOrCreateSubcommand(&result, "upload")
			operation := &cli.Command{
				Name:  strings.TrimSuffix(resource.SingularName, "_async_upload"),
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
			addCreateFlags(operation, resourceName, &jsopenapi)
			subcommand.Subcommands = append(subcommand.Subcommands, operation)
		} else if resource.Download {
			subcommand := getOrCreateSubcommand(&result, "download")
			operation := &cli.Command{
				Name:  strings.TrimSuffix(resource.SingularName, "_async_download"),
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
			addCreateFlags(operation, resourceName, &jsopenapi)
			subcommand.Subcommands = append(subcommand.Subcommands, operation)
		} else {
			// Allow 'select' and 'clear' for all resource types; for those that
			// don't have the 'get_many' operation, set the 'id' flag as required.
			subcommand := getOrCreateSubcommand(&result, "select")
			operation := &cli.Command{
				Name:  resource.SingularName,
				Usage: fmt.Sprintf("Save %s to session file", resource.SingularName),
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: resource.Operations.GetMany == nil},
				},
				Action: func(c *cli.Context) error {
					return cliCmdSelect(c, resourceNameCopy, &jsopenapi)
				},
			}
			addFilterTags(operation, resourceName, &jsopenapi, true)
			subcommand.Subcommands = append(subcommand.Subcommands, operation)

			subcommand = getOrCreateSubcommand(&result, "clear")
			operation = &cli.Command{
				Name:  resource.SingularName,
				Usage: fmt.Sprintf("Clear %s from session file", resource.SingularName),
				Action: func(c *cli.Context) error {
					return cliCmdClear(c, resourceNameCopy, &jsopenapi)
				},
			}
			subcommand.Subcommands = append(subcommand.Subcommands, operation)

			if resource.Operations.GetMany != nil {
				subcommand := getOrCreateSubcommand(&result, "get")
				operation := &cli.Command{
					Name:  resource.PluralName,
					Usage: resource.Operations.GetMany.Summary,
					Action: func(c *cli.Context) error {
						return cliCmdGetMany(c, resourceNameCopy, &jsopenapi)
					},
				}
				addFilterTags(operation, resourceName, &jsopenapi, false)
				subcommand.Subcommands = append(subcommand.Subcommands, operation)
			}

			if resource.Operations.GetOne != nil {
				subcommand := getOrCreateSubcommand(&result, "get")
				operation := &cli.Command{
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
				addFilterTags(operation, resourceName, &jsopenapi, true)
				subcommand.Subcommands = append(subcommand.Subcommands, operation)
			}

			if resource.Operations.EditOne != nil {
				subcommand := getOrCreateSubcommand(&result, "edit")
				operation := &cli.Command{
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
				addFilterTags(operation, resourceName, &jsopenapi, true)
				for _, field := range resource.Operations.EditOne.Attributes {
					operation.Flags = append(
						operation.Flags, &cli.StringFlag{Name: fmt.Sprintf("set-%s", field)},
					)
				}
				subcommand.Subcommands = append(subcommand.Subcommands, operation)
			}

			if resource.Operations.CreateOne != nil {
				subcommand := getOrCreateSubcommand(&result, "create")
				operation := &cli.Command{
					Name:  resource.SingularName,
					Usage: resource.Operations.CreateOne.Summary,
					Action: func(c *cli.Context) error {
						return cliCmdCreateOne(c, resourceNameCopy, &jsopenapi)
					},
				}
				addCreateFlags(operation, resourceName, &jsopenapi)
				subcommand.Subcommands = append(subcommand.Subcommands, operation)
			}

			if resource.Operations.DeleteOne != nil {
				subcommand := getOrCreateSubcommand(&result, "delete")
				operation := &cli.Command{
					Name:  resource.SingularName,
					Usage: resource.Operations.DeleteOne.Summary,
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "id"},
					},
					Action: func(c *cli.Context) error {
						return cliCmdDelete(c, resourceNameCopy, &jsopenapi)
					},
				}
				addFilterTags(operation, resourceName, &jsopenapi, true)
				subcommand.Subcommands = append(subcommand.Subcommands, operation)
			}

			for relationshipName, relationship := range resource.Relationships {
				if relationship.Operations.Get != nil {
					addRelationshipCommand(
						&result, "get", resourceName, relationshipName, &jsopenapi,
					)
				}

				if relationship.Operations.Change != nil {
					addRelationshipCommand(
						&result, "change", resourceName, relationshipName, &jsopenapi,
					)
				}

				if relationship.Operations.Add != nil {
					addRelationshipCommand(
						&result, "add", resourceName, relationshipName, &jsopenapi,
					)
				}

				if relationship.Operations.Remove != nil {
					addRelationshipCommand(
						&result, "remove", resourceName, relationshipName, &jsopenapi,
					)
				}

				if relationship.Operations.Reset != nil {
					addRelationshipCommand(
						&result, "reset", resourceName, relationshipName, &jsopenapi,
					)
				}
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
	resource := jsopenapi.Resources[resourceName]
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
	obj, err := api.Get(resourceName, resourceId)
	if err != nil {
		return err
	}
	if c.Bool("no-interactive") {
		var changedFields []string
		for _, field := range resource.Operations.EditOne.Attributes {
			value := c.String(fmt.Sprintf("set-%s", field))
			if value != "" {
				obj.Attributes[field] = value
				changedFields = append(changedFields, field)
			}
		}
		if len(changedFields) == 0 {
			return errors.New("nothing changed")
		}
		err = obj.Save(changedFields)
	} else {
		err = edit(
			c.String("editor"),
			&obj,
			jsopenapi.Resources[resourceName].Operations.EditOne.Attributes,
		)
	}
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
	childId := c.String("related-id")
	if childId == "" {
		childIds, err := selectResourceIds(
			c,
			api,
			jsopenapi.Resources[resourceName].Relationships[relationshipName].Resource,
			relationshipName,
			jsopenapi,
			true,
			false,
		)
		if err != nil {
			return err
		}
		childId = childIds[0]
	}

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
			c, api, resourceName, "", jsopenapi, true, false,
		)
		if err != nil {
			return err
		}
		resourceId = resourceIds[0]
	}

	if !c.Bool("no-interactive") {
		fmt.Printf(
			"About to delete %s: %s, are you sure (y/N)? ",
			resource.SingularName,
			resourceId,
		)
		answer, err := input(
			fmt.Sprintf(
				"About to delete %s: %s, are you sure (y/N)? ",
				resource.SingularName,
				resourceId,
			),
		)
		if err != nil {
			return err
		}
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			return errors.New("deletion aborted")
		}
	}
	obj := jsonapi.Resource{API: api, Type: resourceName, Id: resourceId}
	err = obj.Delete()
	if err != nil {
		return err
	}
	fmt.Printf("Deleted %s: %s\n", resource.SingularName, resourceId)
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
	resourceId := c.String("id")
	if resourceId == "" {
		resourceIds, err := selectResourceIds(
			c, api, resourceName, "", jsopenapi, true, false,
		)
		if err != nil {
			return err
		}
		resourceId = resourceIds[0]
	}
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
	sessionPath, err := getSessionPath()
	if err != nil {
		return err
	}
	if resourceId == "" {
		fmt.Printf("Key %s has no entry in %s\n", resource.SingularName, sessionPath)
		return nil
	}

	return clear(resource.SingularName)
}

func cliCmdAdd(
	c *cli.Context, resourceName, relationshipName string, jsopenapi *jsopenapi_t,
) error {
	resource := jsopenapi.Resources[resourceName]
	relatedResourceName := resource.Relationships[relationshipName].Resource
	relatedResource := jsopenapi.Resources[relatedResourceName]

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
	if c.String("ids") != "" {
		childIds = strings.Split(c.String("ids"), ",")
	} else if relatedResource.Operations.GetMany == nil {
		return fmt.Errorf("cannot fetch %s to select", relatedResource.PluralName)
	} else if c.Bool("no-interactive") {
		return fmt.Errorf(
			"cannot select %s with --no-interactive, use the --ids flag",
			relatedResource.PluralName,
		)
	} else {
		childIds, err = selectResourceIds(
			c, api, relatedResourceName, relationshipName, jsopenapi, true, true,
		)
		if err != nil {
			return err
		}
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
	relatedResource := jsopenapi.Resources[relatedResourceName]

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
	relatedUrl := parent.Relationships[relationshipName].Links.Related
	if c.String("ids") != "" {
		childIds = strings.Split(c.String("ids"), ",")
	} else if c.Bool("no-interactive") {
		return fmt.Errorf(
			"cannot select %s with --no-interactive, use the --ids flag",
			relatedResource.PluralName,
		)
	} else if relatedUrl == "" {
		return fmt.Errorf(
			"cannot fetch %s to select", relatedResource.PluralName,
		)
	} else {
		body, err := api.ListBodyFromPath(relatedUrl)
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
	}

	if !c.Bool("no-interactive") {
		answer, err := input(
			fmt.Sprintf(
				"You are about to remove the %s %s from the %s %s, Are you sure (y/N)? ",
				strings.Join(childIds, ", "),
				relationshipName,
				parent.Id,
				resource.SingularName,
			),
		)
		if err != nil {
			return err
		}
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			return errors.New("removal aborted")
		}
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
	relatedResource := jsopenapi.Resources[relatedResourceName]

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
	if c.String("ids") != "" {
		childIds = strings.Split(c.String("ids"), ",")
	} else if relatedResource.Operations.GetMany == nil {
		return fmt.Errorf("cannot fetch %s to select", relatedResource.PluralName)
	} else if c.Bool("no-interactive") {
		return fmt.Errorf(
			"cannot select %s with --no-interactive, use the --ids flag",
			relatedResource.PluralName,
		)
	} else {
		childIds, err = selectResourceIds(
			c, api, relatedResourceName, relationshipName, jsopenapi, true, true,
		)
		if err != nil {
			return err
		}
	}

	if !c.Bool("no-interactive") {
		answer, err := input(
			fmt.Sprintf(
				"You are about to replace the %s %s's %s with %s, Are you sure (y/N)? ",
				parent.Id,
				resource.SingularName,
				relationshipName,
				strings.Join(childIds, ", "),
			),
		)
		if err != nil {
			return err
		}
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			return errors.New("removal aborted")
		}
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
	obj, err := createObject(c, resourceName, jsopenapi, false)
	if err != nil {
		return err
	}
	fmt.Printf("Created %s: %s\n", resource.SingularName, obj.Id)
	return nil
}

func cliCmdUpload(c *cli.Context, resourceName string, jsopenapi *jsopenapi_t) error {
	upload, err := createObject(c, resourceName, jsopenapi, true)
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
	download, err := createObject(c, resourceName, jsopenapi, false)
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
