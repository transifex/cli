package api_explorer

import (
	"fmt"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

func selectTeamId(
	api *jsonapi.Connection, organizationId string, allowEmpty bool,
) (string, error) {
	query := jsonapi.Query{Filters: map[string]string{"organization": organizationId}}

	body, err := api.ListBody("teams", query.Encode())
	if err != nil {
		return "", err
	}

	teamId, err := fuzzy(
		api,
		body,
		"Select team",
		func(team *jsonapi.Resource) string {
			return fmt.Sprintf(
				"%s (%s)",
				team.Attributes["name"],
				team.Attributes["slug"],
			)
		},
		allowEmpty,
	)
	if err != nil {
		return "", err
	}
	return teamId, nil
}

func getTeamId(
	api *jsonapi.Connection, organizationId string, allowEmpty bool,
) (string, error) {
	teamId, err := load("team")
	if err != nil {
		return "", err
	}
	if teamId == "" {
		if organizationId == "" {
			organizationId, err = getOrganizationId(api)
			if err != nil {
				return "", err
			}
		}
		teamId, err = selectTeamId(api, organizationId, allowEmpty)
		if err != nil {
			return "", err
		}
	}
	return teamId, nil
}

func cliCmdGetTeams(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	organizationId, err := getOrganizationId(api)
	if err != nil {
		return err
	}
	query := jsonapi.Query{
		Filters: map[string]string{"organization": organizationId},
	}
	body, err := api.ListBody("teams", query.Encode())
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdGetTeam(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	teamId, err := getTeamId(api, "", false)
	if err != nil {
		return err
	}
	body, err := api.GetBody("teams", teamId)
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdSelectTeam(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	teamId, err := getTeamId(api, "", false)
	if err != nil {
		return err
	}
	err = save("team", teamId)
	if err != nil {
		return err
	}
	fmt.Printf("Saved team: %s\n", teamId)
	return nil
}

func cliCmdEditTeam(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	teamId, err := getTeamId(api, "", false)
	if err != nil {
		return err
	}
	team, err := api.Get("teams", teamId)
	if err != nil {
		return err
	}
	err = edit(
		c.String("editor"),
		&team,
		[]string{"auto_join", "cla", "cla_required", "name"},
	)
	if err != nil {
		return err
	}
	return nil
}
