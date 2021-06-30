package txlib

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
)

func TestPullCommandResourceExists(t *testing.T) {
	cfg := config.Config{
		Local: &config.LocalConfig{
			Resources: []config.Resource{
				{
					OrganizationSlug: "orgslug",
					ProjectSlug:      "projslug",
					ResourceSlug:     "resslug",
					Type:             "I18N_TYPE",
					SourceFile:       "source",
				},
			},
		},
	}

	projectsUrl := "/projects?" +
		"filter%5Borganization%5D=o%3Aorgslug&filter%5Bslug%5D=projslug"
	resourcesUrl := "/resources?filter%5Bproject%5D=o%3Aorgslug%3Ap%3Aprojslug"
	projectLanguagesUrl := "/projects/o:orgslug:p:projslug/languages"
	mockData := jsonapi.MockData{
		"/organizations": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "organizations",
					                  "id": "o:orgslug",
									  "attributes": {"slug": "orgslug"}}]}`,
				},
			}},
		},
		projectsUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{
						"type": "project",
						"id": "o:orgslug:p:projslug",
						"relationships": {
							"organization": {},
							"languages": {"links": {
								"related": "/projects/o:orgslug:p:projslug/languages"
							}}
						}
					}]}`,
				},
			}},
		},
		projectLanguagesUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "languages",
						              "id": "l:el",
						              "attributes": {"code": "el"}}]}`,
				},
			}},
		},
		resourcesUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "resources",
						              "id": "o:orgslug:p:projslug:r:resslug",
						              "attributes": {"slug": "resslug"},
						              "relationships": {"project": {}}}],
							"links": {"next": "",
									  "previous": "",
									  "self": ""}}`,
				},
			}},
		},
		"/resource_translations_async_downloads": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": {
						"type": "resource_translations_async_downloads",
						"id": "download_1",
						"relationships": {"resource": {"data": {
							"type": "resources",
							"id": "o:orgslug:p:projslug:r:resslug"
						}}}
					}}`,
				},
			}},
		},
		"/resource_translations_async_downloads/download_1": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": {
						"type": "resource_translations_async_downloads",
						"id": "download_1",
						"attributes": {"status": "succeeded"}
					}}`,
				},
			}},
		},
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:    "default",
		Mode:        "default",
		Force:       true,
		All:         true,
		ResourceIds: nil,
	}

	err := PullCommand(&cfg, api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	endpoint := mockData["/organizations"]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '/organizations', expected 1",
			endpoint.Count)
	}
	actual := endpoint.Requests[0].Request
	expected := jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected %+v", actual, expected)
	}

	endpoint = mockData[projectsUrl]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '%s', expected 1",
			endpoint.Count, projectsUrl)
	}
	actual = endpoint.Requests[0].Request
	expected = jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected %+v", actual, expected)
	}

	endpoint = mockData[resourcesUrl]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '%s', expected 1",
			endpoint.Count, resourcesUrl)
	}
	actual = endpoint.Requests[0].Request
	expected = jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected %+v", actual, expected)
	}

	endpoint = mockData["/resource_translations_async_downloads"]
	if endpoint.Count != 1 {
		t.Errorf(
			"Made %d requests to '/resource_translations_async_downloads', "+
				"expected 1",
			endpoint.Count,
		)
	}
	actual = endpoint.Requests[0].Request
	if actual.Method != "POST" ||
		len(actual.Payload) == 0 {
		t.Errorf("Something was wrong with the request '%+v'", actual)
	}

	endpoint = mockData["/resource_translations_async_downloads/download_1"]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '%s', expected 1",
			endpoint.Count,
			"/resource_translations_async_downloads/download_1")
	}
	actual = endpoint.Requests[0].Request
	expected = jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected %+v", actual, expected)
	}
}

