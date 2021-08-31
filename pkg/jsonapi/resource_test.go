package jsonapi

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFetchSingular(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedPayload []byte
	var capturedContentType string

	resource := Resource{
		API: &Connection{RequestMethod: func(
			method, path string, payload []byte, contentType string,
		) ([]byte, error) {
			capturedMethod = method
			capturedPath = path
			capturedPayload = payload
			capturedContentType = contentType

			response := `{"data": {"type": "parents",
                                   "id": "2",
                                   "attributes": {"full_name": "Zeus"}}}`
			return []byte(response), nil
		}},
		Type: "children",
		Id:   "1",
		Relationships: map[string]*Relationship{"parent": {
			Type:         SINGULAR,
			Fetched:      false,
			DataSingular: &Resource{Type: "parents", Id: "2"},
			Links:        Links{Related: "/parents/2"},
		}},
	}

	parent, err := resource.Fetch("parent")
	if err != nil {
		t.Error(err)
	}

	if capturedMethod != "GET" || capturedPath != "/parents/2" ||
		capturedPayload != nil || capturedContentType != "" {
		t.Error("Captured wrong arguments to Request")
	}

	testCases := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{"parent type", func() interface{} { return parent.Type }, SINGULAR},
		{"fetched", func() interface{} { return parent.Fetched }, true},
		{
			"type",
			func() interface{} { return parent.DataSingular.Type },
			"parents",
		},
		{"ID", func() interface{} { return parent.DataSingular.Id }, "2"},
		{
			"full_name",
			func() interface{} {
				return parent.DataSingular.Attributes["full_name"]
			},
			"Zeus",
		},
	}

	for _, testCase := range testCases {
		value := testCase.getter()
		if value != testCase.expected {
			t.Errorf("Relationship's %s was '%s', expected '%s'",
				testCase.name, value, testCase.expected)
		}
	}
}

func TestFetchPlural(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedPayload []byte
	var capturedContentType string

	resource := Resource{
		API: &Connection{RequestMethod: func(
			method, path string, payload []byte, contentType string,
		) ([]byte, error) {

			capturedMethod = method
			capturedPath = path
			capturedPayload = payload
			capturedContentType = contentType

			response := `{"data": [
				{"type": "children",
				 "id": "2",
				 "attributes": {"full_name": "Child One"}},
				{"type": "children",
				 "id": "3",
				 "attributes": {"full_name": "Child Two"}}
			]}`

			return []byte(response), nil
		}},
		Type: "parents",
		Id:   "1",
		Relationships: map[string]*Relationship{"children": {
			Type:    PLURAL,
			Fetched: false,
			Links:   Links{Related: "/parents/1/children"},
		}},
	}

	relationship, err := resource.Fetch("children")
	if err != nil {
		t.Error(err)
	}

	if capturedMethod != "GET" ||
		capturedPath != "/parents/1/children" ||
		capturedPayload != nil || capturedContentType != "" {
		t.Error("Captured wrong arguments to Request")
	}
	testCases := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{
			"relationship type",
			func() interface{} { return relationship.Type },
			PLURAL,
		},
		{"fetched", func() interface{} { return relationship.Fetched }, true},
		{
			"next",
			func() interface{} { return relationship.DataPlural.Next },
			"",
		},
		{
			"previous",
			func() interface{} { return relationship.DataPlural.Previous },
			"",
		},
		{
			"API",
			func() interface{} { return relationship.DataPlural.API },
			resource.API,
		},
		{
			"first child's type",
			func() interface{} { return relationship.DataPlural.Data[0].Type },
			"children",
		},
		{
			"first child's ID",
			func() interface{} { return relationship.DataPlural.Data[0].Id },
			"2",
		},
		{
			"first child's full_name",
			func() interface{} {
				return relationship.DataPlural.Data[0].Attributes["full_name"]
			},
			"Child One",
		},
		{
			"second child's type",
			func() interface{} { return relationship.DataPlural.Data[1].Type },
			"children",
		},
		{
			"second child's ID",
			func() interface{} { return relationship.DataPlural.Data[1].Id },
			"3",
		},
		{
			"second child's full_name",
			func() interface{} {
				return relationship.DataPlural.Data[1].Attributes["full_name"]
			},
			"Child Two",
		},
	}

	for _, testCase := range testCases {
		value := testCase.getter()
		if value != testCase.expected {
			t.Errorf("Relationship's %s was '%s', expected '%s'",
				testCase.name,
				value,
				testCase.expected)
		}
	}
}

