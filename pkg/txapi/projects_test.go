package txapi

import (
	"testing"

	"github.com/transifex/cli/pkg/jsonapi"
)

func TestGetProject(t *testing.T) {
	responseIndex := 0
	responses := []string{
		`{"data": [
			{"type": "organizations",
			 "id": "o:orgslug",
			 "attributes": {"name": "Org Name", "slug": "orgslug"}},
			{"type": "organizations",
			 "id": "orgslug2",
			 "attributes": {"name": "Org Name2", "slug": "orgslug2"}}
		]}`,
		`{"data": [
            {
                "type": "projects",
                "id": "o:orgslug:p:projslug",
                "attributes": {"name": "Proj Name", "slug": "projslug"},
                "relationships": {
                    "organization": {
                        "data": {"type": "organizations", "id": "o:orgslug"},
                        "links": {"related": "/organizations/o:orgslug"}
                    }
                }
            }
        ]}`,
	}
	api := jsonapi.Connection{
		RequestMethod: func(
			method, path string, payload []byte, contentType string,
		) ([]byte, error) {
			response := responses[responseIndex]
			responseIndex += 1
			return []byte(response), nil
		},
	}

	organization, err := GetOrganization(&api, "orgslug")
	if err != nil {
		t.Error(err)
	}
	project, err := GetProject(&api, organization, "projslug")
	if err != nil {
		t.Errorf("Got error while getting project: %s", err)
	}

	testCases := []struct {
		Name     string
		Getter   func() any
		Expected any
	}{
		{"type",
			func() any { return project.Type },
			"projects"},
		{"id",
			func() any { return project.Id },
			"o:orgslug:p:projslug"},
		{"name",
			func() any { return project.Attributes["name"] },
			"Proj Name"},
		{"slug",
			func() any { return project.Attributes["slug"] },
			"projslug"},
		{"organization relationship exists",
			func() any {
				_, ok := project.Relationships["organization"]
				return ok
			},
			true},
		{
			"organization relationship plurality",
			func() any {
				return project.Relationships["organization"].Type
			},
			jsonapi.SINGULAR,
		},
		{
			"organization relationship type",
			func() any {
				return project.Relationships["organization"].DataSingular.Type
			},
			"organizations",
		},
		{
			"organization relationship id",
			func() any {
				return project.Relationships["organization"].DataSingular.Id
			},
			"o:orgslug",
		},
		{
			"organization relationship fetched",
			func() any {
				return project.Relationships["organization"].Fetched
			},
			true,
		},
		{
			"organization relationship name",
			func() any {
				organizationRelationship := project.Relationships["organization"]
				organization := organizationRelationship.DataSingular
				return organization.Attributes["name"]
			},
			"Org Name",
		},
	}
	for _, testCase := range testCases {
		value := testCase.Getter()
		if value != testCase.Expected {
			t.Errorf("Got %s '%s', expected '%s'",
				testCase.Name, value, testCase.Expected)
		}
	}
}

func TestGetProjects(t *testing.T) {
	responseIndex := 0
	responses := []string{
		`{"data": [
			{"type": "organizations",
			 "id": "o:orgslug",
			 "attributes": {"name": "Org Name", "slug": "orgslug"}},
			{"type": "organizations",
			 "id": "orgslug2",
			 "attributes": {"name": "Org Name2", "slug": "orgslug2"}}
		]}`,
		`{"data": [
            {
                "type": "projects",
                "id": "o:orgslug:p:projslug",
                "attributes": {"name": "Proj Name", "slug": "projslug"},
                "relationships": {
                    "organization": {
                        "data": {"type": "organizations", "id": "o:orgslug"},
                        "links": {"related": "/organizations/o:orgslug"}
                    }
                }
            },
			{
				"type": "projects",
                "id": "o:orgslug:p:projslug2",
                "attributes": {"name": "Proj Name2", "slug": "projslug2"},
                "relationships": {
                    "organization": {
                        "data": {"type": "organizations", "id": "o:orgslug"},
                        "links": {"related": "/organizations/o:orgslug"}
                    }
                }
            }
        ]}`,
	}
	api := jsonapi.Connection{
		RequestMethod: func(
			method, path string, payload []byte, contentType string,
		) ([]byte, error) {
			response := responses[responseIndex]
			responseIndex += 1
			return []byte(response), nil
		},
	}

	organization, err := GetOrganization(&api, "orgslug")
	if err != nil {
		t.Error(err)
	}
	projects, err := GetProjects(&api, organization)
	if err != nil {
		t.Errorf("Got error while getting project: %s", err)
	}

	testCases := []struct {
		Name     string
		Getter   func() any
		Expected any
	}{
		{"slug",
			func() any { return projects[0].Attributes["slug"] },
			"projslug"},
		{"type",
			func() any { return projects[0].Type },
			"projects"},
		{"id",
			func() any { return projects[0].Id },
			"o:orgslug:p:projslug"},
		{"name",
			func() any { return projects[0].Attributes["name"] },
			"Proj Name"},
		{"slug",
			func() any { return projects[1].Attributes["slug"] },
			"projslug2"},
		{"type",
			func() any { return projects[1].Type },
			"projects"},
		{"id",
			func() any { return projects[1].Id },
			"o:orgslug:p:projslug2"},
		{"name",
			func() any { return projects[1].Attributes["name"] },
			"Proj Name2"},
		{"slug",
			func() any { return projects[1].Attributes["slug"] },
			"projslug2"},
		{"organization relationship exists",
			func() any {
				_, ok := projects[0].Relationships["organization"]
				return ok
			},
			true},
		{
			"organization relationship plurality",
			func() any {
				return projects[0].Relationships["organization"].Type
			},
			jsonapi.SINGULAR,
		},
		{
			"organization relationship type",
			func() any {
				return projects[0].Relationships["organization"].DataSingular.Type
			},
			"organizations",
		},
		{
			"organization relationship id",
			func() any {
				return projects[0].Relationships["organization"].DataSingular.Id
			},
			"o:orgslug",
		},
		{
			"organization relationship fetched",
			func() any {
				return projects[0].Relationships["organization"].Fetched
			},
			true,
		},
		{
			"organization relationship name",
			func() any {
				organizationRelationship := projects[0].Relationships["organization"]
				organization := organizationRelationship.DataSingular
				return organization.Attributes["name"]
			},
			"Org Name",
		},
	}
	for _, testCase := range testCases {
		value := testCase.Getter()
		if value != testCase.Expected {
			t.Errorf("Got %s '%s', expected '%s'",
				testCase.Name, value, testCase.Expected)
		}
	}
}
