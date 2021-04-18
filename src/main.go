package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Load configuration from `FILE`",
		},
		&cli.StringFlag{
			Name:    "token",
			Aliases: []string{"t"},
			Usage:   "The api token to use",
			EnvVars: []string{"TX_TOKEN"},
		},
		&cli.StringFlag{
			Name:    "hostname",
			Aliases: []string{"H"},
			Usage:   "The API hostname",
			EnvVars: []string{"TX_HOSTNAME"},
		},
	}
	app := &cli.App{
		Before: func(c *cli.Context) error {
			err := setMetadata(c)
			if err != nil {
				return err
			}
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:    "migrate",
				Aliases: []string{"mg"},
				Usage:   "Migrate legacy configuration.",
				Action:  migrateLegacyConfigFile,
			},
			{
				Name:    "showconf",
				Aliases: []string{"sc"},
				Usage:   "Print the active configuration",
				Action: func(c *cli.Context) error {
					fmt.Printf("Root config file: %s\n", c.App.Metadata["RootConfigFilePath"])
					fmt.Printf("Config file : %s\n", c.App.Metadata["ConfigFilePath"])
					fmt.Printf("Project dir: %s\n", c.App.Metadata["ProjectDir"])
					configJSON, _ := JSONMarshal(c.App.Metadata["Config"])
					fmt.Printf("Config:\n%s\n\n", string(configJSON))

					fileMappingsJSON, _ := JSONMarshal(c.App.Metadata["FileMappings"])
					fmt.Printf("FileMappings:\n%s\n", string(fileMappingsJSON))
					return nil
				},
			},
			{
				Name:    "pull",
				Aliases: []string{"p"},
				Usage:   "Pull translation files",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "force",
						Value: false,
						Usage: "Whether to skip timestamp checks",
					},
					&cli.BoolFlag{
						Name:  "all",
						Value: false,
						Usage: "Whether non existing files as well",
					},
					&cli.BoolFlag{
						Name:  "disable-overwrite",
						Value: false,
						Usage: "Whether skip existing files",
					},
					&cli.BoolFlag{
						Name:  "use-git-timestamps",
						Value: false,
						Usage: "Whether to use git commit timestamps instead of file timestamps",
					},
					&cli.BoolFlag{
						Name:  "skip",
						Value: false,
						Usage: "Whether to continue with downloading and write files to disk even if a download has failed",
					},
					&cli.BoolFlag{
						Name:  "xliff",
						Value: false,
						Usage: "",
					},
					&cli.IntFlag{
						Name:  "parallel",
						Value: 1,
						Usage: "Whether to use git commit timestamps instead of file timestamps",
					},
				},
				Action: pullCommand,
			},
			{
				Name:    "upload",
				Aliases: []string{"g"},
				Usage:   "Upload files",
				Subcommands: []*cli.Command{
					{
						Name:  "source",
						Usage: "Upload source file of resource",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "rid",
								Aliases:  []string{"r"},
								Required: true,
								Usage:    "The id of resource to upload source file",
							},
							&cli.StringFlag{
								Name:     "path",
								Required: true,
								Usage:    "The path of the source file",
							},
						},
						Action: func(c *cli.Context) error {
							config := c.App.Metadata["Config"].(*Config)
							client := NewClient(config.Token, config.Hostname)

							resourceID := c.String("rid")
							path := c.String("path")
							resp, err := client.createResourceStringsUpload(resourceID, path)
							if err != nil {
								return err
							}

							ctx, cancelFunc := context.WithTimeout(context.Background(), 60*time.Second)
							defer cancelFunc()
							result, err := client.pollResourceStringsUpload(
								ctx, resp.ID, 1*time.Second,
							)

							if result.Attributes.Details != nil {
								if err = PrintResponse(result.Attributes.Details); err != nil {
									return err
								}
							}
							return nil
						},
					},
				},
			},
			{
				Name:    "get",
				Aliases: []string{"g"},
				Usage:   "Retrieve an API resource",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "output",
						Aliases:  []string{"o"},
						Required: false,
						Value:    "json",
						Usage:    "How the output will be presented",
					},
				},
				Subcommands: []*cli.Command{
					{
						Name:    "i18n_formats",
						Usage:   "Retrieve i18n_formats",
						Aliases: []string{"formats", "f"},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "oid",
								Aliases:  []string{"o"},
								Required: true,
								Usage:    "The id of the organization to retrieve i18n_formats from",
							},
						},
						Action: func(c *cli.Context) error {
							config := c.App.Metadata["Config"].(*Config)
							client := NewClient(config.Token, config.Hostname)

							orgID := c.String("oid")
							resp, err := client.getI18NFormats(orgID)
							if err != nil {
								return err
							}

							if err = PrintResponse(resp); err != nil {
								return err
							}
							return nil
						},
					},
					{
						Name:  "stats",
						Usage: "retrieve rlstats",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "pid",
								Aliases:  []string{"p"},
								Required: true,
								Usage:    "The id of project",
							},
							&cli.StringFlag{
								Name:    "rid",
								Aliases: []string{"r"},
								Usage:   "The id of resource",
							},
							&cli.StringFlag{
								Name:    "lid",
								Aliases: []string{"l"},
								Usage:   "The id of language",
							},
						},
						Action: func(c *cli.Context) error {
							config := c.App.Metadata["Config"].(*Config)
							client := NewClient(config.Token, config.Hostname)

							projectID := c.String("pid")

							var resourceID *string = nil
							if c.IsSet("rid") {
								rid := c.String("rid")
								resourceID = &rid
							}

							var languageID *string = nil
							if c.IsSet("lid") {
								lid := c.String("lid")
								languageID = &lid
							}

							resp, err := client.getProjectLanguageStats(projectID, resourceID, languageID)
							if err != nil {
								return err
							}

							if err = PrintResponse(resp); err != nil {
								return err
							}
							return nil
						},
					},
					{
						Name:    "organizations",
						Usage:   "Retrieve organizations",
						Aliases: []string{"org", "o"},
						Action: func(c *cli.Context) error {
							config := c.App.Metadata["Config"].(*Config)
							client := NewClient(config.Token, config.Hostname)

							orgID := c.Args().Get(0)

							var resp interface{}
							var err error
							if strings.Compare(orgID, "") != 0 {
								resp, err = client.getOrganization(orgID)
							} else {
								resp, err = client.getOrganizations()
							}
							if err != nil {
								return err
							}

							if err = PrintResponse(resp); err != nil {
								return err
							}
							return nil
						},
					},
					{
						Name:    "projects",
						Usage:   "Retrieve projects",
						Aliases: []string{"pro", "p"},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "oid",
								Aliases: []string{"o"},
								Usage:   "The id of the organization the project belongs to",
							},
						},
						Action: func(c *cli.Context) error {
							config := c.App.Metadata["Config"].(*Config)
							client := NewClient(config.Token, config.Hostname)

							projectID := c.Args().Get(0)

							var resp interface{}
							var err error
							if strings.Compare(projectID, "") == 0 && !c.IsSet("oid") {
								return fmt.Errorf("Flag \"oid\" is required when no project ID is provided")
							}

							if strings.Compare(projectID, "") != 0 {
								resp, err = client.getProject(projectID)
							} else {
								resp, err = client.getProjects(c.String("oid"))
							}
							if err != nil {
								return err
							}

							if err = PrintResponse(resp); err != nil {
								return err
							}
							return nil
						},
					},
					{
						Name:    "resources",
						Usage:   "Retrieve projects",
						Aliases: []string{"res", "r"},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "pid",
								Aliases: []string{"p"},
								Usage:   "The id of the project the resource belongs to",
							},
						},
						Action: func(c *cli.Context) error {
							config := c.App.Metadata["Config"].(*Config)
							client := NewClient(config.Token, config.Hostname)

							resourceID := c.Args().Get(0)

							var resp interface{}
							var err error
							if strings.Compare(resourceID, "") == 0 && !c.IsSet("pid") {
								return fmt.Errorf("Flag \"pid\" is required when no project ID is provided")
							}

							if strings.Compare(resourceID, "") != 0 {
								resp, err = client.getResource(resourceID)
							} else {
								resp, err = client.getResources(c.String("pid"))
							}
							if err != nil {
								return err
							}

							if err = PrintResponse(resp); err != nil {
								return err
							}
							return nil
						},
					},
				},
			},
			{
				Name:    "create",
				Aliases: []string{"c"},
				Usage:   "create and API resource",
				Subcommands: []*cli.Command{
					{
						Name:    "resources",
						Usage:   "Create a resource",
						Aliases: []string{"r", "res", "resource"},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "pid",
								Aliases:  []string{"p"},
								Required: true,
								Usage:    "The id of the project the resource belongs to",
							},
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Required: true,
								Usage:    "The name of the resource",
							},
							&cli.StringFlag{
								Name:     "slug",
								Aliases:  []string{"s"},
								Required: true,
								Usage:    "The slug of the resource",
							},
							&cli.StringFlag{
								Name:     "type",
								Aliases:  []string{"t"},
								Required: true,
								Usage:    "The type of the resource",
							},
						},
						Action: func(c *cli.Context) error {
							config := c.App.Metadata["Config"].(*Config)
							client := NewClient(config.Token, config.Hostname)

							resp, err := client.createResource(
								c.String("pid"),
								c.String("name"),
								c.String("slug"),
								c.String("type"),
							)
							if err != nil {
								return err
							}

							output, err := JSONMarshal(resp)
							if err != nil {
								return err
							}
							fmt.Printf("%s\n", string(output))
							return nil
						},
					},
				},
			},
		},
		Flags: flags,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
