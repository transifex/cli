package api_explorer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

const CREATE_TEAM_STRING = `{
  "// Required fields": "",

  "name": "The team's name",

  "// Optional fields (remember to remove the leading '//' from the keys)": "",

  "//auto_join": false,
  "//cla": "",
  "//cla_required": false
}`

func selectTeamId(
	api *jsonapi.Connection, organizationId string, allowEmpty bool, header string,
) (string, error) {
	if header == "" {
		header = "Select team"
	}

	query := jsonapi.Query{Filters: map[string]string{"organization": organizationId}}

	body, err := api.ListBody("teams", query.Encode())
	if err != nil {
		return "", err
	}

	teamId, err := fuzzy(
		api,
		body,
		header,
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
		teamId, err = selectTeamId(api, organizationId, allowEmpty, "")
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
	if c.String("name") != "" {
		query.Filters["name"] = c.String("name")
	}
	if c.String("slug") != "" {
		query.Filters["slug"] = c.String("slug")
	}
	body, err := api.ListBody("teams", query.Encode())
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
	organizationId, err := getOrganizationId(api)
	if err != nil {
		return err
	}
	err = save("organization", organizationId)
	if err != nil {
		return err
	}
	teamId, err := getTeamId(api, organizationId, false)
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
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		_, err = api.Request("PATCH", fmt.Sprintf("/teams/%s", teamId), body, "")
		if err != nil {
			return err
		}
		return nil
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

func cliCmdDeleteTeam(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	organizationId, err := getOrganizationId(api)
	if err != nil {
		return err
	}
	teamId, err := selectTeamId(api, organizationId, false, "Select team to delete")
	if err != nil {
		return err
	}
	fmt.Printf("About to delete team: %s, are you sure (y/N)? ", teamId)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(strings.ToLower(answer)) == "y" {
		team := jsonapi.Resource{API: api, Type: "teams", Id: teamId}
		err = team.Delete()
		if err != nil {
			return err
		}
		fmt.Printf("Deleted team: %s\n", teamId)
	}
	return nil
}

func cliCmdCreateTeam(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		_, err = api.Request("POST", "/teams", body, "")
		if err != nil {
			return err
		}
		return nil
	}
	organizationId, err := getOrganizationId(api)
	if err != nil {
		return err
	}
	attributes, err := create(
		CREATE_TEAM_STRING,
		c.String("editor"),
		[]string{"name", "auto_join", "cla", "cla_required"},
	)
	if err != nil {
		return err
	}
	team := jsonapi.Resource{
		API:        api,
		Type:       "teams",
		Attributes: attributes,
	}
	team.SetRelated("organization", &jsonapi.Resource{
		Type: "organizations",
		Id:   organizationId,
	})
	err = team.Save(nil)
	if err != nil {
		return err
	}
	fmt.Printf("Created team: %s\n", team.Id)
	return nil
}

func cliCmdGetTeamManagers(c *cli.Context) error {
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
	url := team.Relationships["managers"].Links.Related
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
}

func cliCmdAddTeamManagers(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	teamId, err := getTeamId(api, "", false)
	if err != nil {
		return err
	}
	fmt.Printf(
		"Write usernames of managers to be added to the team " +
			"(separated by comma):\n> ",
	)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	usernames := strings.Split(answer, ",")
	var payload []*jsonapi.Resource
	for _, username := range usernames {
		username = strings.TrimSpace(username)
		payload = append(payload, &jsonapi.Resource{
			Type: "users",
			Id:   fmt.Sprintf("u:%s", username),
		})
	}
	team := &jsonapi.Resource{
		API:  api,
		Type: "teams",
		Id:   teamId,
		Relationships: map[string]*jsonapi.Relationship{
			"managers": {Type: jsonapi.PLURAL},
		},
	}
	err = team.Add("managers", payload)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdResetTeamManagers(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	teamId, err := getTeamId(api, "", false)
	if err != nil {
		return err
	}
	fmt.Printf(
		"Write usernames of users that will replace the managers of the team " +
			"(separated by comma):\n> ",
	)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	usernames := strings.Split(answer, ",")
	var payload []*jsonapi.Resource
	for _, username := range usernames {
		username = strings.TrimSpace(username)
		if username == "" {
			continue
		}
		payload = append(payload, &jsonapi.Resource{
			Type: "users",
			Id:   fmt.Sprintf("u:%s", username),
		})
	}
	team := &jsonapi.Resource{
		API:  api,
		Type: "teams",
		Id:   teamId,
		Relationships: map[string]*jsonapi.Relationship{
			"managers": {Type: jsonapi.PLURAL},
		},
	}
	err = team.Reset("managers", payload)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdRemoveTeamManagers(c *cli.Context) error {
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
	body, err := api.ListBodyFromPath(
		fmt.Sprintf("/teams/%s/managers", teamId),
	)
	if err != nil {
		return err
	}
	userIds, err := fuzzyMulti(
		api,
		body,
		"Select managers to remove (TAB for multiple selection)",
		func(user *jsonapi.Resource) string {
			return user.Attributes["username"].(string)
		},
		false,
	)
	if err != nil {
		return err
	}

	var payload []*jsonapi.Resource
	for _, userId := range userIds {
		payload = append(payload, &jsonapi.Resource{
			Type: "users",
			Id:   userId,
		})
	}
	err = team.Remove("managers", payload)
	if err != nil {
		return err
	}
	return nil
}
