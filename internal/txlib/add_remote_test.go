package txlib

import (
	"fmt"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
)

func TestAddRemote(t *testing.T) {
	curDir, _ := os.Getwd()
	tempDir, _ := os.MkdirTemp("", "")
	defer os.RemoveAll(tempDir)
	_ = os.Chdir(tempDir)
	defer os.Chdir(curDir)

	resourcesUrl := fmt.Sprintf(
		"/resources?%s=%s",
		url.QueryEscape("filter[project]"),
		url.QueryEscape(projectId),
	)
	i18nFormatsUrl := fmt.Sprintf(
		"/i18n_formats?%s=%s",
		url.QueryEscape("filter[organization]"),
		url.QueryEscape("o:orgslug"),
	)
	mockData := jsonapi.MockData{
		projectUrl: getProjectEndpoint(),
		resourcesUrl: jsonapi.GetMockTextResponse(
			`{"data": [{
				"type": "resources",
				"id": "o:orgslug:p:projslug:r:resslug",
				"attributes": {"slug": "resslug"},
				"relationships": {
					"i18n_format": {"data": {"type": "i18n_formats", "id": "PO"}}
				}
			}]}`,
		),
		i18nFormatsUrl: jsonapi.GetMockTextResponse(
			`{"data": [{
				"type": "i18n_formats",
				"id": "PO",
				"attributes": {"file_extensions": [".po"]}
			}]}`,
		),
	}

	api := jsonapi.GetTestConnection(mockData)
	cfg := &config.Config{Local: &config.LocalConfig{}}

	err := AddRemoteCommand(
		cfg,
		&api,
		"https://www.transifex.com/orgslug/projslug/whatever/whatever/",
		// Lets make the file filter a bit weird
		"locale/<project_slug><project_slug>.<resource_slug>/<lang>.<ext>",
		50,
	)
	if err != nil {
		t.Errorf("%s", err)
	}

	testSimpleGet(t, mockData, projectUrl)
	testSimpleGet(t, mockData, resourcesUrl)
	testSimpleGet(t, mockData, i18nFormatsUrl)
	actual := cfg.Local.Resources
	expected := []config.Resource{{
		OrganizationSlug:  "orgslug",
		ProjectSlug:       "projslug",
		ResourceSlug:      "resslug",
		FileFilter:        "locale/projslugprojslug.resslug/<lang>.po",
		SourceFile:        "locale/projslugprojslug.resslug/en.po",
		SourceLanguage:    "en",
		Type:              "PO",
		MinimumPercentage: 50,
	}}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got request '%+v', expected '%+v'", actual, expected)
	}
}
