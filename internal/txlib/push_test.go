package txlib

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
)

func TestPushCommandResourceExists(t *testing.T) {
	afterTest := beforePushTest(t, nil, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/organizations":                  getOrganizationEndpoint(),
		projectsUrl:                       getProjectsEndpoint(),
		resourcesUrl:                      getResourcesEndpoint(),
		"/resource_strings_async_uploads": getSourceUploadPostEndpoint(),
		"/resource_strings_async_uploads/upload_1": getSourceUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Force: true, Branch: "-1",
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	testSimpleUpload(t, mockData, "/resource_strings_async_uploads")
	testSimpleGet(t, mockData, "/resource_strings_async_uploads/upload_1")
}

func TestPushSpecificResource(t *testing.T) {
	afterTest := beforePushTest(t, nil, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/organizations":                  getOrganizationEndpoint(),
		projectsUrl:                       getProjectsEndpoint(),
		resourcesUrl:                      getResourcesEndpoint(),
		"/resource_strings_async_uploads": getSourceUploadPostEndpoint(),
		"/resource_strings_async_uploads/upload_1": getSourceUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Force:       true,
		ResourceIds: []string{"projslug.resslug"},
		Branch:      "-1",
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	testSimpleUpload(t, mockData, "/resource_strings_async_uploads")
	testSimpleGet(t, mockData, "/resource_strings_async_uploads/upload_1")
}

func TestPushCommandResourceDoesNotExist(t *testing.T) {
	afterTest := beforePushTest(t, nil, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/organizations":                  getOrganizationEndpoint(),
		projectsUrl:                       getProjectsEndpoint(),
		resourcesUrl:                      getEmptyEndpoint(),
		"/resources":                      getResourceEndpoint(),
		"/resource_strings_async_uploads": getSourceUploadPostEndpoint(),
		"/resource_strings_async_uploads/upload_1": getSourceUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Force:  true,
		Branch: "-1",
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimplePost(t, mockData, "/resources", `{"data": {
		"type": "resources",
		"attributes": {"name": "aaa.json", "slug": "resslug"},
		"relationships": {
			"project": {"data": {"type": "projects",
								 "id": "o:orgslug:p:projslug"}},
			"i18n_format": {"data": {"type": "i18n_formats",
									 "id": "I18N_TYPE"}}
		}
	}}`)
	testSimpleUpload(t, mockData, "/resource_strings_async_uploads")
	testSimpleGet(t, mockData, "/resource_strings_async_uploads/upload_1")
}

func TestPushTranslation(t *testing.T) {
	afterTest := beforePushTest(t, []string{"fr"}, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/organizations": getOrganizationEndpoint(),
		projectsUrl:      getProjectsEndpoint(),
		resourcesUrl:     getResourcesEndpoint(),
		"/projects/o:orgslug:p:projslug/languages": getLanguagesEndpoint(
			[]string{"fr"},
		),
		uploadsUrl: getTranslationUploadPostEndpoint(),
		uploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Translation: true,
		Force:       true,
		Branch:      "-1",
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	testSimpleGet(t, mockData, "/projects/o:orgslug:p:projslug/languages")
	testSimpleUpload(t, mockData, uploadsUrl)
	testSimpleGet(t, mockData, uploadUrl)
}

func TestPushXliff(t *testing.T) {
	afterTest := beforePushTest(t, nil, nil)
	defer afterTest()

	file, err := os.OpenFile("aaa-fr.json.xlf",
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		t.Error(err)
	}
	_, err = file.WriteString("hello world")
	if err != nil {
		t.Error(err)
	}
	err = file.Close()
	if err != nil {
		t.Error(err)
	}

	mockData := jsonapi.MockData{
		"/organizations": getOrganizationEndpoint(),
		projectsUrl:      getProjectsEndpoint(),
		resourcesUrl:     getResourcesEndpoint(),
		"/projects/o:orgslug:p:projslug/languages": getLanguagesEndpoint(
			[]string{"fr"},
		),
		uploadsUrl: getTranslationUploadPostEndpoint(),
		uploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err = PushCommand(getStandardConfig(), api, PushCommandArguments{
		Translation: true,
		Force:       true,
		Xliff:       true,
		Branch:      "-1",
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	testSimpleGet(t, mockData, "/projects/o:orgslug:p:projslug/languages")
	testSimpleUpload(t, mockData, uploadsUrl)
	testSimpleGet(t, mockData, uploadUrl)
}

func TestPushTranslationWithLanguageMapping(t *testing.T) {
	afterTest := beforePushTest(t, []string{"froutzes"}, nil)
	defer afterTest()

	cfg := getStandardConfig()
	cfg.Local.Resources[0].LanguageMappings = map[string]string{
		"fr": "froutzes",
	}

	mockData := jsonapi.MockData{
		"/organizations": getOrganizationEndpoint(),
		projectsUrl:      getProjectsEndpoint(),
		resourcesUrl:     getResourcesEndpoint(),
		"/projects/o:orgslug:p:projslug/languages": getLanguagesEndpoint(
			[]string{"fr"},
		),
		uploadsUrl: getTranslationUploadPostEndpoint(),
		uploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(cfg, api, PushCommandArguments{
		Translation: true,
		Force:       true,
		Branch:      "-1",
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	testSimpleGet(t, mockData, "/projects/o:orgslug:p:projslug/languages")
	testSimpleUpload(t, mockData, uploadsUrl)
	testSimpleGet(t, mockData, uploadUrl)
}

func TestPushTranslationWithOverrides(t *testing.T) {
	afterTest := beforePushTest(t, []string{"el"},
		[]string{"source.json"},
	)
	defer afterTest()

	cfg := getStandardConfig()
	cfg.Local.Resources[0].Overrides = map[string]string{
		"fr": "source.json",
	}

	mockData := jsonapi.MockData{
		"/organizations": getOrganizationEndpoint(),
		projectsUrl:      getProjectsEndpoint(),
		resourcesUrl:     getResourcesEndpoint(),
		"/projects/o:orgslug:p:projslug/languages": getLanguagesEndpoint(
			[]string{"fr"},
		),
		uploadsUrl: getTranslationUploadPostEndpoint(),
		uploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(cfg, api, PushCommandArguments{
		Translation: true,
		Force:       true,
		Branch:      "-1",
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	testSimpleGet(t, mockData, "/projects/o:orgslug:p:projslug/languages")
	testSimpleUpload(t, mockData, uploadsUrl)
	testSimpleGet(t, mockData, uploadUrl)
}

func TestPushTranslationRemoteLanguageDoesNotExist(t *testing.T) {
	afterTest := beforePushTest(t, []string{"el"}, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/organizations": getOrganizationEndpoint(),
		projectsUrl:      getProjectsEndpoint(),
		resourcesUrl:     getResourcesEndpoint(),
		"/projects/o:orgslug:p:projslug/languages": getLanguagesEndpoint(nil),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Translation: true,
		Force:       true,
		Branch:      "-1",
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	testSimpleGet(t, mockData, "/projects/o:orgslug:p:projslug/languages")
}

func TestPushTranslationLocalFileIsOlderThanRemote(t *testing.T) {
	afterTest := beforePushTest(t, []string{"fr"}, nil)
	defer afterTest()

	now := time.Now().UTC()
	duration, _ := time.ParseDuration("5m")
	languagesUrl := "/projects/o:orgslug:p:projslug/languages"
	mockData := jsonapi.MockData{
		"/organizations": getOrganizationEndpoint(),
		projectsUrl:      getProjectsEndpoint(),
		resourcesUrl:     getResourcesEndpoint(),
		resourceLanguageStatsUrl: getResourceLanguageStatsEndpoint(
			now.Add(duration),
		),
		languagesUrl: getLanguagesEndpoint([]string{"fr"}),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Translation: true,
		Branch:      "-1",
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	testSimpleGet(t, mockData, resourceLanguageStatsUrl)
	testSimpleGet(t, mockData, "/projects/o:orgslug:p:projslug/languages")
}

func TestPushTranslationLocalFileIsNewerThanRemote(t *testing.T) {
	afterTest := beforePushTest(t, []string{"fr"}, nil)
	defer afterTest()

	now := time.Now().UTC()
	duration, _ := time.ParseDuration("-5m")
	mockData := jsonapi.MockData{
		"/organizations":         getOrganizationEndpoint(),
		projectsUrl:              getProjectsEndpoint(),
		resourcesUrl:             getResourcesEndpoint(),
		resourceLanguageStatsUrl: getResourceLanguageStatsEndpoint(now.Add(duration)),
		"/projects/o:orgslug:p:projslug/languages": getLanguagesEndpoint(
			[]string{"fr"},
		),
		uploadsUrl: getTranslationUploadPostEndpoint(),
		uploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Translation: true,
		Branch:      "-1",
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	testSimpleGet(t, mockData, resourceLanguageStatsUrl)
	testSimpleGet(t, mockData, "/projects/o:orgslug:p:projslug/languages")
	testSimpleUpload(t, mockData, uploadsUrl)
	testSimpleGet(t, mockData, uploadUrl)
}

func TestPushTranslationLimitLanguages(t *testing.T) {
	afterTest := beforePushTest(t, []string{"el", "fr"}, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/organizations": getOrganizationEndpoint(),
		projectsUrl:      getProjectsEndpoint(),
		resourcesUrl:     getResourcesEndpoint(),
		"/projects/o:orgslug:p:projslug/languages": getLanguagesEndpoint(
			[]string{"el", "fr"},
		),
		uploadsUrl: getTranslationUploadPostEndpoint(),
		uploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Translation: true,
		Force:       true,
		Languages:   []string{"fr"},
		Branch:      "-1",
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	testSimpleGet(t, mockData, "/projects/o:orgslug:p:projslug/languages")
	testSimpleUpload(t, mockData, uploadsUrl)
	testSimpleGet(t, mockData, uploadUrl)
}

func TestOrganizationNotFound(t *testing.T) {
	afterTest := beforePushTest(t, nil, nil)
	defer afterTest()
	mockData := jsonapi.MockData{
		"/organizations": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{Text: `{"data": []}`},
			}},
		},
	}
	api := jsonapi.GetTestConnection(mockData)
	err := PushCommand(
		getStandardConfig(), api, PushCommandArguments{Branch: "-1"},
	)
	if err == nil {
		t.Error("Expected error")
	}
	expected := "Fetching organization 'orgslug': Not found"
	if err != nil && err.Error() != expected {
		t.Errorf("Got error message '%s', expected '%s'",
			err, expected)
	}

	testSimpleGet(t, mockData, "/organizations")
}

func TestProjectNotFound(t *testing.T) {
	afterTest := beforePushTest(t, nil, nil)
	defer afterTest()
	mockData := jsonapi.MockData{
		"/organizations": getOrganizationEndpoint(),
		projectsUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{Text: `{"data": []}`},
			}},
		},
	}
	api := jsonapi.GetTestConnection(mockData)
	err := PushCommand(
		getStandardConfig(), api, PushCommandArguments{Branch: "-1"},
	)
	if err == nil {
		t.Error("Expected error")
	}
	expected := "Fetching project 'projslug': Not found"
	if err != nil && err.Error() != expected {
		t.Errorf("Got error message '%s', expected '%s'",
			err, expected)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
}

func TestLocalFileNotFound(t *testing.T) {
	mockData := jsonapi.MockData{
		"/organizations": getOrganizationEndpoint(),
		projectsUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{Text: `{"data": []}`},
			}},
		},
	}
	api := jsonapi.GetTestConnection(mockData)
	err := PushCommand(
		getStandardConfig(), api, PushCommandArguments{Branch: "-1"},
	)
	if err == nil {
		t.Error("Expected error")
	}
	expected := "could not find file 'aaa.json'. Aborting."
	if err != nil && err.Error() != expected {
		t.Errorf("Got error message '%s', expected '%s'",
			err, expected)
	}
}

func TestPushCommandBranch(t *testing.T) {
	afterTest := beforePushTest(t, nil, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/organizations": getOrganizationEndpoint(),
		projectsUrl:      getProjectsEndpoint(),
		resourcesUrl:     getResourcesEndpoint(),
		"/resources": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": {
						"type": "resources",
						"id": "o:orgslug:p:projslug:r:branch--resslug",
						"attributes": {"slug": "branch--resslug"}
					}}`,
				},
			}},
		},
		"/resource_strings_async_uploads":          getSourceUploadPostEndpoint(),
		"/resource_strings_async_uploads/upload_1": getSourceUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Force:  true,
		Branch: "branch",
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	testSimplePost(t, mockData, "/resources", `{"data": {
		"type": "resources",
		"attributes": {"name": "(branch branch) aaa.json",
					   "slug": "branch--resslug"},
		"relationships": {
			"project": {"data": {"type": "projects",
								 "id": "o:orgslug:p:projslug"}},
			"i18n_format": {"data": {"type": "i18n_formats",
									 "id": "I18N_TYPE"}}
		}
	}}`)
	testSimpleUpload(t, mockData, "/resource_strings_async_uploads")
	testSimpleGet(t, mockData, "/resource_strings_async_uploads/upload_1")
}

func TestPushNewLanguage(t *testing.T) {
	afterTest := beforePushTest(t, []string{"fr"}, nil)
	defer afterTest()

	languagesUrl := "/projects/o:orgslug:p:projslug/relationships/languages"
	mockData := jsonapi.MockData{
		"/organizations": getOrganizationEndpoint(),
		projectsUrl:      getProjectsEndpoint(),
		resourcesUrl:     getResourcesEndpoint(),
		"/projects/o:orgslug:p:projslug/languages": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{
				{Response: jsonapi.MockResponse{Text: `{"data": []}`}},
				{Response: jsonapi.MockResponse{Text: `{"data": [{
					"type": "languages",
					"id": "l:fr",
					"attributes": {"code": "fr"}
				}]}`}},
			},
		},
		"/languages": getLanguagesEndpoint([]string{"fr"}),
		"/languages/l:en": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": {"type": "languages",
					                 "id": "l:en",
									 "attributes": {"code": "en"}}}`,
				},
			}},
		},
		languagesUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: "",
				},
			}},
		},
		uploadsUrl: getTranslationUploadPostEndpoint(),
		uploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Branch:      "-1",
		Force:       true,
		Translation: true,
		Languages:   []string{"fr"},
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	endpoint := mockData["/projects/o:orgslug:p:projslug/languages"]
	if endpoint.Count != 2 {
		t.Errorf(
			"Got %d requests to '/projects/o:orgslug:p:projslug/languages', "+
				"expected 2",
			endpoint.Count,
		)
	}
	for i := 0; i < 2; i++ {
		actual := endpoint.Requests[i].Request
		expected := jsonapi.CapturedRequest{Method: "GET"}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Got request '%+v', expected '%+v'", actual, expected)
		}
	}
	testSimpleGet(t, mockData, "/languages")
	testSimpleGet(t, mockData, "/languages/l:en")
	testSimplePost(t, mockData,
		"/projects/o:orgslug:p:projslug/relationships/languages",
		`{"data": [{"type": "languages", "id": "l:fr"}]}`)
	testSimpleUpload(t, mockData, uploadsUrl)
	testSimpleGet(t, mockData, uploadUrl)
}