func TestPullCommandFileExists(t *testing.T) {
	cfg := config.Config{
		Local: &config.LocalConfig{
			Resources: []config.Resource{
				{
					OrganizationSlug: "orgslug",
					ProjectSlug:      "projslug",
					ResourceSlug:     "resslug",
					Type:             "I18N_TYPE",
					SourceFile:       "source",
					FileFilter:       "locale/<lang>/file",
					LanguageMappings: map[string]string{
						"el": "el",
					},
				},
			},
		},
	}

	ts := createTestServer()
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(filepath.Join(workingDir, "locale", "el"), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	ts.Start()
	defer ts.Close()

	projectsUrl := "/projects?" +
		"filter%5Borganization%5D=o%3Aorgslug&filter%5Bslug%5D=projslug"
	resourcesUrl := "/resources?filter%5Bproject%5D=o%3Aorgslug%3Ap%3Aprojslug"
	projectLanguagesUrl := "/projects/o:orgslug:p:projslug/languages"
	asyncDownloadsUrl := "/resource_translations_async_downloads/download_1"
	mockData := jsonapi.MockData{
		"/organizations": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "organizations",
					                  "id": "o:orgslug",
									  "attributes": {"slug": "orgslug"}}]}`,
				},
			}},
		},
		projectsUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{
						"type": "project",
						"id": "o:orgslug:p:projslug",
						"relationships": {
							"organization": {},
							"languages": {"links": {
								"related": "/projects/o:orgslug:p:projslug/languages"
							}}
						}
					}]}`,
				},
			}},
		},
		projectLanguagesUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "languages",
						              "id": "l:el",
						              "attributes": {"code": "el"}}]}`,
				},
			}},
		},
		resourcesUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "resources",
						              "id": "o:orgslug:p:projslug:r:resslug",
						              "attributes": {"slug": "resslug"},
						              "relationships": {"project": {}}}],
							"links": {"next": "", "previous": "", "self": ""}}`,
				},
			}},
		},
		"/resource_translations_async_downloads": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": {
						"type": "resource_translations_async_downloads",
						"id": "download_1",
						"relationships": {"resource": {"data": {
							"type": "resources",
							"id": "o:orgslug:p:projslug:r:resslug"
						}},
							"language": {"data": {"id": "l:el",
							                      "type": "languages"}}
						}
					}}`,
				},
			}},
		},
		asyncDownloadsUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Redirect: ts.URL,
				},
			}},
		},
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:    "default",
		Mode:        "default",
		Force:       true,
		All:         true,
		ResourceIds: nil,
	}

	err = PullCommand(&cfg, api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	endpoint := mockData["/organizations"]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '/organizations', expected 1",
			endpoint.Count)
	}
	actual := endpoint.Requests[0].Request
	expected := jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected %+v", actual, expected)
	}

	endpoint = mockData[projectsUrl]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '%s', expected 1",
			endpoint.Count, projectsUrl)
	}
	actual = endpoint.Requests[0].Request
	expected = jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected %+v", actual, expected)
	}

	endpoint = mockData[resourcesUrl]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '%s', expected 1",
			endpoint.Count, resourcesUrl)
	}
	actual = endpoint.Requests[0].Request
	expected = jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected %+v", actual, expected)
	}

	endpoint = mockData["/resource_translations_async_downloads"]
	if endpoint.Count != 1 {
		t.Errorf(
			"Made %d requests to '/resource_translations_async_downloads', "+
				"expected 1",
			endpoint.Count,
		)
	}
	actual = endpoint.Requests[0].Request
	if actual.Method != "POST" ||
		len(actual.Payload) == 0 {
		t.Errorf("Something was wrong with the request '%+v'", actual)
	}

	endpoint = mockData[asyncDownloadsUrl]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '%s', expected 1",
			endpoint.Count, asyncDownloadsUrl)
	}
	actual = endpoint.Requests[0].Request
	expected = jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected %+v", actual, expected)
	}

	bytes, err := os.ReadFile(filepath.Join(
		workingDir, "locale", "el", "file",
	))
	if err != nil {
		t.Errorf("%s", err)
	}
	expectedPayloadBytes := []byte(fmt.Sprintln(`Here comes the sun`))

	if !reflect.DeepEqual(bytes, expectedPayloadBytes) {
		t.Errorf("File created contains '%+v', expected '%+v'",
			bytes, expectedPayloadBytes)
	}

	// Clean up
	os.Remove(filepath.Join(workingDir, "locale", "el", "file"))
	os.Remove(filepath.Join(workingDir, "locale", "el"))
	os.Remove(filepath.Join(workingDir, "locale"))
}

func TestPullCommandDownloadSource(t *testing.T) {
	cfg := config.Config{
		Local: &config.LocalConfig{
			Resources: []config.Resource{
				{
					OrganizationSlug: "orgslug",
					ProjectSlug:      "projslug",
					ResourceSlug:     "resslug",
					Type:             "I18N_TYPE",
					SourceFile:       "source",
					LanguageMappings: map[string]string{
						"el": "el",
					},
				},
			},
		},
	}

	projectsUrl := "/projects?" +
		"filter%5Borganization%5D=o%3Aorgslug&filter%5Bslug%5D=projslug"
	resourcesUrl := "/resources?filter%5Bproject%5D=o%3Aorgslug%3Ap%3Aprojslug"
	mockData := jsonapi.MockData{
		"/organizations": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "organizations",
					                  "id": "o:orgslug",
									  "attributes": {"slug": "orgslug"}}]}`,
				},
			}},
		},
		projectsUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{
						"type": "project",
						"id": "o:orgslug:p:projslug",
						"relationships": {"organization": {}}
					}]}`,
				},
			}},
		},
		resourcesUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "resources",
						              "id": "o:orgslug:p:projslug:r:resslug",
						              "attributes": {"slug": "resslug"},
						              "relationships": {"project": {}}}]}`,
				},
			}},
		},
		"/resource_strings_async_downloads": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": {
						"type": "resource_strings_async_downloads",
						"id": "download_1",
						"relationships": {"resource": {"data": {
							"type": "resources",
							"id": "o:orgslug:p:projslug:r:resslug"
						}}}
					}}`,
				},
			}},
		},
		"/resource_strings_async_downloads/download_1": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": {"type": "resource_strings_async_downloads",
					                 "id": "download_1",
									 "attributes": {"status": "succeeded"}}}`,
				},
			}},
		},
	}

	api := jsonapi.GetTestConnection(mockData)
	err := PullCommand(
		&cfg,
		api,
		&PullCommandArguments{
			Force:       true,
			Source:      true,
			ResourceIds: []string{"projslug.resslug"},
		},
	)
	if err != nil {
		t.Errorf("%s", err)
	}

	endpoint := mockData["/organizations"]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '/organizations', expected 1",
			endpoint.Count)
	}
	actual := endpoint.Requests[0].Request
	expected := jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected %+v", actual, expected)
	}

	endpoint = mockData[projectsUrl]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '%s', expected 1",
			endpoint.Count, projectsUrl)
	}
	actual = endpoint.Requests[0].Request
	expected = jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected %+v", actual, expected)
	}

	endpoint = mockData[resourcesUrl]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '%s', expected 1",
			endpoint.Count, resourcesUrl)
	}
	actual = endpoint.Requests[0].Request
	expected = jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected %+v", actual, expected)
	}

	endpoint = mockData["/resource_strings_async_downloads"]
	if endpoint.Count != 1 {
		t.Errorf(
			"Made %d requests to '/resource_strings_async_downloads', "+
				"expected 1",
			endpoint.Count,
		)
	}
	actual = endpoint.Requests[0].Request
	if actual.Method != "POST" ||
		len(actual.Payload) == 0 {
		t.Errorf("Something was wrong with the request '%+v'", actual)
	}

	endpoint = mockData["/resource_strings_async_downloads/download_1"]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '%s', expected 1",
			endpoint.Count, "/resource_strings_async_downloads/download_1")
	}
	actual = endpoint.Requests[0].Request
	expected = jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected %+v", actual, expected)
	}
}

func createTestServer() *httptest.Server {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Here comes the sun")
		},
	))
	ts.EnableHTTP2 = true

	return ts
}
