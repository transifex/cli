package jsonapi

import "fmt"

type CapturedRequest struct {
	Method      string
	Payload     []byte
	ContentType string
}
type MockResponse struct {
	Status   int
	Text     string
	Redirect string
}

type MockRequest struct {
	Response MockResponse
	Request  CapturedRequest
}

type MockEndpoint struct {
	Requests []MockRequest
	Count    int
}

type MockData map[string]*MockEndpoint

func (mockData *MockData) Get(path string) *MockRequest {
	endpoint, exists := (*mockData)[path]
	if !exists {
		return nil
	}
	if endpoint.Count >= len(endpoint.Requests) {
		return nil
	}
	endpoint.Count++
	return &endpoint.Requests[endpoint.Count-1]
}

func GetTestConnection(mockData MockData) Connection {
	return Connection{
		RequestMethod: func(
			method, path string, payload []byte, contentType string,
		) ([]byte, error) {
			mockRequest := mockData.Get(path)
			if mockRequest == nil {
				return nil, fmt.Errorf("%s not found", path)
			}
			mockRequest.Request.Method = method
			mockRequest.Request.Payload = payload
			mockRequest.Request.ContentType = contentType

			err := parseErrorResponse(mockRequest.Response.Status, []byte(mockRequest.Response.Text))
			if err != nil {
				return nil, err
			} else if mockRequest.Response.Redirect == "" {
				return []byte(mockRequest.Response.Text), nil
			} else {
				return nil, &RedirectError{mockRequest.Response.Redirect}
			}
		},
	}
}

func GetMockTextResponse(text string) *MockEndpoint {
	return &MockEndpoint{Requests: []MockRequest{{Response: MockResponse{Text: text}}}}
}
