package api_explorer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/urfave/cli/v2"
)

var createProjectString = `{
  "// Required fields": "",

  "name": "The name of the project",
  "slug": "the_slug_of_the_project",
  "private": true,

  "// Optional fields (remember to remove the leading '//' from the keys)": "",

  "//description": "",
  "//homepage_url": "",
  "//instructions_url": "",
  "//license": "",
  "//long_description": "",
  "//machine_tranlation_fillup": false,
  "//repository_url": "",
  "//tags": [],
  "//translation_memory_fillup": false,
  "//type": "file/live"
}`

func selectProjectId(
	api *jsonapi.Connection,
	organizationId string,
	header string,
) (string, error) {
	if header == "" {
		header = "Select project:"
	}
	query := jsonapi.Query{Filters: map[string]string{"organization": organizationId}}
	body, err := api.ListBody("projects", query.Encode())
	if err != nil {
		return "", err
	}
	projectId, err := fuzzy(
		api,
		body,
		header,
		func(project *jsonapi.Resource) string {
			var attributes txapi.ProjectAttributes
			err := project.MapAttributes(&attributes)
			if err != nil {
				return project.Id
			}
			return fmt.Sprintf("%s (%s)", attributes.Name, attributes.Slug)
		},
		false,
	)
	if err != nil {
		return "", err
	}
	return projectId, nil
}

func getProjectId(
	api *jsonapi.Connection,
	organizationId string,
) (string, error) {
	projectId, err := load("project")
	if err != nil {
		return "", err
	}
	if projectId == "" {
		if organizationId == "" {
			organizationId, err = getOrganizationId(api)
			if err != nil {
				return "", err
			}
		}
		projectId, err = selectProjectId(api, organizationId, "")
		if err != nil {
			return "", err
		}
	}
	return projectId, nil
}

