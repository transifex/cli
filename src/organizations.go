package main

import (
	"fmt"
	"net/http"
)

// Organization represents the "organizations" api resource
type Organization struct {
	Attributes struct {
		LogoURL *string `json:"logo_url,omitempty"`
		Name    *string `json:"name,omitempty"`
		Private *bool   `json:"private,omitempty"`
		Slug    *string `json:"slug,omitempty"`
	} `json:"attributes"`
	ID    string `json:"id"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
	Type string `json:"type"`
}

func (c *Client) getOrganization(orgID string) (*Organization, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/organizations/%s", c.Host, orgID), nil)
	organization := Organization{}
	if err := c.DoRequest(req, &organization); err != nil {
		return nil, err
	}
	return &organization, nil
}

func (c *Client) getOrganizations() (*[]Organization, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/organizations", c.Host), nil)
	organization := []Organization{}
	if err := c.DoRequest(req, &organization); err != nil {
		return nil, err
	}
	return &organization, nil
}
