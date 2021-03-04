package main

import (
	"fmt"
	"log"
	"os"

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
	}
	app := &cli.App{
		Before: func(c *cli.Context) error {
			var err error
			err = setMetadata(c)
			if err != nil {
				return err
			}
			err = formatConfigFile(c)
			if err != nil {
				return err
			}
			return nil
		},
		Commands: []*cli.Command{
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

					// I wanted to see how to transform an interface to a map
					fileMappings := c.App.Metadata["FileMappings"].(map[string]FileMapping)
					// Now for example you could delete a key since it is a map
					// delete(fileMappings, "DEFAULT")
					fileMappingsJSON, _ := JSONMarshal(fileMappings)
					fmt.Printf("FileMappings:\n%s\n", string(fileMappingsJSON))
					return nil
				},
			},
			{
				Name:    "getorgs",
				Aliases: []string{"go"},
				Usage:   "Yolo",
				Action: func(c *cli.Context) error {
					config := c.App.Metadata["Config"].(*Config)
					organizations := getOrganizations(config.Token, config.RestHostname)
					for _, organization := range organizations {
						projects := getProjects(organization.ID, config.Token, config.RestHostname)
						for _, project := range projects {
							getResources(project.ID, config.Token, config.RestHostname)
						}
					}
					return nil
				},
			},
			{
				Name:    "template",
				Aliases: []string{"t"},
				Usage:   "options for task templates",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "add a new template",
						Action: func(c *cli.Context) error {
							fmt.Println("new task template: ", c.Args().First())
							return nil
						},
					},
					{
						Name:  "remove",
						Usage: "remove an existing template",
						Action: func(c *cli.Context) error {
							fmt.Println("removed task template: ", c.Args().First())
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
