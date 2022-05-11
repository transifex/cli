package txlib

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/assert"
	"github.com/transifex/cli/pkg/jsonapi"
)

const (
	projectsUrlStatusCommand = "/projects?" +
		"filter%5Borganization%5D=o%3Aorgslug&filter%5Bslug%5D=projslug"
	resourcesUrlStatusCommand = "/resources?filter%5B" +
		"project%5D=o%3Aorgslug%3Ap%3Aprojslug"
	resourceUrlStatusCommand    = "/resources/o:orgslug:p:projslug:r:resslug"
	resource1UrlStatusCommand   = "/resources/o:orgslug:p:projslug:r:resslug1"
	languagesUrlStatusCommand   = "/languages/l:en"
	languagesUrlElStatusCommand = "/languages/l:el"
)

func TestGetSourceLangFromConfig(t *testing.T) {
	mockData := getMockedDataForResourceStatus()
	api := jsonapi.GetTestConnection(mockData)
	cfg := getStandardConfigStatus()
	cfgResource := cfg.Local.Resources[0]
	result, err := getSourceLanguage(cfg, &api, &cfgResource)
	if err != nil {
		t.Errorf("Should not get error for getting source lang: %s", err)
	}

	assert.Equal(t, result, "en")

	cfgResource = cfg.Local.Resources[1]
	result, err = getSourceLanguage(cfg, &api, &cfgResource)
	if err != nil {
		t.Errorf("Should not get error for getting source lang: %s", err)
	}

	assert.Equal(t, result, "el")
}

func TestGetSourceLangFromServer(t *testing.T) {
	mockData := getMockedDataForResourceStatus()
	api := jsonapi.GetTestConnection(mockData)
	cfg := getStandardConfigStatus()
	cfgResource := cfg.Local.Resources[0]
	cfgResource.SourceLanguage = ""
	result, err := getSourceLanguage(cfg, &api, &cfgResource)
	if err != nil {
		t.Errorf("Should not get error for getting source lang: %s", err)
	}

	assert.Equal(t, result, "en")

	cfgResource = cfg.Local.Resources[1]
	result, err = getSourceLanguage(cfg, &api, &cfgResource)
	if err != nil {
		t.Errorf("Should not get error for getting source lang: %s", err)
	}

	assert.Equal(t, result, "el")
}

func TestStatusWithNoResourcesAsParameters(t *testing.T) {
	var pkgDir, tmpDir = beforeStatusTest(t, []string{"el", "fr", "en"})
	defer afterStatusTest(pkgDir, tmpDir)

	mockData := getMockedDataForResourceStatus()
	api := jsonapi.GetTestConnection(mockData)
	cfg := getStandardConfigStatus()

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = StatusCommand(
		cfg,
		api,
		&StatusCommandArguments{},
	)

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	result := string(out)
	assert.True(t, strings.Contains(
		result, "projslug -> resslug (1 of 2)"))
	assert.True(t, strings.Contains(
		result, "projslug -> resslug1 (2 of 2)"))
	assert.True(t, strings.Contains(
		result, "aaa-en.json  (source)"))
	assert.True(t, strings.Contains(
		result, "aaa-el.json  (source)"))
}

func TestStatusWithOverrides(t *testing.T) {
	var pkgDir, tmpDir = beforeStatusTest(t, []string{"el", "fr", "en"})
	defer afterStatusTest(pkgDir, tmpDir)

	mockData := getMockedDataForResourceStatus()
	api := jsonapi.GetTestConnection(mockData)
	cfg := getStandardConfigStatus()
	cfg.Local.Resources[0].Overrides = map[string]string{
		"el": "greekOverride.json",
	}
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = StatusCommand(
		cfg,
		api,
		&StatusCommandArguments{},
	)

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	result := string(out)
	assert.True(t, strings.Contains(
		result, "projslug -> resslug (1 of 2)"))
	assert.True(t, strings.Contains(
		result, "projslug -> resslug1 (2 of 2)"))
	assert.True(t, strings.Contains(
		result, "aaa-en.json  (source)"))
	assert.True(t, strings.Contains(
		result, "aaa-el.json  (source)"))
	assert.True(t, strings.Contains(
		result, "el: greekOverride.json"))
}

