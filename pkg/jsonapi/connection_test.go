package jsonapi

import (
	"testing"
)

func TestGet(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedPayload []byte
	var capturedContentType string

	api := Connection{
		RequestMethod: func(
			method, path string, payload []byte, contentType string,
		) ([]byte, error) {
			capturedMethod = method
			capturedPath = path
			capturedPayload = payload
			capturedContentType = contentType

			response := `{"data": {"type": "students",
                                  "id": "1",
                                  "attributes": {"full_name": "John Doe"}}}`
			return []byte(response), nil
		},
	}

	student, err := api.Get("students", "1")
	if err != nil {
		t.Error(err)
	}

	if capturedMethod != "GET" || capturedPath != "/students/1" ||
		capturedPayload != nil || capturedContentType != "" {
		t.Error("Captured wrong arguments to Request")
	}
	testCases := []struct {
		name     string
		getter   func() any
		expected any
	}{
		{"type", func() any { return student.Type }, "students"},
		{"ID", func() any { return student.Id }, "1"},
		{"full_name",
			func() any { return student.Attributes["full_name"] },
			"John Doe"},
	}
	for _, testCase := range testCases {
		value := testCase.getter()
		if value != testCase.expected {
			t.Errorf("Student's %s was '%s', expected %s",
				testCase.name, value, testCase.expected)
		}
	}
}

func TestList(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedPayload []byte
	var capturedContentType string

	api := Connection{
		RequestMethod: func(
			method, path string, payload []byte, contentType string,
		) ([]byte, error) {
			capturedMethod = method
			capturedPath = path
			capturedPayload = payload
			capturedContentType = contentType

			response := `{"data": [
                {"type": "students",
                 "id": "1",
                 "attributes": {"full_name": "Student One"}},
                {"type": "students",
                 "id": "2",
                 "attributes": {"full_name": "Student Two"}}
            ]}`

			return []byte(response), nil
		},
	}
	students, err := api.List("students", "")
	if err != nil {
		t.Error(err)
	}

	if capturedMethod != "GET" || capturedPath != "/students" ||
		capturedPayload != nil || capturedContentType != "" {
		t.Error("Captured wrong arguments to Request")
	}

	testCases := []struct {
		name     string
		getter   func() any
		expected any
	}{
		{"next", func() any { return students.Next }, ""},
		{"previous", func() any { return students.Previous }, ""},
		{"first student's type",
			func() any { return students.Data[0].Type },
			"students"},
		{"first student's Id",
			func() any { return students.Data[0].Id },
			"1"},
		{
			"first student's full_name",
			func() any {
				return students.Data[0].Attributes["full_name"]
			},
			"Student One",
		},
		{"second student's type",
			func() any { return students.Data[1].Type },
			"students"},
		{"second student's Id",
			func() any { return students.Data[1].Id },
			"2"},
		{
			"second student's full_name",
			func() any {
				return students.Data[1].Attributes["full_name"]
			},
			"Student Two",
		},
	}
	for _, testCase := range testCases {
		value := testCase.getter()
		if value != testCase.expected {
			t.Errorf("Students' %s was '%s', expected '%s'",
				testCase.name, value, testCase.expected)
		}
	}
}
