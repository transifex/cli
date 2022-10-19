package api_explorer

import (
	"encoding/json"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/urfave/cli/v2"
)

type I18nFormatsItem struct {
	Type       string                      `json:"type"`
	Id         string                      `json:"id"`
	Attributes txapi.I18nFormatsAttributes `json:"attributes"`
}

func selectI18nFormatId(
	api *jsonapi.Connection,
	organizationId string,
	includeFileless bool,
) (string, error) {
	query := jsonapi.Query{Filters: map[string]string{"organization": organizationId}}
	body, err := api.ListBody("i18n_formats", query.Encode())
	if err != nil {
		return "", err
	}
	if includeFileless {
		var bodyJson struct {
			Data []I18nFormatsItem `json:"data"`
		}
		err = json.Unmarshal(body, &bodyJson)
		if err != nil {
			return "", err
		}
		found := false
		for _, item := range bodyJson.Data {
			if item.Id == "FILELESS" {
				found = true
				break
			}
		}
		if !found {
			bodyJson.Data = append(bodyJson.Data, I18nFormatsItem{
				Type: "i18n_formats",
				Id:   "FILELESS",
				Attributes: txapi.I18nFormatsAttributes{
					Description:    "Fileless resource",
					FileExtensions: []string{".json"},
					MediaType:      "application/json",
					Name:           "FILELESS",
				},
			})
			body, err = json.Marshal(bodyJson)
			if err != nil {
				return "", err
			}
		}
	}
	i18nFormatId, err := fuzzy(
		api,
		body,
		"Select i18n format",
		nil,
		false,
	)
	if err != nil {
		return "", err
	}
	return i18nFormatId, nil
}

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
