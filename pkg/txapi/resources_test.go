package txapi

import (
	"testing"

	"github.com/transifex/cli/pkg/jsonapi"
)

func TestGetResource(t *testing.T) {
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
		`{"data": [
            {
                "type": "resources",
                "id": "o:orgslug:p:projslug:r:resslug",
                "attributes": {"name": "Res Name", "slug": "resslug"},
                "relationships": {
                    "project": {
                        "data": {"type": "projects",
                                 "id": "o:orgslug:p:projslug"},
                        "links": {"related": "/projects/o:orgslug:p:projslug"}
                    }
                }
            },
            {
                "type": "resources",
                "id": "o:orgslug:p:projslug:r:resslug2",
                "attributes": {"name": "Res Name2", "slug": "resslug2"},
                "relationships": {
                    "project": {
                        "data": {"type": "projects",
                                 "id": "o:orgslug:p:projslug"},
                        "links": {"related": "/projects/o:orgslug:p:projslug"}
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
		t.Error(err)
	}
	resource, err := GetResource(&api, project, "resslug")
	if err != nil {
		t.Errorf("Got error while getting project: %s", err)
	}
	testCases := []struct {
		Name     string
		Getter   func() any
		Expected any
	}{
		{"type",
			func() any { return resource.Type },
			"resources"},
		{"id",
			func() any { return resource.Id },
			"o:orgslug:p:projslug:r:resslug"},
		{"name",
			func() any { return resource.Attributes["name"] },
			"Res Name"},
		{"slug",
			func() any { return resource.Attributes["slug"] },
			"resslug"},
		{"project relationship exists",
			func() any {
				_, ok := resource.Relationships["project"]
				return ok
			},
			true},
		{
			"project relationship plurality",
			func() any {
				return resource.Relationships["project"].Type
			},
			jsonapi.SINGULAR,
		},
		{
			"project relationship type",
			func() any {
				return resource.Relationships["project"].DataSingular.Type
			},
			"projects",
		},
		{
			"project relationship id",
			func() any {
				return resource.Relationships["project"].DataSingular.Id
			},
			"o:orgslug:p:projslug",
		},
		{
			"project relationship fetched",
			func() any {
				return resource.Relationships["project"].Fetched
			},
			true,
		},
		{
			"organization relationship name",
			func() any {
				projectRelationship := resource.Relationships["project"]
				project := projectRelationship.DataSingular
				return project.Attributes["name"]
			},
			"Proj Name",
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

func TestGetResources(t *testing.T) {
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
		`{"data": [
            {
                "type": "resources",
                "id": "o:orgslug:p:projslug:r:resslug",
                "attributes": {"name": "Res Name", "slug": "resslug"},
                "relationships": {
                    "project": {
                        "data": {"type": "projects",
                                 "id": "o:orgslug:p:projslug"},
                        "links": {"related": "/projects/o:orgslug:p:projslug"}
                    }
                }
            },
            {
                "type": "resources",
                "id": "o:orgslug:p:projslug:r:resslug2",
                "attributes": {"name": "Res Name2", "slug": "resslug2"},
                "relationships": {
                    "project": {
                        "data": {"type": "projects",
                                 "id": "o:orgslug:p:projslug"},
                        "links": {"related": "/projects/o:orgslug:p:projslug"}
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
		t.Error(err)
	}
	resources, err := GetResources(&api, project)

	if err != nil {
		t.Errorf("Got error while getting project: %s", err)
	}
	testCases := []struct {
		Name     string
		Getter   func() any
		Expected any
	}{
		{"type",
			func() any { return resources[0].Type },
			"resources"},
		{"id",
			func() any { return resources[0].Id },
			"o:orgslug:p:projslug:r:resslug"},
		{"name",
			func() any { return resources[0].Attributes["name"] },
			"Res Name"},
		{"slug",
			func() any { return resources[0].Attributes["slug"] },
			"resslug"},
		{"project relationship exists",
			func() any {
				_, ok := resources[0].Relationships["project"]
				return ok
			},
			true},
		{"type",
			func() any { return resources[1].Type },
			"resources"},
		{"id",
			func() any { return resources[1].Id },
			"o:orgslug:p:projslug:r:resslug2"},
		{"name",
			func() any { return resources[1].Attributes["name"] },
			"Res Name2"},
		{"slug",
			func() any { return resources[1].Attributes["slug"] },
			"resslug2"},
		{"project relationship exists",
			func() any {
				_, ok := resources[0].Relationships["project"]
				return ok
			},
			true},
		{
			"project relationship plurality",
			func() any {
				return resources[0].Relationships["project"].Type
			},
			jsonapi.SINGULAR,
		},
		{
			"project relationship type",
			func() any {
				return resources[0].Relationships["project"].DataSingular.Type
			},
			"projects",
		},
		{
			"project relationship id",
			func() any {
				return resources[0].Relationships["project"].DataSingular.Id
			},
			"o:orgslug:p:projslug",
		},
		{
			"project relationship fetched",
			func() any {
				return resources[0].Relationships["project"].Fetched
			},
			true,
		},
		{
			"organization relationship name",
			func() any {
				projectRelationship := resources[0].Relationships["project"]
				project := projectRelationship.DataSingular
				return project.Attributes["name"]
			},
			"Proj Name",
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

func TestDeleteResource(t *testing.T) {
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
		`{"data": [
            {
                "type": "resources",
                "id": "o:orgslug:p:projslug:r:resslug",
                "attributes": {"name": "Res Name", "slug": "resslug"},
                "relationships": {
                    "project": {
                        "data": {"type": "projects",
                                 "id": "o:orgslug:p:projslug"},
                        "links": {"related": "/projects/o:orgslug:p:projslug"}
                    }
                }
            },
            {
                "type": "resources",
                "id": "o:orgslug:p:projslug:r:resslug2",
                "attributes": {"name": "Res Name2", "slug": "resslug2"},
                "relationships": {
                    "project": {
                        "data": {"type": "projects",
                                 "id": "o:orgslug:p:projslug"},
                        "links": {"related": "/projects/o:orgslug:p:projslug"}
                    }
                }
            }
        ]}`,
		`{}`,
		`{errors:[{}]}`,
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
		t.Error(err)
	}
	resource, err := GetResource(&api, project, "resslug")
	if err != nil {
		t.Errorf("Got error while getting project: %s", err)
	}

	err = DeleteResource(&api, resource)
	if err != nil {
		t.Errorf("Got error while deleting resource: %s", err)
	}
}