func TestSaveExisting(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedPayload []byte
	var capturedContentType string

	resource := Resource{
		API: &Connection{RequestMethod: func(
			method, path string, payload []byte,
			contentType string) ([]byte, error) {

			capturedMethod = method
			capturedPath = path
			capturedPayload = payload
			capturedContentType = contentType

			response := `{"data": {"type": "students",
                                   "id": "1",
                                   "attributes": {"name": "My Name",
                                                  "created": "yesterday",
                                                  "updated": "right now"}}}`
			return []byte(response), nil
		}},
		Type:       "students",
		Id:         "1",
		Attributes: map[string]interface{}{"name": "My name"},
	}
	err := resource.Save([]string{"name"})
	if err != nil {
		t.Error(err)
	}

	if capturedMethod != "PATCH" ||
		capturedPath != "/students/1" ||
		capturedContentType != "" {
		t.Error("Captured wrong arguments to Request")
	}
	expectedPayload := `{"data": {"type": "students",
                                  "id": "1",
                                  "attributes": {"name": "My name"}}}`
	isEqual, err := jsonEqual(capturedPayload, []byte(expectedPayload))
	if err != nil {
		t.Error(err)
	}
	if !isEqual {
		t.Errorf("Captured payload %s, expected %s",
			string(capturedPayload), expectedPayload)
	}

	testCases := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{"name",
			func() interface{} { return resource.Attributes["name"] },
			"My Name"},
		{"created",
			func() interface{} { return resource.Attributes["created"] },
			"yesterday"},
		{"updated",
			func() interface{} { return resource.Attributes["updated"] },
			"right now"},
	}
	for _, testCase := range testCases {
		value := testCase.getter()
		if value != testCase.expected {
			t.Errorf("Student's %s was '%s', expected '%s'",
				testCase.name, value, testCase.expected)
		}
	}
}

func TestSaveNew(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedPayload []byte
	var capturedContentType string

	resource := Resource{
		API: &Connection{RequestMethod: func(
			method, path string, payload []byte,
			contentType string) ([]byte, error) {

			capturedMethod = method
			capturedPath = path
			capturedPayload = payload
			capturedContentType = contentType

			response := `{"data": {"type": "students",
                                   "id": "1",
                                   "attributes": {"name": "My Name",
                                                  "created": "right now",
                                                  "updated": "right now"}}}`
			return []byte(response), nil
		}},
		Type:       "students",
		Attributes: map[string]interface{}{"name": "My name"},
	}
	err := resource.Save([]string{"name"})
	if err != nil {
		t.Error(err)
	}

	if capturedMethod != "POST" ||
		capturedPath != "/students" ||
		capturedContentType != "" {
		t.Error("Captured wrong arguments to Request")
	}
	expectedPayload := `{"data": {"type": "students",
                                  "attributes": {"name": "My name"}}}`
	isEqual, err := jsonEqual(capturedPayload, []byte(expectedPayload))
	if err != nil {
		t.Error(err)
	}
	if !isEqual {
		t.Errorf("Captured payload %s, expected %s",
			string(capturedPayload), expectedPayload)
	}

	testCases := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{"ID",
			func() interface{} { return resource.Id },
			"1"},
		{"name",
			func() interface{} { return resource.Attributes["name"] },
			"My Name"},
		{"created",
			func() interface{} { return resource.Attributes["created"] },
			"right now"},
		{"updated",
			func() interface{} { return resource.Attributes["updated"] },
			"right now"},
	}
	for _, testCase := range testCases {
		value := testCase.getter()
		if value != testCase.expected {
			t.Errorf("Student's %s was '%s', expected '%s'",
				testCase.name, value, testCase.expected)
		}
	}
}

