package api_explorer

import (
	"os"

	"github.com/urfave/cli/v2"
)

// TODOs:
//   - Add more stuff
//     - Team managers
//   - Downloads/uploads
//   - Figure out how to generate most of the code from a configuration

var Cmd = &cli.Command{
	Name: "api",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "pager", Value: os.Getenv("PAGER")},
		&cli.StringFlag{Name: "editor", Value: os.Getenv("EDITOR")},
	},
	Subcommands: []*cli.Command{
		{
			Name: "get",
			Subcommands: []*cli.Command{
				{
					Name: "next",
					Action: func(c *cli.Context) error {
						api, err := getApi(c)
						if err != nil {
							return err
						}
						url, err := load("next")
						if err != nil {
							return err
						}
						body, err := api.ListBodyFromPath(url)
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
					},
				},
				{
					Name: "previous",
					Action: func(c *cli.Context) error {
						api, err := getApi(c)
						if err != nil {
							return err
						}
						url, err := load("previous")
						if err != nil {
							return err
						}
						body, err := api.ListBodyFromPath(url)
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
					},
				},
				{
					Name:   "organizations",
					Flags:  []cli.Flag{&cli.StringFlag{Name: "slug"}},
					Action: cliCmdGetOrganizations,
				},
				{Name: "organization", Action: cliCmdGetOrganization},
				{
					Name: "projects",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "name"},
						&cli.StringFlag{Name: "slug"},
					},
					Action: cliCmdGetProjects,
				},
				{
					Name:   "project",
					Action: cliCmdGetProject,
					Subcommands: []*cli.Command{
						{Name: "languages", Action: cliCmdGetProjectLanguages},
						{Name: "maintainers", Action: cliCmdGetProjectMaintainers},
						{Name: "team", Action: cliCmdGetProjectTeam},
						{Name: "organization", Action: cliCmdGetProjectOrganization},
					},
				},
				{
					Name: "teams",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "name"},
						&cli.StringFlag{Name: "slug"},
					},
					Action: cliCmdGetTeams,
				},
				{Name: "team", Action: cliCmdGetTeam},
				{
					Name: "languages",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "code"},
						&cli.StringFlag{Name: "code-any"},
					},
					Action: cliCmdGetLanguages,
				},
				{Name: "language", Action: cliCmdGetLanguage},
				{
					Name:   "i18n_formats",
					Flags:  []cli.Flag{&cli.StringFlag{Name: "name"}},
					Action: cliCmdGetI18nFormats,
				},
			},
		},
		{
			Name: "select",
			Subcommands: []*cli.Command{
				{Name: "organization", Action: cliCmdSelectOrganization},
				{Name: "project", Action: cliCmdSelectProject},
				{Name: "team", Action: cliCmdSelectTeam},
			},
		},
		{
			Name: "clear",
			Action: func(c *cli.Context) error {
				if c.Args().Present() {
					return clear(c.Args().First())
				} else {
					return os.Remove(".tx/api_explorer_session.json")
				}
			},
		},
		{
			Name: "edit",
			Subcommands: []*cli.Command{
				{Name: "project", Action: cliCmdEditProject},
				{Name: "team", Action: cliCmdEditTeam},
			},
		},
		{
			Name: "create",
			Subcommands: []*cli.Command{
				{Name: "project", Action: cliCmdCreateProject},
				{Name: "team", Action: cliCmdCreateTeam},
			},
		},
		{
			Name: "delete",
			Subcommands: []*cli.Command{
				{Name: "project", Action: cliCmdDeleteProject},
				{Name: "team", Action: cliCmdDeleteTeam},
			},
		},
		{
			Name: "change",
			Subcommands: []*cli.Command{
				{
					Name: "project",
					Subcommands: []*cli.Command{
						{Name: "team", Action: cliCmdChangeProjectTeam},
					},
				},
			},
		},
		{
			Name: "add",
			Subcommands: []*cli.Command{
				{
					Name: "project",
					Subcommands: []*cli.Command{
						{Name: "languages", Action: cliCmdAddProjectLanguages},
						{Name: "maintainers", Action: cliCmdAddProjectMaintainers},
					},
				},
			},
		},
		{
			Name: "remove",
			Subcommands: []*cli.Command{
				{
					Name: "project",
					Subcommands: []*cli.Command{
						{Name: "languages", Action: cliCmdRemoveProjectLanguages},
						{Name: "maintainers", Action: cliCmdRemoveProjectMaintainers},
					},
				},
			},
		},
		{
			Name: "reset",
			Subcommands: []*cli.Command{
				{
					Name: "project",
					Subcommands: []*cli.Command{
						{Name: "languages", Action: cliCmdResetProjectLanguages},
						{Name: "maintainers", Action: cliCmdResetProjectMaintainers},
					},
				},
			},
		},
	},
}