func TestPushAll(t *testing.T) {
	afterTest := beforePushTest(t, []string{"fr"}, nil)
	defer afterTest()

	languagesUrl := "/projects/o:orgslug:p:projslug/relationships/languages"
	mockData := jsonapi.MockData{
		"/organizations": getOrganizationEndpoint(),
		projectsUrl:      getProjectsEndpoint(),
		resourcesUrl:     getResourcesEndpoint(),
		"/projects/o:orgslug:p:projslug/languages": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{
				{Response: jsonapi.MockResponse{Text: `{"data": []}`}},
				{Response: jsonapi.MockResponse{Text: `{"data": [{
					"type": "languages",
					"id": "l:fr",
					"attributes": {"code": "fr"}
				}]}`}},
			},
		},
		"/languages": getLanguagesEndpoint([]string{"fr"}),
		"/languages/l:en": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: `{"data": {"type": "languages",
					                 "id": "l:en",
									 "attributes": {"code": "en"}}}`,
				},
			}},
		},
		languagesUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{{
				Response: jsonapi.MockResponse{
					Text: "",
				},
			}},
		},
		uploadsUrl: getTranslationUploadPostEndpoint(),
		uploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Branch:      "-1",
		Force:       true,
		Translation: true,
		All:         true,
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, "/organizations")
	testSimpleGet(t, mockData, projectsUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	endpoint := mockData["/projects/o:orgslug:p:projslug/languages"]
	if endpoint.Count != 2 {
		t.Errorf(
			"Got %d requests to '/projects/o:orgslug:p:projslug/languages', "+
				"expected 2",
			endpoint.Count,
		)
	}
	for i := 0; i < 2; i++ {
		actual := endpoint.Requests[i].Request
		expected := jsonapi.CapturedRequest{Method: "GET"}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Got request '%+v', expected '%+v'", actual, expected)
		}
	}
	testSimpleGet(t, mockData, "/languages")
	testSimpleGet(t, mockData, "/languages/l:en")
	testSimplePost(t, mockData,
		"/projects/o:orgslug:p:projslug/relationships/languages",
		`{"data": [{"type": "languages", "id": "l:fr"}]}`)
	testSimpleUpload(t, mockData, uploadsUrl)
	testSimpleGet(t, mockData, uploadUrl)
}

