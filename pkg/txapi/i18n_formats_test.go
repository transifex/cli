package txapi

import (
	"testing"

	"github.com/transifex/cli/pkg/jsonapi"
)

func TestGetI18nTypes(t *testing.T) {
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
                "type": "i18n_formats",
                "id": "YML_KEY",
                "attributes": {
					"description": "YAML Files based on the content",
					"file_extensions": [ ".yml", ".yaml" ],
					"media_type": "text/plain",
					"name": "YML_KEY"
				},
                "relationships": {
                    "organization": {
                        "data": {"type": "organizations", "id": "o:orgslug"},
                        "links": {"related": "/organizations/o:orgslug"}
                    }
                }
            },
            {
                "type": "i18n_formats",
                "id": "ANDROID",
                "attributes": {
					"description": "Android String Resources",
					"file_extensions": [ ".xml"],
					"media_type": "application/xml",
					"name": "ANDROID"
				},
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
	formats, err := GetI18nFormats(&api, organization)
	if err != nil {
		t.Errorf("Got error while getting project: %s", err)
	}
	ymlFormat, exists := formats["YML_KEY"]
	if !exists {
		t.Error("Format YML_KEY not found")
	}
	androidFormat, exists := formats["ANDROID"]
	if !exists {
		t.Error("Format ANDROID not found")
	}

	testCases := []struct {
		Name     string
		Getter   func() interface{}
		Expected interface{}
	}{
		{"description",
			func() interface{} { return ymlFormat.Attributes["description"] },
			"YAML Files based on the content"},
		{"type",
			func() interface{} { return ymlFormat.Type },
			"i18n_formats"},
		{"id",
			func() interface{} { return ymlFormat.Id },
			"YML_KEY"},
		{"name",
			func() interface{} { return ymlFormat.Attributes["name"] },
			"YML_KEY"},
		{"description",
			func() interface{} { return androidFormat.Attributes["description"] },
			"Android String Resources"},
		{"type",
			func() interface{} { return androidFormat.Type },
			"i18n_formats"},
		{"id",
			func() interface{} { return androidFormat.Id },
			"ANDROID"},
		{"name",
			func() interface{} { return androidFormat.Attributes["name"] },
			"ANDROID"},

		{"organization relationship exists",
			func() interface{} {
				_, ok := ymlFormat.Relationships["organization"]
				return ok
			},
			true},
		{
			"organization relationship plurality",
			func() interface{} {
				return ymlFormat.Relationships["organization"].Type
			},
			jsonapi.SINGULAR,
		},
		{
			"organization relationship type",
			func() interface{} {
				return ymlFormat.Relationships["organization"].DataSingular.Type
			},
			"organizations",
		},
		{
			"organization relationship id",
			func() interface{} {
				return ymlFormat.Relationships["organization"].DataSingular.Id
			},
			"o:orgslug",
		},
		{
			"organization relationship fetched",
			func() interface{} {
				return ymlFormat.Relationships["organization"].Fetched
			},
			true,
		},
		{
			"organization relationship name",
			func() interface{} {
				organizationRelationship := ymlFormat.Relationships["organization"]
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
