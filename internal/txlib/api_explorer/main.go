package api_explorer

import (
	"os"

	"github.com/urfave/cli/v2"
)

// TODOs:
//   - reset project languages requires fuzzyMulti with empty

var Cmd = &cli.Command{
	Name: "api",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "pager",
			Value: os.Getenv("PAGER"),
		},
		&cli.StringFlag{
			Name:  "editor",
			Value: os.Getenv("EDITOR"),
		},
	},
	Subcommands: []*cli.Command{
		{
			Name: "get",
			Subcommands: []*cli.Command{
				{Name: "organizations", Action: cliCmdGetOrganization},
				{Name: "organization", Action: cliCmdGetOrganizations},
				{Name: "projects", Action: cliCmdGetProjects},
				{
					Name:   "project",
					Action: cliCmdGetProject,
					Subcommands: []*cli.Command{
						{Name: "languages", Action: cliCmdGetProjectLanguages},
						{Name: "team", Action: cliCmdGetProjectTeam},
						{Name: "organization", Action: cliCmdGetProjectOrganization},
					},
				},
				{Name: "teams", Action: cliCmdGetTeams},
				{Name: "team", Action: cliCmdGetTeam},
				{Name: "languages", Action: cliCmdGetLanguages},
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
					return os.Remove(".tx/api_explorer_data.json")
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
			},
		},
		{
			Name: "delete",
			Subcommands: []*cli.Command{
				{Name: "project", Action: cliCmdDeleteProject},
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
					},
				},
			},
		},
	},
}
