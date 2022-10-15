package api_explorer

import (
	"fmt"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/urfave/cli/v2"
)

func selectOrganizationId(api *jsonapi.Connection) (string, error) {
	body, err := api.ListBody("organizations", "")
	if err != nil {
		return "", err
	}
	organizationId, err := fuzzy(
		api,
		body,
		"Select organization",
		func(organization *jsonapi.Resource) string {
			var attributes txapi.OrganizationAttributes
			err := organization.MapAttributes(&attributes)
			if err != nil {
				return organization.Id
			}
			return fmt.Sprintf("%s (%s)", attributes.Name, attributes.Slug)
		},
		false,
	)
	if err != nil {
		return "", err
	}
	return organizationId, nil
}

func getOrganizationId(api *jsonapi.Connection) (string, error) {
	organizationId, err := load("organization")
	if err != nil {
		return "", err
	}
	if organizationId == "" {
		organizationId, err = selectOrganizationId(api)
		if err != nil {
			return "", err
		}
	}
	return organizationId, nil
}

func cliCmdGetOrganizations(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	query := jsonapi.Query{Filters: make(map[string]string)}
	if c.String("slug") != "" {
		query.Filters["slug"] = c.String("slug")
	}
	body, err := api.ListBody("organizations", query.Encode())
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdGetOrganization(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	organizationId, err := getOrganizationId(api)
	if err != nil {
		return err
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
}

func cliCmdSelectOrganization(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	organizationId, err := selectOrganizationId(api)
	if err != nil {
		return err
	}
	err = save("organization", organizationId)
	if err != nil {
		return err
	}
	fmt.Printf("Saved organization: %s\n", organizationId)
	return nil
}