func TestSaveReplacesSingularRelationship(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedPayload []byte
	var capturedContentType string

	resource := Resource{
		API: &Connection{RequestMethod: func(
			method, path string, payload []byte, contentType string,
		) ([]byte, error) {
			capturedMethod = method
			capturedPath = path
			capturedPayload = payload
			capturedContentType = contentType

			response := `{"data": {
                "type": "children",
                "id": "1",
                "relationships": {"parent": {"data": {"type": "parents",
                                                      "id": "3"}}}
            }}`
			return []byte(response), nil
		}},
		Type: "children",
		Id:   "1",
		Relationships: map[string]*Relationship{"parent": {
			Type:         SINGULAR,
			DataSingular: &Resource{Type: "parents", Id: "2"},
		}},
	}

	err := resource.Save([]string{"parent"})
	if err != nil {
		t.Error(err)
	}

	if capturedMethod != "PATCH" ||
		capturedPath != "/children/1" ||
		capturedContentType != "" {
		t.Error("Captured wrong arguments to Request")
	}
	expectedPayload := `{"data": {
        "type": "children",
        "id": "1",
        "relationships": {"parent": {"data": {"type": "parents", "id": "2"}}}
    }}`
	isEqual, err := jsonEqual(capturedPayload, []byte(expectedPayload))
	if err != nil {
		t.Error(err)
	}
	if !isEqual {
		t.Errorf("Captured payload %s, expected %s",
			string(capturedPayload), expectedPayload)
	}

	if resource.Relationships["parent"].DataSingular.Id != "3" {
		t.Errorf("Resource's new parent has Id '%s', expected '3'",
			resource.Relationships["parent"].DataSingular.Id)
	}
}

func TestSaveAddsIncludedRelationship(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedPayload []byte
	var capturedContentType string

	resource := Resource{
		API: &Connection{RequestMethod: func(
			method, path string, payload []byte, contentType string,
		) ([]byte, error) {
			capturedMethod = method
			capturedPath = path
			capturedPayload = payload
			capturedContentType = contentType

			response := `{
                "data": {
                    "type": "children",
                    "id": "1",
                    "relationships": {"parent": {"data": {"type": "parents",
                                                          "id": "2"}}}
                },
                "included": [{"type": "parents",
                              "id": "2",
                              "attributes": {"name": "Zeus"}}]
            }`
			return []byte(response), nil
		}},
		Type: "children",
		Id:   "1",
		Relationships: map[string]*Relationship{"parent": {
			Type:         SINGULAR,
			Fetched:      false,
			DataSingular: &Resource{Type: "parents", Id: "2"},
		}},
	}
	err := resource.Save([]string{"parent"})
	if err != nil {
		t.Error(err)
	}

	if capturedMethod != "PATCH" ||
		capturedPath != "/children/1" ||
		capturedContentType != "" {
		t.Error("Captured wrong arguments to Request")
	}

	expectedPayload := `{"data": {
        "type": "children",
        "id": "1",
        "relationships": {"parent": {"data": {"type": "parents", "id": "2"}}}
    }}`
	isEqual, err := jsonEqual(capturedPayload, []byte(expectedPayload))
	if err != nil {
		t.Error(err)
	}
	if !isEqual {
		t.Errorf("Captured payload %s, expected %s",
			string(capturedPayload), expectedPayload)
	}

	fetched := resource.Relationships["parent"].Fetched
	name, exists := resource.Relationships["parent"].DataSingular.Attributes["name"]
	if !fetched || !exists || name != "Zeus" {
		t.Error("Parent isn't fetched properly")
	}
}

type ClubsType struct {
	Football   string `json:"football"`
	Basketball string `json:"basketball"`
}
type TestAttributes struct {
	Name    string    `json:"name"`
	Age     int       `json:"age"`
	Married bool      `json:"married"`
	Hobbies []string  `json:"hobbies"`
	Clubs   ClubsType `json:"clubs"`
}

func TestMapAttributes(t *testing.T) {
	resource := Resource{
		Attributes: map[string]interface{}{
			"name":    "John",
			"age":     15,
			"married": false,
			"hobbies": []string{"hockey", "piano", "chess"},
			"clubs": map[string]string{
				"football":   "Aris",
				"basketball": "Panathinaikos",
			},
		},
	}
	var testAttributes TestAttributes
	err := resource.MapAttributes(&testAttributes)
	if err != nil {
		t.Error(err)
	}

	if testAttributes.Name != "John" ||
		testAttributes.Age != 15 ||
		testAttributes.Married != false ||
		len(testAttributes.Hobbies) != 3 ||
		testAttributes.Hobbies[0] != "hockey" ||
		testAttributes.Hobbies[1] != "piano" ||
		testAttributes.Hobbies[2] != "chess" ||
		testAttributes.Clubs.Football != "Aris" ||
		testAttributes.Clubs.Basketball != "Panathinaikos" {
		t.Errorf("Mapped attributes %v are different than original %s",
			testAttributes, resource.Attributes)
	}
}

