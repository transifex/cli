package txapi

import "github.com/transifex/cli/pkg/jsonapi"

type LanguageAttributes struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	PluralEquation string `json:"plural_equation"`
	PluralRules    struct {
		Zero  string `json:"zero"`
		One   string `json:"one"`
		Two   string `json:"two"`
		Few   string `json:"few"`
		Many  string `json:"many"`
		Other string `json:"other"`
	} `json:"plural_rules"`
	Rtl bool `json:"rtl"`
}

func GetLanguages(
	api *jsonapi.Connection,
) (map[string]*jsonapi.Resource, error) {
	collection, err := api.List("languages", "")
	if err != nil {
		return nil, err
	}
	result := make(map[string]*jsonapi.Resource)
	for i := range collection.Data {
		language := collection.Data[i]
		var languageAttributes LanguageAttributes
		err = language.MapAttributes(&languageAttributes)
		if err != nil {
			return nil, err
		}
		result[languageAttributes.Code] = &language
	}
	return result, nil
}

func GetLanguage(
	api *jsonapi.Connection, code string,
) (*jsonapi.Resource, error) {
	languages, err := api.List("languages", "")
	if err != nil {
		return nil, err
	}

	var language *jsonapi.Resource

	for _, l := range languages.Data {
		if l.Attributes["code"] == code {
			language = &l
			break
		}
	}

	return language, nil
}