const (
	projectsUrl = "/projects?" +
		"filter%5Borganization%5D=o%3Aorgslug&filter%5Bslug%5D=projslug"
	resourcesUrl = "/resources?filter%5B" +
		"project%5D=o%3Aorgslug%3Ap%3Aprojslug"
	resourceLanguageStatsUrl = "/resource_language_stats?" +
		"filter%5Bproject%5D=o%3Aorgslug%3Ap%3Aprojslug&" +
		"filter%5Bresource%5D=o%3Aorgslug%3Ap%3Aprojslug%3Ar%3Aresslug"
	uploadsUrl = "/resource_translations_async_uploads"
	uploadUrl  = "/resource_translations_async_uploads/upload1"
)

func beforePushTest(t *testing.T,
	languageCodes []string,
	customFiles []string) func() {
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

func getOrganizationEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: `{"data": [{"type": "organizations",
				                  "id": "o:orgslug",
								  "attributes": {"slug": "orgslug"}}]}`,
			},
		}},
	}
}

func getProjectsEndpoint() *jsonapi.MockEndpoint {
	selfUrl := "/projects/o:orgslug:p:projslug/relationships/languages"
	relatedUrl := "/projects/o:orgslug:p:projslug/languages"
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: fmt.Sprintf(`{"data": [{
					"type": "projects",
					"id": "o:orgslug:p:projslug",
					"attributes": {"slug": "projslug"},
					"relationships": {
						"languages": {
							"links": {
								"self": "%s",
								"related": "%s"
							}
						},
						"source_language": {
							"data": {"type": "languages", "id": "l:en"},
							"links": {"related": "/languages/l:en"}
						}
					}
				}]}`, selfUrl, relatedUrl),
			},
		}},
	}
}

func getResourceEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: `{"data": {"type": "resources",
								 "id": "o:orgslug:p:projslug:r:resslug",
								 "attributes": {"slug": "resslug"}}}`,
			},
		}},
	}
}

func getResourcesEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: `{"data": [{"type": "resources",
								  "id": "o:orgslug:p:projslug:r:resslug",
								  "attributes": {"slug": "resslug"}}]}`,
			},
		}},
	}
}

func getEmptyEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{Text: `{"data": []}`},
		}},
	}
}

func getSourceUploadPostEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: `{"data": {
					"type": "resource_strings_async_uploads",
					"id": "upload_1",
					"relationships": {"resource": {"data": {
						"type": "resources",
						"id": "o:orgslug:p:projslug:r:resslug"
					}}}
				}}`,
			},
		}},
	}
}

func getSourceUploadGetEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: `{"data": {"type": "resource_strings_async_uploads",
								 "id": "upload_1",
								 "attributes": {"status": "succeeded"}}}`,
			},
		}},
	}
}

func getTranslationUploadPostEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: `{"data": {
						"type": "resource_translations_async_uploads",
						"id": "upload1",
						"relationships": {
							"resource": {"data": {
								"type": "resources",
								"id": "o:orgslug:p:projslug:r:resslug"
							}},
							"language": {"data": {"type": "languages",
							                      "id": "l:fr"}}
						}
					}}`,
			},
		}},
	}
}

func getTranslationUploadGetEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: `{"data": {
					"type": "resource_translations_async_uploads",
					"id": "upload1",
					"attributes": {"status": "succeeded"},
					"relationships": {
						"resource": {"data": {
							"type": "resources",
							"id": "o:orgslug:p:projslug:r:resslug"
						}},
						"language": {"data": {"type": "languages",
						                      "id": "l:el"}}
					}
				}}`,
			},
		}},
	}
}

func getLanguagesEndpoint(codes []string) *jsonapi.MockEndpoint {
	var result []string
	for _, code := range codes {
		result = append(result,
			fmt.Sprintf(`{"type": "languages",
						  "id": "l:%s",
						  "attributes": {"code": "%s"}}`,
				code, code))
	}
	text := fmt.Sprintf(`{"data": [%s]}`, strings.Join(result, ", "))
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{Text: text},
		}},
	}
}

func getResourceLanguageStatsEndpoint(
	timestamp time.Time,
) *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: fmt.Sprintf(
					`{"data": [{
						"type": "resource_language_stats",
						"id":"stats1",
						"attributes": {"last_update": "%s"},
						"relationships": {
							"language": {"data": {"type": "languages",
												  "id": "l:fr"}},
							"resource": {}
						}
					}]}`,
					timestamp.Format(time.RFC3339),
				),
			},
		}},
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

func testSimpleUpload(t *testing.T, mockData jsonapi.MockData, path string) {
	endpoint := mockData[path]
	if endpoint.Count != 1 {
		t.Errorf("Made %d requests to '%s', expected 1", endpoint.Count, path)
	}
	actual := endpoint.Requests[0].Request
	if actual.Method != "POST" ||
		len(actual.Payload) == 0 ||
		!strings.HasPrefix(actual.ContentType, "multipart/form-data") {
		t.Errorf("Something was wrong with the request '%+v'", actual)
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
