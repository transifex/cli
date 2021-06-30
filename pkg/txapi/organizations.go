package txapi

import (
	"github.com/transifex/cli/pkg/jsonapi"
)

type OrganizationAttributes struct {
	LogoUrl string `json:"logo_url"`
	Name    string `json:"name"`
	Private bool   `json:"private"`
	Slug    string `json:"slug"`
}

func GetOrganization(
	api *jsonapi.Connection, organizationSlug string,
) (*jsonapi.Resource, error) {
	page, err := api.List("organizations", "")
	if err != nil {
		return nil, err
	}

	for { // pagination
		for i := range page.Data {
			organization := page.Data[i]
			var organizationAttributes OrganizationAttributes
			err := organization.MapAttributes(&organizationAttributes)
			if err != nil {
				return nil, err
			}
			if organizationAttributes.Slug == organizationSlug {
				return &organization, nil
			}
		}
		if page.Next != "" {
			page, err = page.GetNext()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, nil
		}
	}
}

func GetOrganizations(api *jsonapi.Connection) (
	[]*jsonapi.Resource, error,
) {

	organizations, err := api.List("organizations", "")
	if err != nil {
		return nil, err
	}
	var result []*jsonapi.Resource

	for {
		for i := range organizations.Data {
			var organizationAttributes OrganizationAttributes
			var organization = organizations.Data[i]
			err := organization.MapAttributes(&organizationAttributes)
			if err != nil {
				return nil, err
			}
			result = append(result, &organization)
		}
		if organizations.Next == "" {
			break
		} else {
			organizations, err = organizations.GetNext()
			if err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}
