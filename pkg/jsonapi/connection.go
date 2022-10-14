package jsonapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Connection struct {
	Host    string
	Token   string
	Client  http.Client
	Headers map[string]string

	// Used for testing
	RequestMethod func(method, path string,
		payload []byte, contentType string) ([]byte, error)
}

func (c *Connection) request(
	method,
	path string,
	payload []byte,
	contentType string,
) ([]byte, error) {
	if c.RequestMethod != nil {
		return c.RequestMethod(method, path, payload, contentType)
	}

	if strings.HasPrefix(path, "/") {
		path = c.Host + path
	}

	if c.Client.CheckRedirect == nil {
		c.Client.CheckRedirect = func(
			req *http.Request, via []*http.Request,
		) error {
			return &RedirectError{Location: req.URL.String()}
		}
	}

	requestObj, err := http.NewRequest(method, path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	if contentType == "" {
		contentType = "application/vnd.api+json"
	}
	requestObj.Header.Add("Content-Type", contentType)
	requestObj.Header.Add("Authorization", "Bearer "+c.Token)
	for header, value := range c.Headers {
		requestObj.Header.Add(header, value)
	}
	response, err := c.Client.Do(requestObj)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	throttleErrorResponse := parseThrottleResponse(response)
	if throttleErrorResponse != nil {
		return nil, throttleErrorResponse
	}

	errorResponse := parseErrorResponse(response.StatusCode, body)
	if errorResponse != nil {
		return nil, errorResponse
	}

	return body, nil
}

/*
Get
Returns a Resource instance from the server based on its 'type' and 'id'
*/
func (c *Connection) GetBody(Type, Id string) ([]byte, error) {
	url := fmt.Sprintf("/%s/%s", Type, Id)
	body, err := c.request("GET", url, nil, "")
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *Connection) Get(Type, Id string) (Resource, error) {
	url := fmt.Sprintf("/%s/%s", Type, Id)
	return c.getFromPath(url)
}

func (c *Connection) getFromPath(path string) (Resource, error) {
	var result Resource

	body, err := c.request("GET", path, nil, "")
	if err != nil {
		return result, err
	}
	return postProcessGetResponse(c, body)
}

func postProcessGetResponse(c *Connection, body []byte) (Resource, error) {
	var response PayloadSingular
	err := json.Unmarshal(body, &response)
	if err != nil {
		return Resource{}, err
	}
	return payloadToResource(response.Data, nil, c)
}

/*
List
Returns a Collection instance from the server. Query is a URL encoded set of GET
variables that can be easily generated from the Query type and Query.Encode
method.
*/
func (c *Connection) ListBody(Type, Query string) ([]byte, error) {
	Url := fmt.Sprintf("/%s", Type)
	if Query != "" {
		Url = Url + "?" + Query
	}
	body, err := c.request("GET", Url, nil, "")
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *Connection) List(Type, Query string) (Collection, error) {
	body, err := c.ListBody(Type, Query)
	if err != nil {
		return Collection{}, err
	}
	return postProcessListResponse(c, body)
}

func (c *Connection) listFromPath(Url string) (Collection, error) {
	body, err := c.request("GET", Url, nil, "")
	if err != nil {
		return Collection{}, err
	}
	return postProcessListResponse(c, body)
}

func postProcessListResponse(c *Connection, body []byte) (Collection, error) {
	var result Collection

	var response PayloadPluralRead
	err := json.Unmarshal(body, &response)
	if err != nil {
		return result, err
	}

	included, err := makeIncludedMap(response.Included, c)
	if err != nil {
		return result, err
	}

	result.API = c
	result.Previous = response.Links.Previous
	result.Next = response.Links.Next
	result.Data = make([]Resource, 0, len(response.Data))

	for _, item := range response.Data {
		resource, err := payloadToResource(item, &included, c)
		if err != nil {
			return result, err
		}
		result.Data = append(result.Data, resource)
	}

	return result, nil
}
