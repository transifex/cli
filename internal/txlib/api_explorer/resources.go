package api_explorer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/urfave/cli/v2"
)

const CREATE_RESOURCE_STRING = `{
  "Required fields": "",

  "name": "The name of the resource",
  "slug": "the_slug_of_the_resource",

  "Optional fields (remember to remove the leading '//' from the keys)": "",

  "//priority": "",
  "//accept_translations": true,
  "//categories": [],
  "//i18n_options": {},
  "//mp4_url": "",
  "//ogg_url": "",
  "//webm_url": "",
  "//youtube_url": ""
}`

func selectResourceId(
	api *jsonapi.Connection,
	projectId string,
	header string,
) (string, error) {
	if header == "" {
		header = "Select resource:"
	}
	query := jsonapi.Query{Filters: map[string]string{"project": projectId}}
	body, err := api.ListBody("resources", query.Encode())
	if err != nil {
		return "", err
	}
	resourceId, err := fuzzy(
		api,
		body,
		header,
		func(resource *jsonapi.Resource) string {
			var attributes txapi.ResourceAttributes
			err := resource.MapAttributes(&attributes)
			if err != nil {
				return resource.Id
			}
			return fmt.Sprintf("%s (%s)", attributes.Name, attributes.Slug)
		},
		false,
	)
	if err != nil {
		return "", err
	}
	return resourceId, nil
}

func getResourceId(api *jsonapi.Connection, projectId string) (string, error) {
	resourceId, err := load("resource")
	if err != nil {
		return "", err
	}
	if resourceId == "" {
		if projectId == "" {
			projectId, err = getProjectId(api, "")
			if err != nil {
				return "", err
			}
		}
		resourceId, err = selectResourceId(api, projectId, "Select resource")
		if err != nil {
			return "", err
		}
	}
	return resourceId, nil
}

func cliCmdSelectResource(c *cli.Context) error {
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
	projectId, err := getProjectId(api, organizationId)
	if err != nil {
		return err
	}
	err = save("project", projectId)
	if err != nil {
		return err
	}
	resourceId, err := getResourceId(api, projectId)
	if err != nil {
		return err
	}
	err = save("resource", resourceId)
	if err != nil {
		return err
	}
	fmt.Printf("Saved resource: %s\n", resourceId)
	return nil
}

func cliCmdGetResources(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectId, err := getProjectId(api, "")
	if err != nil {
		return err
	}
	query := jsonapi.Query{
		Filters: map[string]string{"project": projectId},
	}
	if c.String("name") != "" {
		query.Filters["name"] = c.String("name")
	}
	if c.String("slug") != "" {
		query.Filters["slug"] = c.String("slug")
	}
	body, err := api.ListBody("resources", query.Encode())
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

func cliCmdCreateResource(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		_, err = api.Request("POST", "/resources", body, "")
		if err != nil {
			return err
		}
		return nil
	}
	organizationId, err := getOrganizationId(api)
	if err != nil {
		return err
	}
	projectId, err := getProjectId(api, organizationId)
	if err != nil {
		return err
	}
	i18nFormatId, err := selectI18nFormatId(api, organizationId)
	if err != nil {
		return err
	}
	attributes, err := create(
		CREATE_RESOURCE_STRING,
		c.String("editor"),
		[]string{
			"slug", "name", "priority", "accept_translations", "categories",
			"i18n_options", "mp4_url", "ogg_url", "webm_url", "youtube_url",
		},
	)
	if err != nil {
		return err
	}
	resource := jsonapi.Resource{
		API:        api,
		Type:       "resources",
		Attributes: attributes,
	}
	resource.SetRelated("project", &jsonapi.Resource{
		Type: "projects",
		Id:   projectId,
	})
	resource.SetRelated("i18n_format", &jsonapi.Resource{
		Type: "i18n_formats",
		Id:   i18nFormatId,
	})
	err = resource.Save(nil)
	if err != nil {
		return err
	}
	fmt.Printf("Created resource: %s\n", resource.Id)
	return nil
}

func cliCmdDeleteResource(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectId, err := getProjectId(api, "")
	if err != nil {
		return err
	}
	resourceId, err := selectResourceId(api, projectId, "Select resource to delete:")
	if err != nil {
		return err
	}
	fmt.Printf("About to delete resource: %s, are you sure (y/N)? ", resourceId)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(strings.ToLower(answer)) == "y" {
		resource := jsonapi.Resource{API: api, Type: "resources", Id: resourceId}
		err = resource.Delete()
		if err != nil {
			return err
		}
		fmt.Printf("Deleted resource: %s\n", resourceId)
	}
	return nil
}

func cliCmdGetResource(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	resourceId, err := getResourceId(api, "")
	if err != nil {
		return err
	}
	body, err := api.GetBody("resources", resourceId)
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdEditResource(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	resourceId, err := getResourceId(api, "")
	if err != nil {
		return err
	}
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		_, err = api.Request("PATCH", fmt.Sprintf("/resources/%s", resourceId), body, "")
		if err != nil {
			return err
		}
		return nil
	}
	resource, err := api.Get("resources", resourceId)
	if err != nil {
		return err
	}
	err = edit(
		c.String("editor"),
		&resource,
		[]string{
			"name", "priority", "accept_translations", "categories", "mp4_url",
			"ogg_url", "webm_url", "youtube_url", "slug",
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdGetResourceProject(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	resourceId, err := getResourceId(api, "")
	if err != nil {
		return err
	}
	resource, err := api.Get("resources", resourceId)
	if err != nil {
		return err
	}
	url := resource.Relationships["project"].Links.Related
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
