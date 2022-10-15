package api_explorer

import (
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

func cliCmdGetI18nFormats(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	organizationId, err := getOrganizationId(api)
	if err != nil {
		return err
	}
	query := jsonapi.Query{Filters: map[string]string{"organization": organizationId}}
	if c.String("name") != "" {
		query.Filters["name"] = c.String("name")
	}
	body, err := api.ListBody("i18n_formats", query.Encode())
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
