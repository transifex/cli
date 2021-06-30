package txapi

import (
	"reflect"
	"testing"

	"github.com/transifex/cli/pkg/assert"
	"github.com/transifex/cli/pkg/jsonapi"
)

func TestGetOrganization(t *testing.T) {
	mockData := jsonapi.MockData{
		"/organizations": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{
				{
					Response: jsonapi.MockResponse{
						Text: `{"data": [
							{"type": "organizations",
							 "id": "o:org",
							 "attributes": {"slug": "org"}},
							{"type": "organizations",
							 "id": "o:org2",
							 "attributes": {"slug": "org2"}}
						]}`,
					},
				},
			},
		},
	}
	api := jsonapi.GetTestConnection(mockData)
	organization, err := GetOrganization(&api, "org")
	if err != nil {
		t.Error(err)
	}
	if organization.Id != "o:org" {
		t.Errorf("found id '%s', expected 'o:org'",
			organization.Id)
	}
	if organization.Attributes["slug"] != "org" {
		t.Errorf("found slug '%s', expected 'org'",
			organization.Attributes["slug"])
	}

	if mockData["/organizations"].Count != 1 {
		t.Errorf("got %d calls to '/organizations', expected 1",
			mockData["/organizations"].Count)
	}
	actual := mockData["/organizations"].Requests[0].Request
	expected := jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got request '%+v', expected '%+v'", actual, expected)
	}
}

func TestGetOrganizations(t *testing.T) {
	var capturedMethod, capturedPath, capturedContentType string
	var capturedPayload []byte
	api := jsonapi.Connection{
		RequestMethod: func(
			method, path string, payload []byte, contentType string,
		) ([]byte, error) {
			capturedMethod = method
			capturedPath = path
			capturedPayload = payload
			capturedContentType = contentType
			response := `{"data": [{"type": "organizations",
                                    "id": "o:orgslug",
                                    "attributes": {"name": "Org Name",
                                                   "slug": "orgslug"}},
                                   {"type": "organizations",
                                    "id": "o:orgslug2",
                                    "attributes": {"name": "Org Name2",
                                                   "slug": "orgslug2"}}]}`
			return []byte(response), nil
		},
	}
	organizations, err := GetOrganizations(&api)
	if err != nil {
		t.Errorf("Got error while getting organization: %s", err)
	}

	if capturedMethod != "GET" || capturedPath != "/organizations" ||
		capturedPayload != nil || capturedContentType != "" {
		t.Error("Captured wrong arguments to request")
	}

	assert.Equal(t, len(organizations), 2)

	testCases := []struct {
		Name     string
		Getter   func() interface{}
		Expected string
	}{
		{"type",
			func() interface{} { return organizations[0].Type },
			"organizations"},
		{"id", func() interface{} { return organizations[0].Id }, "o:orgslug"},
		{"name",
			func() interface{} { return organizations[0].Attributes["name"] },
			"Org Name"},
		{"slug",
			func() interface{} { return organizations[0].Attributes["slug"] },
			"orgslug"},
		{"type",
			func() interface{} { return organizations[1].Type },
			"organizations"},
		{"id",
			func() interface{} { return organizations[1].Id },
			"o:orgslug2"},
		{"name",
			func() interface{} { return organizations[1].Attributes["name"] },
			"Org Name2"},
		{"slug",
			func() interface{} { return organizations[1].Attributes["slug"] },
			"orgslug2"},
	}

	for _, testCase := range testCases {
		value := testCase.Getter()
		if value != testCase.Expected {
			t.Errorf("Got %s '%s', expected '%s'",
				testCase.Name, value, testCase.Expected)
		}
	}
}
