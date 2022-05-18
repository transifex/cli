package jsonapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

/*
Error type for {json:api} errors.

You can inspect the contents of the error response with type assertions.
Example:

    project := jsonapi.Resource{...}
    err := project.Save() // Here the server responds with an error
    switch e := err.(type) {
    case *jsonapi.Error:
		// "Smartly" inspect the contents of the error
		for _, errorItem := range e.Errors {
			if errorItem.Status == "404" {
				fmt.Println("Something was not found")
			}
		}
    default:
        fmt.Printf("%s\n", e)
    }
*/
type Error struct {
	StatusCode int
	Errors     []ErrorItem `json:"errors"`
}

type ErrorItem struct {
	Status string `json:"status,omitempty"`
	Code   string `json:"code,omitempty"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
	Source struct {
		Pointer   string `json:"pointer,omitempty"`
		Parameter string `json:"parameter,omitempty"`
	} `json:"source,omitempty"`
}

func (e *Error) Error() string {
	// 400:
	result := make([]string, 0, len(e.Errors)+1)
	result = append(result, fmt.Sprint(e.StatusCode))
	for _, errorItem := range e.Errors {
		result = append(result,
			fmt.Sprintf("%s: %s", errorItem.Code, errorItem.Detail))
	}
	return strings.Join(result, ", ")
}

func parseErrorResponse(statusCode int, body []byte) *Error {
	if statusCode < 400 {
		return nil
	}
	errorResponse := Error{StatusCode: statusCode}

	// Intentionally ignore parse errors
	_ = json.Unmarshal(body, &errorResponse)

	return &errorResponse
}

type RedirectError struct {
	Location string
}

func (m *RedirectError) Error() string {
	return "jsonapi does not handle redirects. You can access the Location " +
		"header with " +
		"`var e *jsonapi.RedirectError; errors.As(err, &e); e.Location`"
}

type ThrottleError struct {
	RetryAfter int
}

func (err ThrottleError) Error() string {
	return fmt.Sprintf("throttled; retry after %d", err.RetryAfter)
}

func parseThrottleResponse(response *http.Response) *ThrottleError {
	if response.StatusCode != 429 {
		return nil
	}
	retryAfter, err := strconv.Atoi(response.Header.Get("Retry-After"))
	if err != nil {
		return &ThrottleError{1}
	}
	return &ThrottleError{retryAfter}
}
