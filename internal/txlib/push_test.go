package txlib

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/transifex/cli/pkg/jsonapi"
)

func TestPushCommandResourceExists(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/languages":                      getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:                       getResourceEndpoint(),
		projectUrl:                        getProjectEndpoint(),
		statsUrlSourceLanguage:            getStatsEndpointSourceLanguage(),
		"/resource_strings_async_uploads": getSourceUploadPostEndpoint(),
		"/resource_strings_async_uploads/upload_1": getSourceUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Force: true, Branch: "-1", Workers: 1,
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlSourceLanguage)
	testSimpleUpload(t, mockData, "/resource_strings_async_uploads")
	testSimpleGet(t, mockData, "/resource_strings_async_uploads/upload_1")
}

func TestPushSpecificResource(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/languages":           getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:            getResourceEndpoint(),
		projectUrl:             getProjectEndpoint(),
		statsUrlSourceLanguage: getStatsEndpointSourceLanguage(),
		sourceUploadsUrl:       getSourceUploadPostEndpoint(),
		sourceUploadUrl:        getSourceUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Force:       true,
		ResourceIds: []string{"projslug.resslug"},
		Branch:      "-1",
		Workers:     1,
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlSourceLanguage)
	testSimpleUpload(t, mockData, "/resource_strings_async_uploads")
	testSimpleGet(t, mockData, "/resource_strings_async_uploads/upload_1")
}

func TestPushCommandResourceDoesNotExist(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/languages":           getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:            getEmptyEndpoint(),
		projectUrl:             getProjectEndpoint(),
		statsUrlSourceLanguage: getStatsEndpointSourceLanguage(),
		resourcesUrl:           getResourceCreatedEndpoint(),
		sourceUploadsUrl:       getSourceUploadPostEndpoint(),
		sourceUploadUrl:        getSourceUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Force:   true,
		Branch:  "-1",
		Workers: 1,
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlSourceLanguage)
	testSimplePost(
		t,
		mockData,
		resourcesUrl,
		`{"data": {
			"type": "resources",
			"attributes": {"name": "aaa.json", "slug": "resslug"},
			"relationships": {
				"project": {"data": {"type": "projects", "id": "o:orgslug:p:projslug"}},
				"i18n_format": {"data": {"type": "i18n_formats", "id": "I18N_TYPE"}}
			}
		}}`,
	)
	testSimpleUpload(t, mockData, "/resource_strings_async_uploads")
	testSimpleGet(t, mockData, "/resource_strings_async_uploads/upload_1")
}

func TestPushCommandBranchResourceDoesNotExist(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	branchResourceId := "o:orgslug:p:projslug:r:branch--resslug"
	branchResourceUrl := fmt.Sprintf("/resources/%s", branchResourceId)

	mockData := jsonapi.MockData{
		"/languages":           getLanguagesEndpoint([]string{"en", "fr", "el"}),
		branchResourceUrl:      getEmptyEndpoint(),
		resourceUrl:            getEmptyEndpoint(),
		projectUrl:             getProjectEndpoint(),
		statsUrlSourceLanguage: getStatsEndpointSourceLanguage(),
		resourcesUrl:           getResourceCreatedEndpoint(),
		sourceUploadsUrl:       getSourceUploadPostEndpoint(),
		sourceUploadUrl:        getSourceUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Force:   true,
		Branch:  "branch",
		Base:    "-1",
		Workers: 1,
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, branchResourceUrl)
	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlSourceLanguage)
	testSimplePost(
		t,
		mockData,
		resourcesUrl,
		`{"data": {
			"type": "resources",
			"attributes": {"name": "aaa.json (branch branch)", "slug": "branch--resslug"},
			"relationships": {
				"project": {"data": {"type": "projects", "id": "o:orgslug:p:projslug"}},
				"i18n_format": {"data": {"type": "i18n_formats", "id": "I18N_TYPE"}}
			}
		}}`,
	)
	testSimpleUpload(t, mockData, "/resource_strings_async_uploads")
	testSimpleGet(t, mockData, "/resource_strings_async_uploads/upload_1")
}

