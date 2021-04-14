package main

import (
	"fmt"
	"net/http"
	"net/url"
)

type ResourceLanguageStats struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
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
	} `json:"attributes"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
	Relationships struct {
		Language struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
			Links struct {
				Related string `json:"related"`
			} `json:"links"`
		} `json:"language"`
		Resource struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
			Links struct {
				Related string `json:"related"`
			} `json:"links"`
		} `json:"resource"`
	} `json:"relationships"`
}

func (c *Client) getResourceLanguageStats(rlstatsID string) (*ResourceLanguageStats, error) {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/resource_language_stats/%s", c.Host, rlstatsID),
		nil,
	)
	stats := ResourceLanguageStats{}
	if err := c.DoRequest(req, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// TODO: no pointers here maybe? Maybe use a struct for filters?
func (c *Client) getProjectLanguageStats(projectID string, resourceID *string, languageID *string) (*[]ResourceLanguageStats, error) {
	u, err := url.Parse(fmt.Sprintf("%s/resource_language_stats", c.Host))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("filter[project]", projectID)
	if resourceID != nil {
		q.Set("filter[resource]", *resourceID)
	}
	if languageID != nil {
		q.Set("filter[language]", *languageID)
	}
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest(
		"GET",
		u.String(),
		nil,
	)
	stats := []ResourceLanguageStats{}
	if err := c.DoRequest(req, &stats); err != nil {
		return nil, err
	}
	return &stats, nil

}
