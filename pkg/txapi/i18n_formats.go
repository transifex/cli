package txapi

import (
	"github.com/transifex/cli/pkg/jsonapi"
)

type I18nFormatsAttributes struct {
	Description    string   `json:"description"`
	FileExtensions []string `json:"file_extensions"`
	MediaType      string   `json:"media_type"`
	Name           string   `json:"name"`
}

func GetI18nFormats(
	api *jsonapi.Connection, organization *jsonapi.Resource,
) ([]*jsonapi.Resource, error) {
	query := jsonapi.Query{Filters: map[string]string{
		"organization": organization.Id,
	}}.Encode()
	i18nFormats, err := api.List("i18n_formats", query)
	if err != nil {
		return nil, err
	}

	var result []*jsonapi.Resource

	for i := range i18nFormats.Data {
		var i18nFormatsAttributes I18nFormatsAttributes
		var i18nFormat = &i18nFormats.Data[i]
		err := i18nFormat.MapAttributes(&i18nFormatsAttributes)
		if err != nil {
			return nil, err
		}
		i18nFormat.SetRelated("organization", organization)
		result = append(result, i18nFormat)
	}

	return result, nil
}