func TestPushTranslation(t *testing.T) {
	afterTest := beforeTest(t, []string{"el"}, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/languages":          getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:           getResourceEndpoint(),
		projectUrl:            getProjectEndpoint(),
		statsUrlAllLanguages:  getStatsEndpointAllLanguages(),
		translationUploadsUrl: getTranslationUploadPostEndpoint(),
		translationUploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Translation: true,
		Force:       true,
		Branch:      "-1",
		Workers:     1,
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleUpload(t, mockData, translationUploadsUrl)
	testSimpleGet(t, mockData, translationUploadUrl)
}

func TestPushXliff(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	file, err := os.OpenFile("aaa-el.json.xlf",
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
		"/languages":          getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:           getResourceEndpoint(),
		projectUrl:            getProjectEndpoint(),
		statsUrlAllLanguages:  getStatsEndpointAllLanguages(),
		translationUploadsUrl: getTranslationUploadPostEndpoint(),
		translationUploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err = PushCommand(getStandardConfig(), api, PushCommandArguments{
		Translation: true,
		Force:       true,
		Xliff:       true,
		Branch:      "-1",
		Workers:     1,
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleUpload(t, mockData, translationUploadsUrl)
	testSimpleGet(t, mockData, translationUploadUrl)
}

func TestPushTranslationWithLanguageMapping(t *testing.T) {
	afterTest := beforeTest(t, []string{"froutzes"}, nil)
	defer afterTest()

	cfg := getStandardConfig()
	cfg.Local.Resources[0].LanguageMappings = map[string]string{
		"el": "froutzes",
	}

	mockData := jsonapi.MockData{
		"/languages":          getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:           getResourceEndpoint(),
		projectUrl:            getProjectEndpoint(),
		statsUrlAllLanguages:  getStatsEndpointAllLanguages(),
		translationUploadsUrl: getTranslationUploadPostEndpoint(),
		translationUploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(cfg, api, PushCommandArguments{
		Translation: true,
		Force:       true,
		Branch:      "-1",
		Workers:     1,
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleUpload(t, mockData, translationUploadsUrl)
	testSimpleGet(t, mockData, translationUploadUrl)
}

func TestPushTranslationWithOverrides(t *testing.T) {
	afterTest := beforeTest(t, []string{"el"},
		[]string{"source.json"},
	)
	defer afterTest()

	cfg := getStandardConfig()
	cfg.Local.Resources[0].Overrides = map[string]string{
		"el": "source.json",
	}

	mockData := jsonapi.MockData{
		"/languages":          getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:           getResourceEndpoint(),
		projectUrl:            getProjectEndpoint(),
		statsUrlAllLanguages:  getStatsEndpointAllLanguages(),
		translationUploadsUrl: getTranslationUploadPostEndpoint(),
		translationUploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(cfg, api, PushCommandArguments{
		Translation: true,
		Force:       true,
		Branch:      "-1",
		Workers:     1,
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleUpload(t, mockData, translationUploadsUrl)
	testSimpleGet(t, mockData, translationUploadUrl)
}

func TestPushTranslationRemoteLanguageDoesNotExist(t *testing.T) {
	afterTest := beforeTest(t, []string{"fr"}, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/languages":         getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:          getResourceEndpoint(),
		projectUrl:           getProjectEndpoint(),
		statsUrlAllLanguages: getStatsEndpointAllLanguages(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Translation: true,
		Force:       true,
		Branch:      "-1",
		Workers:     1,
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
}

func TestPushTranslationLocalFileIsOlderThanRemote(t *testing.T) {
	afterTest := beforeTest(t, []string{"fr"}, nil)
	defer afterTest()

	now := time.Now().UTC()
	duration, _ := time.ParseDuration("5m")
	mockData := jsonapi.MockData{
		"/languages": getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:  getResourceEndpoint(),
		projectUrl:   getProjectEndpoint(),
		statsUrlAllLanguages: getResourceLanguageStatsEndpoint(
			now.Add(duration),
		),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Translation: true,
		Branch:      "-1",
		Workers:     1,
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
}

func TestPushTranslationLocalFileIsNewerThanRemote(t *testing.T) {
	afterTest := beforeTest(t, []string{"fr"}, nil)
	defer afterTest()

	now := time.Now().UTC()
	duration, _ := time.ParseDuration("-5m")
	mockData := jsonapi.MockData{
		"/languages":          getLanguagesEndpoint([]string{"en", "fr", "el"}),
		projectUrl:            getProjectEndpoint(),
		resourceUrl:           getResourceEndpoint(),
		statsUrlAllLanguages:  getResourceLanguageStatsEndpoint(now.Add(duration)),
		translationUploadsUrl: getTranslationUploadPostEndpoint(),
		translationUploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Translation: true,
		Branch:      "-1",
		Workers:     1,
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleUpload(t, mockData, translationUploadsUrl)
	testSimpleGet(t, mockData, translationUploadUrl)
}

func TestPushTranslationLimitLanguages(t *testing.T) {
	afterTest := beforeTest(t, []string{"el", "fr"}, nil)
	defer afterTest()

	mockData := jsonapi.MockData{
		"/languages":          getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:           getResourceEndpoint(),
		projectUrl:            getProjectEndpoint(),
		statsUrlAllLanguages:  getStatsEndpointAllLanguages(),
		translationUploadsUrl: getTranslationUploadPostEndpoint(),
		translationUploadUrl:  getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Translation: true,
		Force:       true,
		Languages:   []string{"el"},
		Branch:      "-1",
		Workers:     1,
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleUpload(t, mockData, translationUploadsUrl)
	testSimpleGet(t, mockData, translationUploadUrl)
}

func TestProjectNotFound(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()
	mockData := jsonapi.MockData{
		"/languages": getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:  getResourceEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)
	err := PushCommand(
		getStandardConfig(),
		api,
		PushCommandArguments{Branch: "-1", Workers: 1},
	)
	if err == nil {
		t.Error("Expected error")
	}

	testSimpleGet(t, mockData, resourceUrl)
}

func TestPushCommandBranch(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	resourceId := "o:orgslug:p:projslug:r:branch--resslug"
	resourceUrl := fmt.Sprintf("/resources/%s", resourceId)
	statsUrl := fmt.Sprintf(
		"/resource_language_stats?%s=%s&%s=%s&%s=%s",
		url.QueryEscape("filter[language]"),
		url.QueryEscape("l:en"),
		url.QueryEscape("filter[project]"),
		url.QueryEscape(projectId),
		url.QueryEscape("filter[resource]"),
		url.QueryEscape(resourceId),
	)

	mockData := jsonapi.MockData{
		"/languages": getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{
				{
					Response: jsonapi.MockResponse{
						Text: fmt.Sprintf(
							`{"data": {
								"type": "resources",
								"id": "%s",
								"attributes": {"slug": "branch--resslug"},
								"relationships": {"project": {"data": {"type": "projects",
																	   "id": "%s"}}}
							}}`,
							resourceId,
							projectId,
						),
					},
				},
				{
					Response: jsonapi.MockResponse{
						Text: fmt.Sprintf(
							`{"data": {
								"type": "resources",
								"id": "%s",
								"attributes": {"slug": "branch--resslug"},
								"relationships": {"project": {"data": {"type": "projects",
																	   "id": "%s"}}}
							}}`,
							resourceId,
							projectId,
						),
					},
				},
			},
		},
		projectUrl:       getProjectEndpoint(),
		statsUrl:         getStatsEndpointSourceLanguage(),
		sourceUploadsUrl: getSourceUploadPostEndpoint(),
		sourceUploadUrl:  getSourceUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Force:   true,
		Branch:  "branch",
		Workers: 1,
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	testMultipleRequests(t, mockData, resourceUrl, []string{"GET", "PATCH"}, []string{"",
		`{"data":{"type":"resources","id":"o:orgslug:p:projslug:r:branch--resslug","relationships":{"base":{"data":{"type":"resources","id":"o:orgslug:p:projslug:r:resslug"}}}}}
		`})
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrl)
	testSimpleUpload(t, mockData, sourceUploadsUrl)
	testSimpleGet(t, mockData, sourceUploadUrl)
}

func TestPushCommandChangeBase(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	resourceId := "o:orgslug:p:projslug:r:branch--resslug"
	resourceUrl := fmt.Sprintf("/resources/%s", resourceId)
	statsUrl := fmt.Sprintf(
		"/resource_language_stats?%s=%s&%s=%s&%s=%s",
		url.QueryEscape("filter[language]"),
		url.QueryEscape("l:en"),
		url.QueryEscape("filter[project]"),
		url.QueryEscape(projectId),
		url.QueryEscape("filter[resource]"),
		url.QueryEscape(resourceId),
	)

	mockData := jsonapi.MockData{
		"/languages": getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl: &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{
				{
					Response: jsonapi.MockResponse{
						Text: fmt.Sprintf(
							`{"data": {
								"type": "resources",
								"id": "%s",
								"attributes": {"slug": "branch--resslug"},
								"relationships": {"project": {"data": {"type": "projects",
																	   "id": "%s"}}}
							}}`,
							resourceId,
							projectId,
						),
					},
				},
				{
					Response: jsonapi.MockResponse{
						Text: fmt.Sprintf(
							`{"data": {
								"type": "resources",
								"id": "%s",
								"attributes": {"slug": "branch--resslug"},
								"relationships": {"project": {"data": {"type": "projects",
																	   "id": "%s"}}}
							}}`,
							resourceId,
							projectId,
						),
					},
				},
			},
		},
		projectUrl:       getProjectEndpoint(),
		statsUrl:         getStatsEndpointSourceLanguage(),
		sourceUploadsUrl: getSourceUploadPostEndpoint(),
		sourceUploadUrl:  getSourceUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Force:   true,
		Branch:  "branch",
		Workers: 1,
		Base:    "foo",
	})
	if err != nil {
		t.Errorf("%s", err)
	}

	testMultipleRequests(t, mockData, resourceUrl, []string{"GET", "PATCH"}, []string{"",
		`{
			"data":{
				"type":"resources",
				"id":"o:orgslug:p:projslug:r:branch--resslug",
				"relationships":{
					"base":{
						"data":{
							"type":"resources",
							"id":"o:orgslug:p:projslug:r:foo--resslug"
						}
					}
				}
			}
		}`})
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrl)
	testSimpleUpload(t, mockData, sourceUploadsUrl)
	testSimpleGet(t, mockData, sourceUploadUrl)
}

func TestPushNewLanguage(t *testing.T) {
	afterTest := beforeTest(t, []string{"fr"}, nil)
	defer afterTest()

	languagesRelationshipUrl := "/projects/o:orgslug:p:projslug/relationships/languages"
	mockData := jsonapi.MockData{
		"/languages":             getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:              getResourceEndpoint(),
		projectUrl:               getProjectEndpoint(),
		statsUrlAllLanguages:     getStatsEndpointAllLanguages(),
		languagesRelationshipUrl: jsonapi.GetMockTextResponse(""),
		translationUploadsUrl:    getTranslationUploadPostEndpoint(),
		translationUploadUrl:     getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Branch:      "-1",
		Force:       true,
		Translation: true,
		Languages:   []string{"fr"},
		Workers:     1,
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimplePost(
		t,
		mockData,
		languagesRelationshipUrl,
		`{"data": [{"type": "languages", "id": "l:fr"}]}`,
	)
	testSimpleUpload(t, mockData, translationUploadsUrl)
	testSimpleGet(t, mockData, translationUploadUrl)
}

func TestPushAll(t *testing.T) {
	afterTest := beforeTest(t, []string{"fr"}, nil)
	defer afterTest()

	languagesRelationshipUrl := "/projects/o:orgslug:p:projslug/relationships/languages"
	mockData := jsonapi.MockData{
		"/languages":             getLanguagesEndpoint([]string{"en", "fr", "el"}),
		resourceUrl:              getResourceEndpoint(),
		projectUrl:               getProjectEndpoint(),
		statsUrlAllLanguages:     jsonapi.GetMockTextResponse(`{"data": []}`),
		languagesRelationshipUrl: jsonapi.GetMockTextResponse(""),
		translationUploadsUrl:    getTranslationUploadPostEndpoint(),
		translationUploadUrl:     getTranslationUploadGetEndpoint(),
	}
	api := jsonapi.GetTestConnection(mockData)

	err := PushCommand(getStandardConfig(), api, PushCommandArguments{
		Branch:      "-1",
		Force:       true,
		Translation: true,
		All:         true,
		Workers:     1,
	})
	if err != nil {
		t.Error(err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimplePost(
		t,
		mockData,
		languagesRelationshipUrl,
		`{"data": [{"type": "languages", "id": "l:fr"}]}`,
	)
	testSimpleUpload(t, mockData, translationUploadsUrl)
	testSimpleGet(t, mockData, translationUploadUrl)
}

func getEmptyEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{Status: 404},
		}},
	}
}

func getResourceCreatedEndpoint() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(
		`{"data": {"type": "resources",
               "id": "o:orgslug:p:projslug:r:resslug",
							 "attributes": {"name": "aaa", "slug": "resslug"},
							 "relationships": {"project": {"data": {"type": "projects",
							                                        "id": "o:orgslug:p:projslug"}},
							                   "i18n_format": {"data": {"type": "i18n_formats",
							                                            "id": "I18N_TYPE"}}}}}`,
	)
}

func getSourceUploadPostEndpoint() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(
		`{"data": {
			"type": "resource_strings_async_uploads",
			"id": "upload_1",
			"relationships": {"resource": {"data": {"type": "resources",
			                                        "id": "o:orgslug:p:projslug:r:resslug"}}}
		}}`,
	)
}

func getSourceUploadGetEndpoint() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(
		`{"data": {"type": "resource_strings_async_uploads",
		           "id": "upload_1",
							 "attributes": {"status": "succeeded"}}}`,
	)
}

func getTranslationUploadPostEndpoint() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(
		`{"data": {
			"type": "resource_translations_async_uploads",
			"id": "upload_1",
			"relationships": {"resource": {"data": {"type": "resources",
			                                        "id": "o:orgslug:p:projslug:r:resslug"}},
			                  "language": {"data": {"type": "languages", "id": "l:fr"}}}
		}}`,
	)
}

func getTranslationUploadGetEndpoint() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(
		`{"data": {
			"type": "resource_translations_async_uploads",
			"id": "upload_1",
			"attributes": {"status": "succeeded"},
			"relationships": {"resource": {"data": {"type": "resources",
			                                        "id": "o:orgslug:p:projslug:r:resslug"}},
			                  "language": {"data": {"type": "languages", "id": "l:el"}}}
		}}`,
	)
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
	return jsonapi.GetMockTextResponse(
		fmt.Sprintf(`{"data": [%s]}`, strings.Join(result, ", ")),
	)
}

func getResourceLanguageStatsEndpoint(
	timestamp time.Time,
) *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(
		fmt.Sprintf(
			`{"data": [{"type": "resource_language_stats",
			            "id":"stats1",
									"attributes": {"last_update": "%s"},
									"relationships": {"language": {"data": {"type": "languages",
									                                        "id": "l:fr"}},
									                  "resource": {}}}]}`,
			timestamp.Format(time.RFC3339),
		),
	)
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
