package main

import (
	"bytes"
	"fmt"
	"net/http"
)

const ResourceIDRegex = "^o:[a-zA-Z0-9._-]+:p:[a-zA-Z0-9_-]+:r:[a-zA-Z0-9_-]+$"

// Resource represents the "resources" api resource
type Resource struct {
	Attributes struct {
		AcceptTranslations *bool     `json:"accept_translations,omitempty"`
		Categories         *[]string `json:"categories,omitempty"`
		DatetimeCreated    *string   `json:"datetime_created,omitempty"`
		DatetimeModified   *string   `json:"datetime_modified,omitempty"`
		I18NVersion        *int      `json:"i18n_version,omitempty"`
		Mp4URL             *string   `json:"mp4_url,omitempty"`
		Name               *string   `json:"name,omitempty"`
		OggURL             *string   `json:"ogg_url,omitempty"`
		Priority           *string   `json:"priority,omitempty"`
		Slug               *string   `json:"slug,omitempty"`
		StringCount        *int      `json:"string_count,omitempty"`
		WebmURL            *string   `json:"webm_url,omitempty"`
		WordCount          *int      `json:"word_count,omitempty"`
		YoutubeURL         *string   `json:"youtube_url,omitempty"`
	} `json:"attributes"`
	ID    *string `json:"id,omitempty"`
	Links *struct {
		Self string `json:"self"`
	} `json:"links,omitempty"`
	Relationships struct {
		I18NFormat struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
		} `json:"i18n_format"`
		Project struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
			Links *struct {
				Related string `json:"related"`
			} `json:"links,omitempty"`
		} `json:"project"`
	} `json:"relationships"`
	Type string `json:"type"`
}

func (c *Client) getResources(projectID string) (*[]Resource, error) {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/resources?filter[project]=%s", c.Host, projectID),
		nil,
	)
	resources := []Resource{}
	if err := c.DoRequest(req, &resources); err != nil {
		return nil, err
	}
	return &resources, nil

}

func (c *Client) getResource(resourceID string) (*Resource, error) {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/resources/%s", c.Host, resourceID),
		nil,
	)
	resource := Resource{}
	if err := c.DoRequest(req, &resource); err != nil {
		return nil, err
	}
	return &resource, nil

}

func (c *Client) createResource(pid string, name string, slug string, resourceType string) (*Resource, error) {
	resource := Resource{}
	resource.Type = "resources"
	resource.Attributes.Slug = &slug
	resource.Attributes.Name = &name
	resource.Relationships.I18NFormat.Data.Type = "i18n_formats"
	resource.Relationships.I18NFormat.Data.ID = resourceType
	resource.Relationships.Project.Data.Type = "projects"
	resource.Relationships.Project.Data.ID = pid

	body, err := JSONMarshal(map[string]Resource{
		"data": resource,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/resources", c.Host),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	if err := c.DoRequest(req, &resource); err != nil {
		return nil, err
	}
	return &resource, nil
}
