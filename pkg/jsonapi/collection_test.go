package jsonapi

import (
	"testing"
)

func TestPaginationNext(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedPayload []byte

	firstPage := Collection{
		API: &Connection{RequestMethod: func(
			method, path string, payload []byte, contentType string,
		) ([]byte, error) {

			capturedMethod = method
			capturedPath = path
			capturedPayload = payload

			response := `{"data": [{"type": "students",
                                    "id": "1",
                                    "attributes": {"name": "Student One"}},
                                   {"type": "students",
                                    "id": "2",
                                    "attributes": {"name": "Student Two"}}]}`
			return []byte(response), nil
		}},
		Next: "/students?page=2",
	}

	secondPage, err := firstPage.GetNext()
	if err != nil {
		t.Error(err)
	}

	if capturedMethod != "GET" ||
		capturedPath != "/students?page=2" ||
		capturedPayload != nil {
		t.Error("Captured wrong arguments to Request")
	}

	testCases := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{"response's length",
			func() interface{} { return len(secondPage.Data) },
			2},
		{"response's next",
			func() interface{} { return secondPage.Next },
			""},
		{"response's previous",
			func() interface{} { return secondPage.Previous },
			""},
		{"first student's type",
			func() interface{} { return secondPage.Data[0].Type },
			"students"},
		{"first student's id",
			func() interface{} { return secondPage.Data[0].Id },
			"1"},
		{"first student's name",
			func() interface{} { return secondPage.Data[0].Attributes["name"] },
			"Student One"},
		{"second student's type",
			func() interface{} { return secondPage.Data[1].Type },
			"students"},
		{"second student's id",
			func() interface{} { return secondPage.Data[1].Id },
			"2"},
		{"second student's name",
			func() interface{} { return secondPage.Data[1].Attributes["name"] },
			"Student Two"},
	}

	for _, testCase := range testCases {
		value := testCase.getter()
		if value != testCase.expected {
			t.Errorf("Second page's %s was '%s', expected '%s'",
				testCase.name, value, testCase.expected)
		}
	}
}

func TestPaginationPrevious(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedPayload []byte
	var capturedContentType string

	firstPage := Collection{
		API: &Connection{RequestMethod: func(
			method, path string, payload []byte, contentType string,
		) ([]byte, error) {

			capturedMethod = method
			capturedPath = path
			capturedPayload = payload
			capturedContentType = contentType

			response := `{"data": [{"type": "students",
                                    "id": "1",
                                    "attributes": {"name": "Student One"}},
                                   {"type": "students",
                                    "id": "2",
                                    "attributes": {"name": "Student Two"}}]}`
			return []byte(response), nil
		}},
		Previous: "/students?page=1",
	}

	secondPage, err := firstPage.GetPrevious()
	if err != nil {
		t.Error(err)
	}

	if (capturedMethod != "GET" ||
		capturedPath != "/students?page=1" ||
		capturedPayload != nil) ||
		capturedContentType != "" {
		t.Error("Captured wrong arguments to Request")
	}

	testCases := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{"response's length",
			func() interface{} { return len(secondPage.Data) },
			2},
		{"response's next",
			func() interface{} { return secondPage.Next },
			""},
		{"response's previous",
			func() interface{} { return secondPage.Previous },
			""},
		{"first student's type",
			func() interface{} { return secondPage.Data[0].Type },
			"students"},
		{"first student's id",
			func() interface{} { return secondPage.Data[0].Id },
			"1"},
		{"first student's name",
			func() interface{} { return secondPage.Data[0].Attributes["name"] },
			"Student One"},
		{"second student's type",
			func() interface{} { return secondPage.Data[1].Type },
			"students"},
		{"second student's id",
			func() interface{} { return secondPage.Data[1].Id },
			"2"},
		{"second student's name",
			func() interface{} { return secondPage.Data[1].Attributes["name"] },
			"Student Two"},
	}

	for _, testCase := range testCases {
		value := testCase.getter()
		if value != testCase.expected {
			t.Errorf("Second page's %s was '%s', expected '%s'",
				testCase.name, value, testCase.expected)
		}
	}
}