func TestUnmapAttributes(t *testing.T) {
	testAttributes := TestAttributes{
		Name:    "John",
		Age:     15,
		Married: false,
		Hobbies: []string{"hockey", "piano", "chess"},
		Clubs: ClubsType{
			Basketball: "Aris",
			Football:   "Panathinaikos",
		},
	}
	resource := Resource{}
	err := resource.UnmapAttributes(testAttributes)
	if err != nil {
		t.Error(err)
	}

	left, err := json.Marshal(resource.Attributes)
	if err != nil {
		t.Error(err)
	}

	right, err := json.Marshal(testAttributes)
	if err != nil {
		t.Error(err)
	}

	isEqual, err := jsonEqual(left, right)
	if err != nil {
		t.Error(err)
	}
	if !isEqual {
		t.Errorf("Mapped attributes %s are different than original %s",
			string(left), string(right))
	}
}

func TestAdd(t *testing.T) {
	mockData := MockData{
		"/teachers/t1": &MockEndpoint{
			Requests: []MockRequest{{
				Response: MockResponse{
					Text: `{"data": {
						"type": "teachers",
						"id": "t1",
						"relationships": {"students": {"links": {
							"self": "/teachers/t1/relationships/students"
						}}}
					}}`,
				},
			}},
		},
		"/teachers/t1/relationships/students": &MockEndpoint{
			Requests: []MockRequest{{}},
		},
	}
	api := GetTestConnection(mockData)
	teacher, err := api.Get("teachers", "t1")
	if err != nil {
		t.Error(err)
	}
	err = teacher.Add("students", []*Resource{
		{Type: "students", Id: "s1"},
		{Type: "students", Id: "s2"},
	})
	if err != nil {
		t.Error(err)
	}

	endpoint := mockData["/teachers/t1"]
	if endpoint.Count != 1 {
		t.Errorf("Got %d requests to '/teachers/t1', expected 1",
			endpoint.Count)
	}
	actual := endpoint.Requests[0].Request
	expected := CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected '%+v'", actual, expected)
	}

	endpoint = mockData["/teachers/t1/relationships/students"]
	if endpoint.Count != 1 {
		t.Errorf("Got %d requests to '/teachers/t1', expected 1",
			endpoint.Count)
	}
	actual = endpoint.Requests[0].Request
	if actual.Method != "POST" || actual.ContentType != "" {
		t.Errorf("Got wrong request '%+v'", actual)
	}

	var actualPayload interface{}
	err = json.Unmarshal(actual.Payload, &actualPayload)
	if err != nil {
		t.Error(err)
	}
	var expectedPayload interface{}
	err = json.Unmarshal([]byte(`{"data": [
		{"type": "students", "id": "s1"},
		{"type": "students", "id": "s2"}
	]}`), &expectedPayload)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(actualPayload, expectedPayload) {
		t.Errorf("Got payload '%+v', expected '%+v'",
			actualPayload,
			expectedPayload)
	}
}

