package api_explorer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/manifoldco/promptui"
	"github.com/transifex/cli/internal/txlib"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

func getApi(c *cli.Context) (*jsonapi.Connection, error) {
	cfg, err := config.LoadFromPaths(
		c.String("root-config"),
		c.String("config"),
	)
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %s", err)
	}
	hostname, token, err := txlib.GetHostAndToken(
		&cfg,
		c.String("hostname"),
		c.String("token"),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting API token: %s", err)
	}

	client, err := txlib.GetClient(c.String("cacert"))
	if err != nil {
		return nil, fmt.Errorf(
			"error getting HTTP client configuration: %s",
			err,
		)
	}

	return &jsonapi.Connection{
		Host:   hostname,
		Token:  token,
		Client: client,
		Headers: map[string]string{
			"Integration": "txclient",
		},
	}, nil
}

func save(key, value string) error {
	if _, err := os.Stat(".tx"); os.IsNotExist(err) {
		err := os.Mkdir(".tx", 0755)
		if err != nil {
			return err
		}
	}
	var body []byte
	if _, err := os.Stat(".tx/api_explorer_data.json"); err == nil {
		body, err = os.ReadFile(".tx/api_explorer_data.json")
		if err != nil {
			return err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		body = []byte("{}")

	} else {
		return err
	}
	var data map[string]string
	err := json.Unmarshal(body, &data)
	if err != nil {
		return err
	}
	data[key] = value
	body, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(".tx/api_explorer_data.json", body, 0644)
	if err != nil {
		return err
	}
	return nil
}

func load(key string) (string, error) {
	_, err := os.Stat(".tx/api_explorer_data.json")
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	body, err := os.ReadFile(".tx/api_explorer_data.json")
	if err != nil {
		return "", err
	}
	var data map[string]string
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}
	value, exists := data[key]
	if !exists {
		return "", nil
	}
	return value, nil
}

func clear(key string) error {
	_, err := os.Stat(".tx/api_explorer_data.json")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	body, err := os.ReadFile(".tx/api_explorer_data.json")
	if err != nil {
		return err
	}
	var data map[string]string
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}
	delete(data, key)
	body, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(".tx/api_explorer_data.json", body, 0644)
	if err != nil {
		return err
	}
	return nil
}

