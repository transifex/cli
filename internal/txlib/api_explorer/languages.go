package api_explorer

import (
	"fmt"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/urfave/cli/v2"
)

func selectLanguageId(api *jsonapi.Connection, header string) (string, error) {
	if header == "" {
		header = "Select language"
	}
	body, err := api.ListBody("languages", "")
	if err != nil {
		return "", err
	}
	languageId, err := fuzzy(
		api,
		body,
		header,
		func(language *jsonapi.Resource) string {
			var attributes txapi.LanguageAttributes
			err := language.MapAttributes(&attributes)
			if err != nil {
				return language.Id
			}
			return fmt.Sprintf("%s (%s)", attributes.Name, attributes.Code)
		},
		false,
	)
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
		func(language *jsonapi.Resource) string {
			var attributes txapi.LanguageAttributes
			language.MapAttributes(&attributes)
			return fmt.Sprintf("%s (%s)", attributes.Name, attributes.Code)
		},
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
	body, err := api.ListBody("languages", "")
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}
