package jsonapi

import (
	"testing"
)

func TestHandleSingleErrorResponse(t *testing.T) {
	body := []byte(`{"errors": [{"status": "400",
                                 "code": "bad_request",
                                 "title": "Bad request",
                                 "detail": "Invalid username"}]}`)
	errorResponse := parseErrorResponse(400, body)
	if errorResponse == nil {
		t.Error("Expected error")
		t.FailNow()
	}
	if errorResponse.StatusCode != 400 {
		t.Errorf("Got status code %d, expected 400", errorResponse.StatusCode)
	}
	expectedError := "400, bad_request: Invalid username"
	if errorResponse.Error() != expectedError {
		t.Errorf("Got error '%s', expected %s",
			errorResponse.Error(), expectedError)
	}
}

func TestHandleSingleErrorResponseStructurally(t *testing.T) {
	body := []byte(`{"errors": [{"status": "400",
                                 "code": "bad_request",
                                 "title": "Bad request",
                                 "detail": "Invalid username"}]}`)
	errorResponse := parseErrorResponse(400, body)
	if errorResponse.StatusCode != 400 {
		t.Errorf("Got status code %d, expected 400", errorResponse.StatusCode)
	}
	if errorResponse == nil {
		t.Error("Expected error")
		t.FailNow()
	}
	// Assign errorResponse to an interface of type error
	var err error = errorResponse

	// Type assertion of error interface type to Error type
	data, ok := err.(*Error)
	if !ok {
		t.Error("Could not type-assert errorResponse to *Error type")
	}

	if data.Errors[0].Status != "400" ||
		data.Errors[0].Code != "bad_request" ||
		data.Errors[0].Title != "Bad request" ||
		data.Errors[0].Detail != "Invalid username" {
		t.Error("Could not parse error data properly")
	}
}

func TestHandleDoubleErrorResponse(t *testing.T) {
	body := []byte(`{"errors": [{"status": "409",
                                 "code": "conflict",
                                 "title": "Conflict",
                                 "detail": "username is already taken"},
                                {"status": "409",
                                 "code": "conflict",
                                 "title": "Conflict",
                                 "detail": "email is already taken"}]}`)
	errorResponse := parseErrorResponse(409, body)
	if errorResponse.StatusCode != 409 {
		t.Errorf("Got status code %d, expected 409", errorResponse.StatusCode)
	}
	if errorResponse == nil {
		t.Error("Expected error")
		t.FailNow()
	}
	expectedError := "409, conflict: username is already taken, conflict: " +
		"email is already taken"
	if errorResponse.Error() != expectedError {
		t.Errorf("Got error '%s', expected %s",
			errorResponse.Error(), expectedError)
	}
}
