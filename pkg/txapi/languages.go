package txapi

import (
	"sync"

	"github.com/transifex/cli/pkg/jsonapi"
)

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

var GetLanguages = func() func(api *jsonapi.Connection) map[string]*jsonapi.Resource {
	result := make(map[string]*jsonapi.Resource)
	var once sync.Once
	return func(api *jsonapi.Connection) map[string]*jsonapi.Resource {
		once.Do(func() {
			collection, err := api.List("languages", "")
			if err != nil {
				result = nil
				return
			}
			for i := range collection.Data {
				language := collection.Data[i]
				var languageAttributes LanguageAttributes
				err = language.MapAttributes(&languageAttributes)
				if err != nil {
					result = nil
					return
				}
				result[languageAttributes.Code] = &language
			}
		})
		return result
	}
}()

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
