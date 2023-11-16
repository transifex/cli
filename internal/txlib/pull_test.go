package txlib

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/transifex/cli/pkg/assert"
	"github.com/transifex/cli/pkg/jsonapi"
)

func TestPullCommandResourceExists(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	cfg := getStandardConfig()

	ts := getNewTestServer("This is the content")
	defer ts.Close()

	mockData := jsonapi.MockData{
		resourceUrl:             getResourceEndpoint(),
		projectUrl:              getProjectEndpoint(),
		statsUrlAllLanguages:    getStatsEndpointAllLanguages(),
		translationDownloadsUrl: getTranslationDownloadsEndpoint(),
		translationDownloadUrl:  getDownloadEndpoint(ts.URL),
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:          "default",
		Mode:              "default",
		Force:             true,
		All:               true,
		ResourceIds:       nil,
		MinimumPercentage: -1,
		Workers:           1,
	}

	err := PullCommand(cfg, &api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleTranslationDownload(t, mockData, "false")
	testSimpleGet(t, mockData, translationDownloadUrl)

	assertFileContent(t, "aaa-el.json", "This is the content")
}

func TestPullCommandFileExists(t *testing.T) {
	afterTest := beforeTest(t, []string{"el"}, nil)
	defer afterTest()

	cfg := getStandardConfig()

	ts := getNewTestServer("This is the content")
	defer ts.Close()

	mockData := jsonapi.MockData{
		resourceUrl:             getResourceEndpoint(),
		projectUrl:              getProjectEndpoint(),
		statsUrlAllLanguages:    getStatsEndpointAllLanguages(),
		translationDownloadsUrl: getTranslationDownloadsEndpoint(),
		translationDownloadUrl:  getDownloadEndpoint(ts.URL),
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:          "default",
		Mode:              "default",
		Force:             true,
		All:               true,
		ResourceIds:       nil,
		MinimumPercentage: -1,
		Workers:           1,
	}

	err := PullCommand(cfg, &api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleTranslationDownload(t, mockData, "false")
	testSimpleGet(t, mockData, translationDownloadUrl)

	assertFileContent(t, "aaa-el.json", "This is the content")
}

func TestPullCommandDownloadSource(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	cfg := getStandardConfig()

	ts := getNewTestServer("New source")
	defer ts.Close()

	mockData := jsonapi.MockData{
		resourceUrl:            getResourceEndpoint(),
		projectUrl:             getProjectEndpoint(),
		statsUrlSourceLanguage: getStatsEndpointSourceLanguage(),
		sourceDownloadsUrl:     getSourceDownloadsEndpoint(),
		sourceDownloadUrl:      getDownloadEndpoint(ts.URL),
	}

	api := jsonapi.GetTestConnection(mockData)
	err := PullCommand(
		cfg,
		&api,
		&PullCommandArguments{
			FileType:          "default",
			Mode:              "default",
			Force:             true,
			Source:            true,
			ResourceIds:       nil,
			MinimumPercentage: -1,
			Workers:           1,
		},
	)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlSourceLanguage)
	testSimpleSourceDownload(t, mockData, "false")
	testSimpleGet(t, mockData, sourceDownloadUrl)

	assertFileContent(t, "aaa.json", "New source")
}

func TestPullCommandSkipOnTranslatedMinPerc(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	cfg := getStandardConfig()

	mockData := jsonapi.MockData{
		resourceUrl: getResourceEndpoint(),
		projectUrl:  getProjectEndpoint(),
		statsUrlAllLanguages: jsonapi.GetMockTextResponse(fmt.Sprintf(
			`{"data": [{"type": "resource_language_stats",
			            "id": "%s:l:en",
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:en"}}}},
			           {"type": "resource_language_stats",
					    "id": "%s:l:el",
						"attributes": {"translated_strings": 30, "total_strings": 100},
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:el"}}}}]}`,
			resourceId,
			resourceId,
		)),
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:          "default",
		Mode:              "default",
		All:               true,
		ResourceIds:       nil,
		MinimumPercentage: 40,
		Workers:           1,
	}

	err := PullCommand(cfg, &api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
}

func TestPullCommandProceedOnEqualTranslatedMinPerc(t *testing.T) {
	afterTest := beforeTest(t, []string{"el"}, nil)
	defer afterTest()

	cfg := getStandardConfig()

	ts := getNewTestServer("This is the content")
	defer ts.Close()

	mockData := jsonapi.MockData{
		resourceUrl: getResourceEndpoint(),
		projectUrl:  getProjectEndpoint(),
		statsUrlAllLanguages: jsonapi.GetMockTextResponse(fmt.Sprintf(
			`{"data": [{"type": "resource_language_stats",
			            "id": "%s:l:en",
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:en"}}}},
			           {"type": "resource_language_stats",
					    "id": "%s:l:el",
						"attributes": {"translated_strings": 40, "total_strings": 100},
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:el"}}}}]}`,
			resourceId,
			resourceId,
		)),
		translationDownloadsUrl: getTranslationDownloadsEndpoint(),
		translationDownloadUrl:  getDownloadEndpoint(ts.URL),
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:          "default",
		Mode:              "default",
		Force:             true,
		All:               true,
		ResourceIds:       nil,
		MinimumPercentage: 40,
		Workers:           1,
	}

	err := PullCommand(cfg, &api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleTranslationDownload(t, mockData, "false")
	testSimpleGet(t, mockData, translationDownloadUrl)

	assertFileContent(t, "aaa-el.json", "This is the content")
}

func TestPullCommandOverrides(t *testing.T) {
	afterTest := beforeTest(t, []string{"el"}, nil)
	defer afterTest()

	cfg := getStandardConfig()
	cfg.Local.Resources[0].Overrides = map[string]string{"el": "custom_path.json"}

	ts := getNewTestServer("This is the content")
	defer ts.Close()

	mockData := jsonapi.MockData{
		resourceUrl:             getResourceEndpoint(),
		projectUrl:              getProjectEndpoint(),
		statsUrlAllLanguages:    getStatsEndpointAllLanguages(),
		translationDownloadsUrl: getTranslationDownloadsEndpoint(),
		translationDownloadUrl:  getDownloadEndpoint(ts.URL),
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:          "default",
		Mode:              "default",
		Force:             true,
		All:               true,
		ResourceIds:       nil,
		MinimumPercentage: -1,
		Workers:           1,
	}

	err := PullCommand(cfg, &api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleTranslationDownload(t, mockData, "false")
	testSimpleGet(t, mockData, translationDownloadUrl)

	assertFileContent(t, "custom_path.json", "This is the content")
}

func TestPullCommandMultipleLangParameters(t *testing.T) {
	afterTest := beforeTest(t, []string{"el"}, nil)
	defer afterTest()

	cfg := getStandardConfig()
	cfg.Local.Resources[0].FileFilter = "locale/<lang>/aaa-<lang>.json"

	ts := getNewTestServer("This is the content")
	defer ts.Close()

	mockData := jsonapi.MockData{
		resourceUrl:             getResourceEndpoint(),
		projectUrl:              getProjectEndpoint(),
		statsUrlAllLanguages:    getStatsEndpointAllLanguages(),
		translationDownloadsUrl: getTranslationDownloadsEndpoint(),
		translationDownloadUrl:  getDownloadEndpoint(ts.URL),
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:          "default",
		Mode:              "default",
		Force:             true,
		All:               true,
		ResourceIds:       nil,
		MinimumPercentage: -1,
		Workers:           1,
	}

	err := PullCommand(cfg, &api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleTranslationDownload(t, mockData, "false")
	testSimpleGet(t, mockData, translationDownloadUrl)

	assertFileContent(t, "locale/el/aaa-el.json", "This is the content")
}

func TestPullCommandSkipOnReviewedMinPerc(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	cfg := getStandardConfig()

	mockData := jsonapi.MockData{
		resourceUrl: getResourceEndpoint(),
		projectUrl:  getProjectEndpoint(),
		statsUrlAllLanguages: jsonapi.GetMockTextResponse(fmt.Sprintf(
			`{"data": [{"type": "resource_language_stats",
			            "id": "%s:l:en",
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:en"}}}},
			           {"type": "resource_language_stats",
					    "id": "%s:l:el",
						"attributes": {"reviewed_strings": 30, "total_strings": 100},
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:el"}}}}]}`,
			resourceId,
			resourceId,
		)),
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:          "default",
		Mode:              "reviewed",
		All:               true,
		ResourceIds:       nil,
		MinimumPercentage: 40,
		Workers:           1,
	}

	err := PullCommand(cfg, &api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
}

func TestPullCommandSkipOnProofreadMinPerc(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	cfg := getStandardConfig()

	mockData := jsonapi.MockData{
		resourceUrl: getResourceEndpoint(),
		projectUrl:  getProjectEndpoint(),
		statsUrlAllLanguages: jsonapi.GetMockTextResponse(fmt.Sprintf(
			`{"data": [{"type": "resource_language_stats",
			            "id": "%s:l:en",
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:en"}}}},
			           {"type": "resource_language_stats",
					    "id": "%s:l:el",
						"attributes": {"proofread_strings": 30, "total_strings": 100},
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:el"}}}}]}`,
			resourceId,
			resourceId,
		)),
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:          "default",
		Mode:              "proofread",
		All:               true,
		ResourceIds:       nil,
		MinimumPercentage: 40,
		Workers:           1,
	}

	err := PullCommand(cfg, &api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
}

func TestPullCommandPercentageArgumentShouldWinOverResource(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	cfg := getStandardConfig()
	cfg.Local.Resources[0].MinimumPercentage = 30

	mockData := jsonapi.MockData{
		resourceUrl: getResourceEndpoint(),
		projectUrl:  getProjectEndpoint(),
		statsUrlAllLanguages: jsonapi.GetMockTextResponse(fmt.Sprintf(
			`{"data": [{"type": "resource_language_stats",
			            "id": "%s:l:en",
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:en"}}}},
			           {"type": "resource_language_stats",
					    "id": "%s:l:el",
						"attributes": {"translated_strings": 40, "total_strings": 100},
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:el"}}}}]}`,
			resourceId,
			resourceId,
		)),
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:          "default",
		Mode:              "default",
		All:               true,
		ResourceIds:       nil,
		MinimumPercentage: 50,
		Workers:           1,
	}

	err := PullCommand(cfg, &api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
}

func TestPercentageWinsOverForce(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	cfg := getStandardConfig()

	mockData := jsonapi.MockData{
		resourceUrl: getResourceEndpoint(),
		projectUrl:  getProjectEndpoint(),
		statsUrlAllLanguages: jsonapi.GetMockTextResponse(fmt.Sprintf(
			`{"data": [{"type": "resource_language_stats",
			            "id": "%s:l:en",
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:en"}}}},
			           {"type": "resource_language_stats",
					    "id": "%s:l:el",
						"attributes": {"translated_strings": 30, "total_strings": 100},
						"relationships": {"language": {"data": {"type": "languages",
						                                        "id": "l:el"}}}}]}`,
			resourceId,
			resourceId,
		)),
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:          "default",
		Mode:              "default",
		All:               true,
		ResourceIds:       nil,
		MinimumPercentage: 40,
		Workers:           1,
		Force:             true,
	}

	err := PullCommand(cfg, &api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
}

func TestGetActedOnStringsPercentage(t *testing.T) {
	result := getActedOnStringsPercentage(float32(2), float32(10))
	assert.Equal(t, result, float32(20))

	result = getActedOnStringsPercentage(float32(1), float32(1000))
	assert.Equal(t, result, float32(0.1))

	result = getActedOnStringsPercentage(float32(12), float32(9876))
	assert.Equal(t, result, float32(0.12150668))

	result = getActedOnStringsPercentage(float32(991), float32(1000))
	assert.Equal(t, result, float32(99.1))
}

func TestShouldSkipDueToStringPercentage(t *testing.T) {
	result := shouldSkipDueToStringPercentage(10, 2, 10)
	assert.Equal(t, result, false)

	result = shouldSkipDueToStringPercentage(100, 10, 10)
	assert.Equal(t, result, false)

	result = shouldSkipDueToStringPercentage(20, 1, 10)
	assert.Equal(t, result, true)

	result = shouldSkipDueToStringPercentage(10, 1, 1000)
	assert.Equal(t, result, true)

	result = shouldSkipDueToStringPercentage(10, 1, 1000)
	assert.Equal(t, result, true)

	result = shouldSkipDueToStringPercentage(99, 991, 1000)
	assert.Equal(t, result, false)

	result = shouldSkipDueToStringPercentage(99, 989, 1000)
	assert.Equal(t, result, true)
}

func TestDownloadPseudoTranslations(t *testing.T) {
	afterTest := beforeTest(t, []string{"el"}, nil)
	defer afterTest()

	cfg := getStandardConfig()
	cfg.Local.Resources[0].FileFilter = "locale/<lang>/aaa-<lang>.json"

	ts := getNewTestServer("This is the content")
	defer ts.Close()

	mockData := jsonapi.MockData{
		resourceUrl:          getResourceEndpoint(),
		projectUrl:           getProjectEndpoint(),
		statsUrlAllLanguages: getStatsEndpointAllLanguages(),
		sourceDownloadsUrl:   getSourceDownloadsEndpoint(),
		sourceDownloadUrl:    getDownloadEndpoint(ts.URL),
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:          "default",
		Mode:              "default",
		Force:             true,
		All:               true,
		ResourceIds:       nil,
		MinimumPercentage: -1,
		Workers:           1,
		Pseudo:            true,
	}

	err := PullCommand(cfg, &api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	assertFileContent(t, "locale/el_pseudo/aaa-el_pseudo.json", "This is the content")
	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleSourceDownload(t, mockData, "true")
	testSimpleGet(t, mockData, sourceDownloadUrl)
}

func TestKeepNewFilesSource(t *testing.T) {
	afterTest := beforeTest(t, nil, nil)
	defer afterTest()

	cfg := getStandardConfig()
	assertFileContent(t, "aaa.json", "")

	ts := getNewTestServer("New source")
	defer ts.Close()

	mockData := jsonapi.MockData{
		resourceUrl:            getResourceEndpoint(),
		projectUrl:             getProjectEndpoint(),
		statsUrlSourceLanguage: getStatsEndpointSourceLanguage(),
		sourceDownloadsUrl:     getSourceDownloadsEndpoint(),
		sourceDownloadUrl:      getDownloadEndpoint(ts.URL),
	}

	api := jsonapi.GetTestConnection(mockData)
	err := PullCommand(
		cfg,
		&api,
		&PullCommandArguments{
			FileType:          "default",
			Mode:              "default",
			Force:             true,
			Source:            true,
			ResourceIds:       nil,
			MinimumPercentage: -1,
			Workers:           1,
			DisableOverwrite:  true,
		},
	)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlSourceLanguage)
	testSimpleSourceDownload(t, mockData, "false")
	testSimpleGet(t, mockData, sourceDownloadUrl)

	assertFileContent(t, "aaa.json.new", "New source")
	assertFileContent(t, "aaa.json", "")
}

func TestKeepNewFilesTranslation(t *testing.T) {
	afterTest := beforeTest(t, []string{"el"}, nil)
	defer afterTest()

	cfg := getStandardConfig()

	ts := getNewTestServer("This is the content")
	defer ts.Close()

	mockData := jsonapi.MockData{
		resourceUrl:             getResourceEndpoint(),
		projectUrl:              getProjectEndpoint(),
		statsUrlAllLanguages:    getStatsEndpointAllLanguages(),
		translationDownloadsUrl: getTranslationDownloadsEndpoint(),
		translationDownloadUrl:  getDownloadEndpoint(ts.URL),
	}

	api := jsonapi.GetTestConnection(mockData)

	arguments := PullCommandArguments{
		FileType:          "default",
		Mode:              "default",
		Force:             true,
		All:               true,
		ResourceIds:       nil,
		MinimumPercentage: -1,
		Workers:           1,
		DisableOverwrite:  true,
	}

	err := PullCommand(cfg, &api, &arguments)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, resourceUrl)
	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, statsUrlAllLanguages)
	testSimpleTranslationDownload(t, mockData, "false")
	testSimpleGet(t, mockData, translationDownloadUrl)
	assertFileContent(t, "aaa-el.json", `{"hello": "world"}`)
	assertFileContent(t, "aaa-el.json.new", "This is the content")
}

func assertFileContent(t *testing.T, expectedPath, expectedContent string) {
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Error(err)
	}
	actual := strings.Trim(string(data), " \n")
	if actual != expectedContent {
		t.Errorf("Wrong file saved; expected: '%s', got '%s'", expectedContent, actual)
	}
}

func getTranslationDownloadsEndpoint() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(
		`{"data": {"type": "resource_translations_async_downloads",
				   "id": "download_1"}}`,
	)
}

func getSourceDownloadsEndpoint() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(
		`{"data": {"type": "resource_strings_async_downloads",
				   "id": "download_1"}}`,
	)
}

func getDownloadEndpoint(url string) *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Status:   303,
				Redirect: url,
			},
		}},
	}
}

func testSimpleTranslationDownload(
	t *testing.T,
	mockData jsonapi.MockData,
	pseudo string,
) {
	testSimplePost(
		t,
		mockData,
		translationDownloadsUrl,
		fmt.Sprintf(
			`{"data": {
				"type": "resource_translations_async_downloads",
				"attributes": {"content_encoding": "",
							   "file_type": "default",
							   "mode": "default",
							   "pseudo": %s},
				"relationships": {
					"language": {"data": {"type": "languages", "id": "l:el"}},
					"resource": {"data": {"type": "resources", "id": "%s"}}
				}
			}}`,
			pseudo, resourceId,
		),
	)
}

func testSimpleSourceDownload(
	t *testing.T,
	mockData jsonapi.MockData,
	pseudo string,
) {
	testSimplePost(
		t,
		mockData,
		sourceDownloadsUrl,
		fmt.Sprintf(
			`{"data": {
				"type": "resource_strings_async_downloads",
				"attributes": {"content_encoding": "",
							   "file_type": "default",
							   "pseudo": %s},
				"relationships": {
					"resource": {"data": {"type": "resources", "id": "%s"}}
				}
			}}`,
			pseudo, resourceId,
		),
	)
}
