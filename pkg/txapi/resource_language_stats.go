package txapi

import (
	"github.com/transifex/cli/pkg/jsonapi"
)

type ResourceLanguageStatsAttributes struct {
	LastProofreadUpdate   string `json:"last_proofread_update"`
	LastReviewUpdate      string `json:"last_review_update"`
	LastTranslationUpdate string `json:"last_translation_update"`
	LastUpdate            string `json:"last_update"`
	ProofreadStrings      int    `json:"proofread_strings"`
	ProofreadWords        int    `json:"proofread_words"`
	ReviewedStrings       int    `json:"reviewed_strings"`
	ReviewedWords         int    `json:"reviewed_words"`
	TotalStrings          int    `json:"total_strings"`
	TotalWords            int    `json:"total_words"`
	TranslatedStrings     int    `json:"translated_strings"`
	TranslatedWords       int    `json:"translated_words"`
	UntranslatedStrings   int    `json:"untranslated_strings"`
	UntranslatedWords     int    `json:"untranslated_words"`
}

func GetResourceStats(
	api *jsonapi.Connection, resource, language *jsonapi.Resource,
) (map[string]*jsonapi.Resource, error) {
	query := jsonapi.Query{Filters: map[string]string{
		"project":  resource.Relationships["project"].DataSingular.Id,
		"resource": resource.Id,
	}}
	if language != nil {
		query.Filters["language"] = language.Id
	}
	page, err := api.List("resource_language_stats", query.Encode())
	if err != nil {
		return nil, err
	}
	result := make(map[string]*jsonapi.Resource)
	for {
		for i := range page.Data {
			stats := page.Data[i]
			stats.SetRelated("resource", resource)
			result[stats.Relationships["language"].DataSingular.Id] = &stats
		}
		if page.Next == "" {
			break
		} else {
			page, err = page.GetNext()
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}
