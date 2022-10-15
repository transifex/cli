package api_explorer

import (
	"fmt"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/urfave/cli/v2"
)

func pprintLanguage(language *jsonapi.Resource) string {
	var attributes txapi.LanguageAttributes
	err := language.MapAttributes(&attributes)
	if err != nil {
		return language.Id
	}
	return fmt.Sprintf("%s (%s)", attributes.Name, attributes.Code)
}

func selectLanguageId(api *jsonapi.Connection, header string) (string, error) {
	if header == "" {
		header = "Select language"
	}
	body, err := api.ListBody("languages", "")
	if err != nil {
		return "", err
	}
	languageId, err := fuzzy(api, body, header, pprintLanguage, false)
	if err != nil {
		return "", err
	}
	return languageId, nil
}

func selectLanguageIds(api *jsonapi.Connection, projectId string, allowEmpty bool) ([]string, error) {
	var body []byte
	var err error
	if projectId == "" {
		body, err = api.ListBody("languages", "")
	} else {
		body, err = api.ListBodyFromPath(
			fmt.Sprintf("/projects/%s/languages", projectId),
		)
	}
	if err != nil {
		return nil, err
	}
	languageIds, err := fuzzyMulti(
		api,
		body,
		"Select languages (TAB for multiple selection)",
		pprintLanguage,
		allowEmpty,
	)
	if err != nil {
		return nil, err
	}
	return languageIds, nil
}

func cliCmdGetLanguages(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	query := jsonapi.Query{Filters: make(map[string]string)}
	if c.String("code") != "" {
		query.Filters["code"] = c.String("code")
	}
	if c.String("code-any") != "" {
		query.Filters["code__any"] = c.String("code-any")
	}
	body, err := api.ListBody("languages", query.Encode())
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdGetLanguage(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	languageId, err := selectLanguageId(
		api,
		"Select language",
	)
	if err != nil {
		return err
	}
	body, err := api.GetBody("languages", languageId)
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}