func page(pager string, body []byte) error {
	var unmarshalled map[string]interface{}
	err := json.Unmarshal(body, &unmarshalled)
	if err != nil {
		return err
	}
	output, err := json.MarshalIndent(unmarshalled, "", "  ")
	if err != nil {
		return err
	}
	cmd := exec.Command(pager)
	cmd.Stdin = bytes.NewBuffer(output)
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func selectOrganization(api *jsonapi.Connection) (string, error) {
	organizations, err := api.List("organizations", "")
	if err != nil {
		return "", err
	}
	var items []string
	for _, organization := range organizations.Data {
		items = append(items, organization.Id)
	}
	prompt := promptui.Select{
		Label: "Select organization",
		Items: items,
	}
	_, organizationId, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return organizationId, nil
}

func selectProject(api *jsonapi.Connection, organizationId string) (string, error) {
	query := jsonapi.Query{Filters: map[string]string{"organization": organizationId}}
	projects, err := api.List("projects", query.Encode())
	if err != nil {
		return "", err
	}
	var items []string
	for _, project := range projects.Data {
		items = append(items, project.Id)
	}
	prompt := promptui.Select{
		Label: "Select project",
		Items: items,
	}
	_, projectId, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return projectId, nil
}

var Cmd = &cli.Command{
	Name: "api",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "pager",
			Value: os.Getenv("PAGER"),
		},
		&cli.BoolFlag{
			Name:  "save",
			Value: false,
		},
	},
	Subcommands: []*cli.Command{
		{
			Name: "get",
			Subcommands: []*cli.Command{
				{
					Name: "organizations",
					Action: func(c *cli.Context) error {
						api, err := getApi(c)
						if err != nil {
							return err
						}
						body, err := api.ListBody("organizations", "")
						if err != nil {
							return err
						}
						err = page(c.String("pager"), body)
						if err != nil {
							return err
						}
						return nil
					},
				},
				{
					Name: "organization",
					Action: func(c *cli.Context) error {
						api, err := getApi(c)
						if err != nil {
							return err
						}
						organizationId, err := load("organization")
						if err != nil {
							return err
						}
						if organizationId == "" {
							organizationId, err = selectOrganization(api)
							if err != nil {
								return err
							}
							if c.Bool("save") {
								err = save("organization", organizationId)
								if err != nil {
									return err
								}
							}
						}
						body, err := api.GetBody("organizations", organizationId)
						if err != nil {
							return err
						}
						err = page(c.String("pager"), body)
						if err != nil {
							return err
						}
						return nil
					},
				},
				{
					Name: "projects",
					Action: func(c *cli.Context) error {
						api, err := getApi(c)
						if err != nil {
							return err
						}
						organizationId, err := load("organization")
						if err != nil {
							return err
						}
						if organizationId == "" {
							organizationId, err = selectOrganization(api)
							if err != nil {
								return err
							}
							if c.Bool("save") {
								err = save("organization", organizationId)
								if err != nil {
									return err
								}
							}
						}
						query := jsonapi.Query{
							Filters: map[string]string{"organization": organizationId},
						}
						body, err := api.ListBody("projects", query.Encode())
						if err != nil {
							return err
						}
						err = page(c.String("pager"), body)
						if err != nil {
							return err
						}
						return nil
					},
				},
				{
					Name: "project",
					Action: func(c *cli.Context) error {
						api, err := getApi(c)
						if err != nil {
							return err
						}
						projectId, err := load("project")
						if err != nil {
							return err
						}
						if projectId == "" {
							organizationId, err := load("organization")
							if err != nil {
								return err
							}
							if organizationId == "" {
								organizationId, err = selectOrganization(api)
								if err != nil {
									return err
								}
								if c.Bool("save") {
									err = save("organization", organizationId)
									if err != nil {
										return err
									}
								}
							}
							projectId, err = selectProject(api, organizationId)
							if err != nil {
								return err
							}
							if c.Bool("save") {
								err = save("project", projectId)
								if err != nil {
									return err
								}
							}
						}
						body, err := api.GetBody("projects", projectId)
						if err != nil {
							return err
						}
						err = page(c.String("pager"), body)
						if err != nil {
							return err
						}
						return nil
					},
				},
			},
		},
		{
			Name: "select",
			Subcommands: []*cli.Command{
				{
					Name: "organization",
					Action: func(c *cli.Context) error {
						api, err := getApi(c)
						if err != nil {
							return err
						}
						organizationId, err := selectOrganization(api)
						if err != nil {
							return err
						}
						err = save("organization", organizationId)
						if err != nil {
							return err
						}
						fmt.Printf("Saved organization: %s\n", organizationId)
						return nil
					},
				},
				{
					Name: "project",
					Action: func(c *cli.Context) error {
						api, err := getApi(c)
						if err != nil {
							return err
						}
						organizationId, err := load("organization")
						if err != nil {
							return err
						}
						if organizationId == "" {
							organizationId, err = selectOrganization(api)
							if err != nil {
								return err
							}
							if c.Bool("save") {
								err = save("organization", organizationId)
								if err != nil {
									return err
								}
							}
						}
						projectId, err := selectProject(api, organizationId)
						if err != nil {
							return err
						}
						err = save("project", projectId)
						if err != nil {
							return err
						}
						fmt.Printf("Saved project: %s\n", projectId)
						return nil
					},
				},
			},
		},
		{
			Name: "clear",
			Subcommands: []*cli.Command{
				{
					Name: "organization",
					Action: func(c *cli.Context) error {
						return clear("organization")
					},
				},
				{
					Name: "project",
					Action: func(c *cli.Context) error {
						return clear("project")
					},
				},
			},
		},
	},
}