func TestStatusWithResourceAsParameter(t *testing.T) {
	var pkgDir, tmpDir = beforeStatusTest(t, []string{"el", "fr", "en"})
	defer afterStatusTest(pkgDir, tmpDir)

	mockData := getMockedDataForResourceStatus()
	api := jsonapi.GetTestConnection(mockData)
	cfg := getStandardConfigStatus()

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = StatusCommand(
		cfg,
		api,
		&StatusCommandArguments{
			ResourceIds: []string{"projslug.resslug1"}},
	)

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	result := string(out)
	assert.True(t, strings.Contains(
		result, "projslug -> resslug1 (1 of 1)"))
	assert.True(t, strings.Contains(
		result, "aaa-el.json  (source)"))
}

func getStandardConfigStatus() *config.Config {
	return &config.Config{
		Local: &config.LocalConfig{
			Resources: []config.Resource{
				{
					OrganizationSlug: "orgslug",
					ProjectSlug:      "projslug",
					ResourceSlug:     "resslug",
					Type:             "I18N_TYPE",
					SourceFile:       "aaa.json",
					FileFilter:       "aaa-<lang>.json",
					SourceLanguage:   "en",
				},
				{
					OrganizationSlug: "orgslug",
					ProjectSlug:      "projslug",
					ResourceSlug:     "resslug1",
					Type:             "I18N_TYPE",
					SourceFile:       "aaa.json",
					FileFilter:       "aaa-<lang>.json",
					SourceLanguage:   "el",
				},
			},
		},
	}
}

func beforeStatusTest(t *testing.T, languageCodes []string) (string, string) {
	pkgDir, _ := os.Getwd()
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}
	_ = os.Chdir(tmpDir)

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

	return pkgDir, tmpDir
}

func afterStatusTest(pkgDir string, tmpDir string) {
	_ = os.Chdir(pkgDir)
	err := os.RemoveAll(tmpDir)
	if err != nil {
		fmt.Println("status error:", err)
	}
}
func statusGetOrganizationEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{
			{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "organizations",
									"id": "o:orgslug",
									"attributes": {"slug": "orgslug"}}]}`,
				},
			},
			{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "organizations",
									  "id": "o:orgslug",
									  "attributes": {"slug": "orgslug"}}]}`,
				},
			},
		},
	}
}

func statusGetProjectsEndpoint() *jsonapi.MockEndpoint {
	selfUrl := "/projects/o:orgslug:p:projslug/relationships/languages"
	relatedUrl := "/projects/o:orgslug:p:projslug/languages"
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{
			{
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
			},
			{
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
			},
		},
	}
}

func statusGetResourceEndpoint() *jsonapi.MockEndpoint {
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

func statusGetResourcesEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{
			{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "resources",
									"id": "o:orgslug:p:projslug:r:resslug",
									"attributes": {"slug": "resslug"}}]}`,
				},
			},
			{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "resources",
									"id": "o:orgslug:p:projslug:r:resslug1",
									"attributes": {"slug": "resslug1"}}]}`,
				},
			},
		},
	}
}

func statusGetLanguagesEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: `{"data": {"type": "languages",
								  "id": "l:en",
								  "attributes": {"code": "en"}}}`,
			},
		}},
	}
}

func statusGetLanguagesEndpointEl() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: `{"data": {"type": "languages",
								  "id": "l:el",
								  "attributes": {"code": "el"}}}`,
			},
		}},
	}
}

func getMockedDataForResourceStatus() jsonapi.MockData {
	return jsonapi.MockData{
		"/organizations":            statusGetOrganizationEndpoint(),
		projectsUrlStatusCommand:    statusGetProjectsEndpoint(),
		resourceUrlStatusCommand:    statusGetResourceEndpoint(),
		resource1UrlStatusCommand:   statusGetResourceEndpoint(),
		resourcesUrlStatusCommand:   statusGetResourcesEndpoint(),
		languagesUrlStatusCommand:   statusGetLanguagesEndpoint(),
		languagesUrlElStatusCommand: statusGetLanguagesEndpointEl(),
	}
}