func cliCmdGetProjects(c *cli.Context) error {
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
	body, err := api.ListBody("projects", query.Encode())
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdGetProject(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectId, err := getProjectId(api, "")
	if err != nil {
		return err
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
}

func cliCmdGetProjectLanguages(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectId, err := getProjectId(api, "")
	if err != nil {
		return err
	}
	project, err := api.Get("projects", projectId)
	if err != nil {
		return err
	}
	url := project.Relationships["languages"].Links.Related
	body, err := api.ListBodyFromPath(url)
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdGetProjectTeam(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectId, err := getProjectId(api, "")
	if err != nil {
		return err
	}
	project, err := api.Get("projects", projectId)
	if err != nil {
		return err
	}
	url := project.Relationships["team"].Links.Related
	body, err := api.ListBodyFromPath(url)
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdGetProjectOrganization(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectId, err := getProjectId(api, "")
	if err != nil {
		return err
	}
	project, err := api.Get("projects", projectId)
	if err != nil {
		return err
	}
	url := project.Relationships["organization"].Links.Related
	body, err := api.ListBodyFromPath(url)
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdSelectProject(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	organizationId, err := getOrganizationId(api)
	if err != nil {
		return err
	}
	projectId, err := selectProjectId(api, organizationId, "")
	if err != nil {
		return err
	}
	err = save("project", projectId)
	if err != nil {
		return err
	}
	fmt.Printf("Saved project: %s\n", projectId)
	return nil
}

func cliCmdEditProject(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectId, err := getProjectId(api, "")
	if err != nil {
		return err
	}
	project, err := api.Get("projects", projectId)
	if err != nil {
		return err
	}
	err = edit(
		c.String("editor"),
		&project,
		[]string{
			"archived", "description", "homepage_url", "instructions_url", "license",
			"long_description", "machine_translation_fillup", "name", "private",
			"repository_url", "tags",
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdCreateProject(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	organizationId, err := getOrganizationId(api)
	if err != nil {
		return err
	}
	teamId, err := getTeamId(api, organizationId, true)
	if err != nil {
		return err
	}
	sourceLanguageId, err := selectLanguageId(
		api,
		"Select source language",
	)
	if err != nil {
		return err
	}
	body, err := invokeEditor(
		[]byte(createProjectString),
		c.String("editor"),
	)
	if err != nil {
		return err
	}
	var attributes map[string]interface{}
	err = json.Unmarshal(body, &attributes)
	if err != nil {
		return err
	}
	FIELDS := []string{
		"name", "slug", "private", "description", "homepage_url", "instructions_url",
		"license", "long_description", "machine_tranlation_fillup", "repository_url",
		"tags", "translation_memory_fillup", "type",
	}
	for field := range attributes {
		if !stringSliceContains(FIELDS, field) {
			delete(attributes, field)
		}
	}
	project := jsonapi.Resource{
		API:        api,
		Type:       "projects",
		Attributes: attributes,
		Relationships: map[string]*jsonapi.Relationship{
			"organization": {
				Type: jsonapi.SINGULAR,
				DataSingular: &jsonapi.Resource{
					Type: "organizations",
					Id:   organizationId,
				},
			},
			"source_language": {
				Type: jsonapi.SINGULAR,
				DataSingular: &jsonapi.Resource{
					Type: "languages",
					Id:   sourceLanguageId,
				},
			},
		},
	}
	if teamId != "<empty>" {
		project.Relationships["team"] = &jsonapi.Relationship{
			Type:         jsonapi.SINGULAR,
			DataSingular: &jsonapi.Resource{Type: "teams", Id: teamId},
		}
	}
	err = project.Save(nil)
	if err != nil {
		return err
	}
	fmt.Printf("Created project: %s\n", project.Id)
	return nil
}

func cliCmdDeleteProject(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	organizationId, err := getOrganizationId(api)
	if err != nil {
		return err
	}
	projectId, err := selectProjectId(api, organizationId, "Select project to delete:")
	if err != nil {
		return err
	}
	fmt.Printf("About to delete project: %s, are you sure (y/N)? ", projectId)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(strings.ToLower(answer)) == "y" {
		project := jsonapi.Resource{API: api, Type: "projects", Id: projectId}
		err = project.Delete()
		if err != nil {
			return err
		}
		fmt.Printf("Deleted project: %s\n", projectId)
	}
	return nil
}

func cliCmdChangeProjectTeam(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	organizationId, err := getOrganizationId(api)
	if err != nil {
		return err
	}
	projectId, err := getProjectId(api, organizationId)
	if err != nil {
		return err
	}
	teamId, err := selectTeamId(api, organizationId, false)
	if err != nil {
		return err
	}
	project, err := api.Get("projects", projectId)
	if err != nil {
		return err
	}
	project.Relationships["team"].DataSingular.Id = teamId
	err = project.Save([]string{"team"})
	if err != nil {
		return err
	}
	return nil
}

func cliCmdAddProjectLanguages(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectId, err := getProjectId(api, "")
	if err != nil {
		return err
	}
	project, err := api.Get("projects", projectId)
	if err != nil {
		return err
	}
	languageIds, err := selectLanguageIds(api, "", false)
	if err != nil {
		return err
	}
	var languages []*jsonapi.Resource
	for _, languageId := range languageIds {
		languages = append(languages, &jsonapi.Resource{
			Type: "languages",
			Id:   languageId,
		})
	}
	err = project.Add("languages", languages)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdRemoveProjectLanguages(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectId, err := getProjectId(api, "")
	if err != nil {
		return err
	}
	project, err := api.Get("projects", projectId)
	if err != nil {
		return err
	}
	languageIds, err := selectLanguageIds(api, projectId, false)
	if err != nil {
		return err
	}
	var languages []*jsonapi.Resource
	for _, languageId := range languageIds {
		languages = append(languages, &jsonapi.Resource{
			Type: "languages",
			Id:   languageId,
		})
	}
	err = project.Remove("languages", languages)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdResetProjectLanguages(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectId, err := getProjectId(api, "")
	if err != nil {
		return err
	}
	project, err := api.Get("projects", projectId)
	if err != nil {
		return err
	}
	languageIds, err := selectLanguageIds(api, "", true)
	if err != nil {
		return err
	}
	var languages []*jsonapi.Resource
	for _, languageId := range languageIds {
		languages = append(languages, &jsonapi.Resource{
			Type: "languages",
			Id:   languageId,
		})
	}
	err = project.Reset("languages", languages)
	if err != nil {
		return err
	}
	return nil
}
