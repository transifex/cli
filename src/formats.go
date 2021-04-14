package main

import (
	"fmt"
	"net/http"
)

// I18NFormat represents the "i18n_formats" api resource
type I18NFormat struct {
	Attributes struct {
		Description    string   `json:"description"`
		FileExtensions []string `json:"file_extensions"`
		MediaType      string   `json:"media_type"`
		Name           string   `json:"name"`
	} `json:"attributes"`
	ID   string `json:"id"`
	Type string `json:"type"`
}

func (c *Client) getI18NFormats(organizationID string) (*[]I18NFormat, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/i18n_formats?filter[organization]=%s", c.Host, organizationID), nil)
	i18nFormats := []I18NFormat{}
	if err := c.DoRequest(req, &i18nFormats); err != nil {
		return nil, err
	}
	return &i18nFormats, nil
}
