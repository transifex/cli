package txlib

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
)

var (
	organizationId       = "o:orgslug"
	projectId            = fmt.Sprintf("%s:p:projslug", organizationId)
	projectUrl           = fmt.Sprintf("/projects/%s", projectId)
	resourceId           = fmt.Sprintf("%s:r:resslug", projectId)
	resourceUrl          = fmt.Sprintf("/resources/%s", resourceId)
	statsUrlAllLanguages = fmt.Sprintf(
		"/resource_language_stats?%s=%s&%s=%s",
		url.QueryEscape("filter[project]"),
		url.QueryEscape(projectId),
		url.QueryEscape("filter[resource]"),
		url.QueryEscape(resourceId),
	)
	statsUrlSourceLanguage = fmt.Sprintf(
		"/resource_language_stats?%s=%s&%s=%s&%s=%s",
		url.QueryEscape("filter[language]"),
		url.QueryEscape("l:en"),
		url.QueryEscape("filter[project]"),
		url.QueryEscape(projectId),
		url.QueryEscape("filter[resource]"),
		url.QueryEscape(resourceId),
	)
	resourcesUrl            = "/resources"
	translationUploadsUrl   = "/resource_translations_async_uploads"
	translationUploadUrl    = fmt.Sprintf("%s/upload_1", translationUploadsUrl)
	sourceUploadsUrl        = "/resource_strings_async_uploads"
	sourceUploadUrl         = fmt.Sprintf("%s/upload_1", sourceUploadsUrl)
	translationDownloadsUrl = "/resource_translations_async_downloads"
	translationDownloadUrl  = fmt.Sprintf("%s/download_1", translationDownloadsUrl)
	sourceDownloadsUrl      = "/resource_strings_async_downloads"
	sourceDownloadUrl       = fmt.Sprintf("%s/download_1", sourceDownloadsUrl)
)

func beforeTest(
	t *testing.T,
	languageCodes []string,
	customFiles []string,
) func() {
	curDir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error(err)
	}
	err = os.Chdir(tempDir)
	if err != nil {
		t.Error(err)
	}

	file, err := os.OpenFile("aaa.json",
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0755)
	if err != nil {
		t.Error(err)
	}
	defer file.Close()
	for _, languageCode := range languageCodes {
		file, err = os.OpenFile(
			fmt.Sprintf("aaa-%s.json", languageCode),
			os.O_RDWR|os.O_CREATE|os.O_TRUNC,
			0755,
		)
		if err != nil {
			t.Error(err)
		}
		_, err = file.WriteString(`{"hello": "world"}`)
		if err != nil {
			t.Error(err)
		}
		defer file.Close()
	}

	for _, customFile := range customFiles {
		file, err = os.OpenFile(
			customFile,
			os.O_RDWR|os.O_CREATE|os.O_TRUNC,
			0755,
		)
		if err != nil {
			t.Error(err)
		}
		_, err = file.WriteString(`{"hello": "world"}`)
		if err != nil {
			t.Error(err)
		}
		defer file.Close()
	}

	return func() {
		err := os.Chdir(curDir)
		if err != nil {
			t.Error(err)
		}
		os.RemoveAll(tempDir)
	}
}

func testSimpleGet(t *testing.T, mockData jsonapi.MockData, path string) {
	endpoint := mockData[path]
	if endpoint.Count != 1 {
		t.Errorf("Got %d requests to '%s', expected 1", endpoint.Count, path)
	}
	actual := endpoint.Requests[0].Request
	expected := jsonapi.CapturedRequest{Method: "GET"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected '%+v'", actual, expected)
	}
}

func testSimplePost(
	t *testing.T, mockData jsonapi.MockData, path, payload string,
) {
	endpoint := mockData[path]
	if endpoint.Count != 1 {
		t.Errorf("Got %d requests to '%s', expected 1", endpoint.Count, path)
	}
	actual := endpoint.Requests[0].Request
	if actual.Method != "POST" || actual.ContentType != "" {
		t.Errorf("Got wrong request %+v", actual)
	}
	var actualPayload interface{}
	err := json.Unmarshal(actual.Payload, &actualPayload)
	if err != nil {
		t.Error(err)
	}
	var expectedPayload interface{}
	err = json.Unmarshal([]byte(payload), &expectedPayload)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(actualPayload, expectedPayload) {
		t.Errorf("Got paylod '%+v', expected '%+v'",
			actualPayload, expectedPayload)
	}
}

func getStandardConfig() *config.Config {
	return &config.Config{
		Local: &config.LocalConfig{
			Resources: []config.Resource{{
				OrganizationSlug: "orgslug",
				ProjectSlug:      "projslug",
				ResourceSlug:     "resslug",
				Type:             "I18N_TYPE",
				SourceFile:       "aaa.json",
				FileFilter:       "aaa-<lang>.json",
			}},
		},
	}
}

func getNewTestServer(output string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, output)
		},
	))
}

func getResourceEndpoint() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(
		`{"data": {"type": "resources",
							 "id": "o:orgslug:p:projslug:r:resslug",
							 "attributes": {"slug": "resslug"},
							 "relationships": {"project": {"data": {"type": "projects",
																											"id": "o:orgslug:p:projslug"}}}}}`,
	)
}

func getProjectEndpoint() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(`{"data": {
			"type": "projects",
			"id": "o:orgslug:p:projslug",
			"attributes": {"slug": "projslug"},
			"relationships": {
				"languages": {"links": {
					"self": "/projects/o:orgslug:p:projslug/relationships/languages",
					"related": "/projects/o:orgslug:p:projslug/languages"
				}},
				"source_language": {"data": {"type": "languages", "id": "l:en"},
				                    "links": {"related": "/languages/l:en"}}
			}
		}}`,
	)
}

func getStatsEndpointAllLanguages() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(fmt.Sprintf(
		`{"data": [{"type": "resource_language_stats",
			            "id": "%s:l:en",
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:en"}}}},
			           {"type": "resource_language_stats",
					    "id": "%s:l:el",
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:el"}}}}]}`,
		resourceId,
		resourceId,
	))
}

func getStatsEndpointSourceLanguage() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(fmt.Sprintf(
		`{"data": [{"type": "resource_language_stats",
			            "id": "%s:l:en",
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:en"}}}}]}`,
		resourceId,
	))
}
