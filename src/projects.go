package main

import (
	"bytes"
	"fmt"
	"net/http"
)

// Project represents the "projects" api resource
type Project struct {
	Attributes struct {
		Archived        *bool     `json:"archived,omitempty"`
		Created         *string   `json:"datetime_created,omitempty"`
		Modified        *string   `json:"datetime_modified,omitempty"`
		Description     *string   `json:"description,omitempty"`
		HomepageURL     *string   `json:"homepage_url,omitempty"`
		InstructionsURL *string   `json:"instructions_url,omitempty"`
		License         *string   `json:"license,omitempty"`
		LongDescription *string   `json:"long_description,omitempty"`
		Name            *string   `json:"name,omitempty"`
		Private         *bool     `json:"private,omitempty"`
		RepositoryURL   *string   `json:"repository_url,omitempty"`
		Slug            *string   `json:"slug,omitempty"`
		Tags            *[]string `json:"tags,omitempty"`
		TMFillup        *bool     `json:"translation_memory_fillup,omitempty"`
		Type            *string   `json:"type,omitempty"`
	} `json:"attributes"`
	ID    *string `json:"id,omitempty"`
	Links *struct {
		Self string `json:"self"`
	} `json:"links,omitempty"`
	Relationships struct {
		Languages *struct {
			Links struct {
				Related string `json:"related"`
			} `json:"links"`
		} `json:"languages,omitempty"`
		Organization struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
			Links *struct {
				Related string `json:"related"`
			} `json:"links,omitempty"`
		} `json:"organization"`
		SourceLanguage struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
			Links *struct {
				Related string `json:"related"`
			} `json:"links,omitempty"`
		} `json:"source_language"`
	} `json:"relationships"`
	Type string `json:"type"`
}

func (c *Client) getProjects(organizationID string) (*[]Project, error) {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/projects?filter[organization]=%s", c.Host, organizationID),
		nil,
	)
	projects := []Project{}
	if err := c.DoRequest(req, &projects); err != nil {
		return nil, err
	}
	return &projects, nil
}

func (c *Client) getProject(projectID string) (*Project, error) {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/projects/%s", c.Host, projectID),
		nil,
	)
	project := Project{}
	if err := c.DoRequest(req, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

func (c *Client) createProject(oid string, name string, slug string, private bool, sourceLang string) (*Project, error) {
	project := Project{}
	project.Type = "projects"
	project.Attributes.Slug = &slug
	project.Attributes.Name = &name
	project.Attributes.Private = &private
	project.Relationships.SourceLanguage.Data.Type = "languages"
	project.Relationships.SourceLanguage.Data.ID = "l:" + sourceLang
	project.Relationships.Organization.Data.Type = "organizations"
	project.Relationships.Organization.Data.ID = oid

	body, err := JSONMarshal(map[string]Project{
		"data": project,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/projects", c.Host),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	if err := c.DoRequest(req, &project); err != nil {
		return nil, err
	}
	return &project, nil
}