func TestRemove(t *testing.T) {
	mockData := MockData{
		"/teachers/t1": &MockEndpoint{
			Requests: []MockRequest{{
				Response: MockResponse{
					Text: `{"data": {
						"type": "teachers",
						"id": "t1",
						"relationships": {"students": {"links": {
							"self": "/teachers/t1/relationships/students"
						}}}
					}}`,
				},
			}},
		},
		"/teachers/t1/relationships/students": &MockEndpoint{
			Requests: []MockRequest{{}},
		},
	}
	api := GetTestConnection(mockData)
	teacher, err := api.Get("teachers", "t1")
	if err != nil {
		t.Error(err)
	}
	err = teacher.Remove("students", []*Resource{
		{Type: "students", Id: "s1"},
		{Type: "students", Id: "s2"},
	})
	if err != nil {
		t.Error(err)
	}

	endpoint := mockData["/teachers/t1"]
	if endpoint.Count != 1 {
		t.Errorf("Got %d requests to '/teachers/t1', expected 1",
			endpoint.Count)
	}
	actual := endpoint.Requests[0].Request
	expected := CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected '%+v'", actual, expected)
	}

	endpoint = mockData["/teachers/t1/relationships/students"]
	if endpoint.Count != 1 {
		t.Errorf("Got %d requests to '/teachers/t1', expected 1",
			endpoint.Count)
	}
	actual = endpoint.Requests[0].Request
	if actual.Method != "DELETE" || actual.ContentType != "" {
		t.Errorf("Got wrong request '%+v'", actual)
	}

	var actualPayload interface{}
	err = json.Unmarshal(actual.Payload, &actualPayload)
	if err != nil {
		t.Error(err)
	}
	var expectedPayload interface{}
	err = json.Unmarshal([]byte(`{"data": [
		{"type": "students", "id": "s1"},
		{"type": "students", "id": "s2"}
	]}`), &expectedPayload)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(actualPayload, expectedPayload) {
		t.Errorf("Got payload '%+v', expected '%+v'",
			actualPayload,
			expectedPayload)
	}
}

func TestReset(t *testing.T) {
	mockData := MockData{
		"/teachers/t1": &MockEndpoint{
			Requests: []MockRequest{{
				Response: MockResponse{
					Text: `{"data": {
						"type": "teachers",
						"id": "t1",
						"relationships": {"students": {"links": {
							"self": "/teachers/t1/relationships/students"
						}}}
					}}`,
				},
			}},
		},
		"/teachers/t1/relationships/students": &MockEndpoint{
			Requests: []MockRequest{{}},
		},
	}
	api := GetTestConnection(mockData)
	teacher, err := api.Get("teachers", "t1")
	if err != nil {
		t.Error(err)
	}
	err = teacher.Reset("students", []*Resource{
		{Type: "students", Id: "s1"},
		{Type: "students", Id: "s2"},
	})
	if err != nil {
		t.Error(err)
	}

	endpoint := mockData["/teachers/t1"]
	if endpoint.Count != 1 {
		t.Errorf("Got %d requests to '/teachers/t1', expected 1",
			endpoint.Count)
	}
	actual := endpoint.Requests[0].Request
	expected := CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected '%+v'", actual, expected)
	}

	endpoint = mockData["/teachers/t1/relationships/students"]
	if endpoint.Count != 1 {
		t.Errorf("Got %d requests to '/teachers/t1', expected 1",
			endpoint.Count)
	}
	actual = endpoint.Requests[0].Request
	if actual.Method != "PATCH" || actual.ContentType != "" {
		t.Errorf("Got wrong request '%+v'", actual)
	}

	var actualPayload interface{}
	err = json.Unmarshal(actual.Payload, &actualPayload)
	if err != nil {
		t.Error(err)
	}
	var expectedPayload interface{}
	err = json.Unmarshal([]byte(`{"data": [
		{"type": "students", "id": "s1"},
		{"type": "students", "id": "s2"}
	]}`), &expectedPayload)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(actualPayload, expectedPayload) {
		t.Errorf("Got payload '%+v', expected '%+v'",
			actualPayload,
			expectedPayload)
	}
}

func TestDelete(t *testing.T) {
	mockData := MockData{
		"/teachers/t1": &MockEndpoint{
			Requests: []MockRequest{
				{
					Response: MockResponse{
						Text: `{"data": {
							"type": "teachers",
							"id": "t1",
							"links": {
								"self": "/teachers/t1"
							}
						}}`,
					},
				},
				{
					Response: MockResponse{
						Text: `{}`,
					},
				},
			},
		},
	}

	api := GetTestConnection(mockData)
	teacher, err := api.Get("teachers", "t1")

	if err != nil {
		t.Error(err)
	}

	endpoint := mockData["/teachers/t1"]
	if endpoint.Count != 1 {
		t.Errorf("Got %d requests to '/teachers/t1', expected 1",
			endpoint.Count)
	}
	actual := endpoint.Requests[0].Request
	expected := CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected '%+v'", actual, expected)
	}

	err = teacher.Delete()
	if err != nil {
		t.Errorf("Deletion should not return error: %s", err)
	}

}
