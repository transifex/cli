package txlib

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/assert"
	"github.com/transifex/cli/pkg/jsonapi"
)

const (
	projectsUrlDeleteCommand = "/projects?" +
		"filter%5Borganization%5D=o%3Aorgslug&filter%5Bslug%5D=projslug"
	resourcesUrlDeleteCommand = "/resources?filter%5B" +
		"project%5D=o%3Aorgslug%3Ap%3Aprojslug"
	resourceUrlDeleteCommand              = "/resources/o:orgslug:p:projslug:r:resslug"
	resourceUrlBranchDeleteCommand        = "/resources/o:orgslug:p:projslug:r:resslug--abranch"
	resource1UrlDeleteCommand             = "/resources/o:orgslug:p:projslug:r:resslug1"
	resource1UrlBranchDeleteCommand       = "/resources/o:orgslug:p:projslug:r:resslug1--abranch"
	resourceLanguageStatsUrlDeleteCommand = "/resource_language_stats?" +
		"filter%5Bproject%5D=o%3Aorgslug%3Ap%3Aprojslug&" +
		"filter%5Bresource%5D=o%3Aorgslug%3Ap%3Aprojslug%3Ar%3Aresslug"
)

func TestDeleteSingleResource(t *testing.T) {
	var pkgDir, tmpDir = beforeDeleteTest(t)
	defer afterDeleteTest(pkgDir, tmpDir)
	mockData := getMockedDataForResourceDelete()

	api := jsonapi.GetTestConnection(mockData)

	err := deleteResource(
		&api,
		getStandardConfigDelete(),
		*getStandardConfigDelete().FindResource("projslug.resslug"),
		DeleteCommandArguments{
			ResourceIds: []string{},
		},
	)
	if err != nil {
		t.Errorf("%s", err)
	}
}
func TestDeleteAbortedByTranslations(t *testing.T) {
	mockData := getMockedDataForResourceDelete()
	mockData[resourceLanguageStatsUrlDeleteCommand] = deleteGetResLangStatsEndpoint()
	api := jsonapi.GetTestConnection(mockData)

	err := deleteResource(
		&api,
		getStandardConfigDelete(),
		*getStandardConfigDelete().FindResource("projslug.resslug"),
		DeleteCommandArguments{
			ResourceIds: []string{},
		},
	)
	if err == nil {
		t.Error("Translations should raise and error.")
	}
	assert.True(t, strings.Contains(fmt.Sprintf("%s", err), "translations"))
}

func TestDeleteForceFlag(t *testing.T) {
	mockData := getMockedDataForResourceDelete()

	api := jsonapi.GetTestConnection(mockData)

	err := deleteResource(
		&api,
		getStandardConfigDelete(),
		*getStandardConfigDelete().FindResource("projslug.resslug"),
		DeleteCommandArguments{
			ResourceIds: []string{},
			Force:       true,
		},
	)
	if err != nil {
		t.Errorf("Translations should raise and error due to force flag. %s",
			err)
	}
}

func TestDeleteCommandResourceNotInConfig(t *testing.T) {
	mockData := getMockedDataForResourceDelete()

	api := jsonapi.GetTestConnection(mockData)

	err := DeleteCommand(
		&config.Config{
			Local: &config.LocalConfig{},
		},
		api,
		&DeleteCommandArguments{
			ResourceIds: []string{"a.b"},
		},
	)
	if err == nil {
		t.Errorf("If resource is not in config there"+
			" should be an error: %s", err)
	}
}

func TestDeleteCommandMultipleDeletes(t *testing.T) {
	var pkgDir, tmpDir = beforeDeleteTest(t)
	defer afterDeleteTest(pkgDir, tmpDir)
	var filePath = filepath.Join(tmpDir, ".tx", "config")
	cfg, _ := config.LoadFromPaths("", filePath)
	cfg.Local.Resources = []config.Resource{
		{
			OrganizationSlug: "orgslug",
			ProjectSlug:      "projslug",
			ResourceSlug:     "resslug",
			Type:             "I18N_TYPE",
			SourceFile:       "aaa.json",
			FileFilter:       "aaa-<lang>.json",
		},
		{
			OrganizationSlug: "orgslug",
			ProjectSlug:      "projslug",
			ResourceSlug:     "resslug1",
			Type:             "I18N_TYPE",
			SourceFile:       "aaa.json",
			FileFilter:       "aaa-<lang>.json",
		},
	}

	mockData := getMockedDataForResourceDelete()

	api := jsonapi.GetTestConnection(mockData)
	err := DeleteCommand(
		&cfg,
		api,
		&DeleteCommandArguments{
			ResourceIds: []string{"projslug.resslug", "projslug.resslug1"},
		},
	)
	if err != nil {
		t.Errorf("Should be deleted with no error: %s", err)
	}
}

func TestDeleteCommandBulkDeletes(t *testing.T) {
	var pkgDir, tmpDir = beforeDeleteTest(t)
	defer afterDeleteTest(pkgDir, tmpDir)
	var filePath = filepath.Join(tmpDir, ".tx", "config")
	cfg, _ := config.LoadFromPaths("", filePath)
	cfg.Local.Resources = []config.Resource{
		{
			OrganizationSlug: "orgslug",
			ProjectSlug:      "projslug",
			ResourceSlug:     "resslug",
			Type:             "I18N_TYPE",
			SourceFile:       "aaa.json",
			FileFilter:       "aaa-<lang>.json",
		},
		{
			OrganizationSlug: "orgslug",
			ProjectSlug:      "projslug",
			ResourceSlug:     "resslug1",
			Type:             "I18N_TYPE",
			SourceFile:       "aaa.json",
			FileFilter:       "aaa-<lang>.json",
		},
	}

	mockData := getMockedDataForResourceDelete()

	api := jsonapi.GetTestConnection(mockData)
	err := DeleteCommand(
		&cfg,
		api,
		&DeleteCommandArguments{
			ResourceIds: []string{"projslug.*"},
		},
	)
	endpoint := mockData["/resources/o:orgslug:p:projslug:r:resslug"]
	if endpoint.Count != 1 {
		t.Errorf("Got %d requests to "+
			"'/resources/o:orgslug:p:projslug:r:resslug', expected 1",
			endpoint.Count)
	}

	endpoint = mockData["/resources/o:orgslug:p:projslug:r:resslug1"]
	if endpoint.Count != 1 {
		t.Errorf("Got %d requests to "+
			"'/resources/o:orgslug:p:projslug:r:resslug1', expected 1",
			endpoint.Count)
	}

	if err != nil {
		t.Errorf("Should be deleted with no error: %s", err)
	}
}

func TestDeleteUpdatesConfig(t *testing.T) {

	var pkgDir, tmpDir = beforeDeleteTest(t)
	defer afterDeleteTest(pkgDir, tmpDir)
	var filePath = filepath.Join(tmpDir, ".tx", "config")
	cfg, _ := config.LoadFromPaths("", filePath)
	cfg.Local.Resources = []config.Resource{
		{
			OrganizationSlug: "orgslug",
			ProjectSlug:      "projslug",
			ResourceSlug:     "resslug",
			Type:             "I18N_TYPE",
			SourceFile:       "aaa.json",
			FileFilter:       "aaa-<lang>.json",
		},
		{
			OrganizationSlug: "orgslug",
			ProjectSlug:      "projslug",
			ResourceSlug:     "resslug1",
			Type:             "I18N_TYPE",
			SourceFile:       "aaa.json",
			FileFilter:       "aaa-<lang>.json",
		},
	}

	mockData := getMockedDataForResourceDelete()

	api := jsonapi.GetTestConnection(mockData)
	err := DeleteCommand(
		&cfg,
		api,
		&DeleteCommandArguments{
			ResourceIds: []string{"projslug.*"},
		},
	)

	if err != nil {
		t.Errorf("%s", err)
	}

	cfgReloaded, err := config.LoadFromPaths("", filePath)

	assert.Equal(t, len(cfgReloaded.Local.Resources), 0)

}

func TestForceWorks(t *testing.T) {

	mockData := getMockedDataForResourceDelete()
	mockData[resourceLanguageStatsUrlDeleteCommand] = deleteGetResLangStatsEndpoint()
	api := jsonapi.GetTestConnection(mockData)

	err := deleteResource(
		&api,
		getStandardConfigDelete(),
		*getStandardConfigDelete().FindResource("projslug.resslug"),
		DeleteCommandArguments{
			ResourceIds: []string{},
			Force:       true,
		},
	)
	if err != nil {
		t.Error("Force should delete even with translations.")
	}

}

func TestSkipWorks(t *testing.T) {
	var pkgDir, tmpDir = beforeDeleteTest(t)
	defer afterDeleteTest(pkgDir, tmpDir)
	var filePath = filepath.Join(tmpDir, ".tx", "config")
	cfg, _ := config.LoadFromPaths("", filePath)
	cfg.Local.Resources = []config.Resource{
		{
			OrganizationSlug: "orgslug",
			ProjectSlug:      "projslug",
			ResourceSlug:     "resslug",
			Type:             "I18N_TYPE",
			SourceFile:       "aaa.json",
			FileFilter:       "aaa-<lang>.json",
		},
		{
			OrganizationSlug: "orgslug",
			ProjectSlug:      "projslug",
			ResourceSlug:     "resslug1",
			Type:             "I18N_TYPE",
			SourceFile:       "aaa.json",
			FileFilter:       "aaa-<lang>.json",
		},
	}

	mockData := getMockedDataForResourceDelete()

	api := jsonapi.GetTestConnection(mockData)
	err := DeleteCommand(
		&cfg,
		api,
		&DeleteCommandArguments{
			ResourceIds: []string{"projslug.resslugdoesntexist",
				"projslug.resslug", "projslug.resslug1"},
		},
	)
	if err == nil {
		t.Error("Should get an error without Skip")
	}
	err = DeleteCommand(
		&cfg,
		api,
		&DeleteCommandArguments{
			ResourceIds: []string{"projslug.resslugdoesntexist",
				"projslug.resslug", "projslug.resslug1"},
			Skip: true,
		},
	)
	if err != nil {
		t.Errorf("Should not get an error with Skip: %s", err)
	}
}

func TestDeleteBranchWorks(t *testing.T) {
	var pkgDir, tmpDir = beforeDeleteTest(t)
	defer afterDeleteTest(pkgDir, tmpDir)
	var filePath = filepath.Join(tmpDir, ".tx", "config")
	cfg, _ := config.LoadFromPaths("", filePath)
	cfg.Local.Resources = []config.Resource{
		{
			OrganizationSlug: "orgslug",
			ProjectSlug:      "projslug",
			ResourceSlug:     "resslug",
			Type:             "I18N_TYPE",
			SourceFile:       "aaa.json",
			FileFilter:       "aaa-<lang>.json",
		},
		{
			OrganizationSlug: "orgslug",
			ProjectSlug:      "projslug",
			ResourceSlug:     "resslug1",
			Type:             "I18N_TYPE",
			SourceFile:       "aaa.json",
			FileFilter:       "aaa-<lang>.json",
		},
	}

	mockData := getMockedDataForBranchResourceDelete()

	api := jsonapi.GetTestConnection(mockData)
	err := DeleteCommand(
		&cfg,
		api,
		&DeleteCommandArguments{
			ResourceIds: []string{"projslug.resslug", "projslug.resslug1"},
			Branch:      "abranch",
		},
	)
	if err != nil {
		t.Errorf("Should be deleted with no error: %s", err)
	}
}

func getStandardConfigDelete() *config.Config {
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
				},
				{
					OrganizationSlug: "orgslug",
					ProjectSlug:      "projslug",
					ResourceSlug:     "resslug1",
					Type:             "I18N_TYPE",
					SourceFile:       "aaa.json",
					FileFilter:       "aaa-<lang>.json",
				},
			},
		},
	}
}

func beforeDeleteTest(t *testing.T) (string, string) {
	pkgDir, _ := os.Getwd()
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}
	_ = os.Chdir(tmpDir)

	err = InitCommand()
	if err != nil {
		t.Error(err)
	}
	return pkgDir, tmpDir
}

func afterDeleteTest(pkgDir string, tmpDir string) {
	_ = os.Chdir(pkgDir)
	err := os.RemoveAll(tmpDir)
	if err != nil {
		fmt.Println("Delete error:", err)
	}
}

func deleteGetOrganizationEndpoint() *jsonapi.MockEndpoint {
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

func deleteGetProjectsEndpoint() *jsonapi.MockEndpoint {
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

func deleteGetResourceEndpoint() *jsonapi.MockEndpoint {
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

func deleteGetResourceBranchEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: `{"data": {"type": "resources",
								 "id": "o:orgslug:p:projslug:r:resslug--abranch",
								 "attributes": {"slug": "resslug--abranch"}}}`,
			},
		}},
	}
}

func deleteGetResource1Endpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: `{"data": {"type": "resources",
								 "id": "o:orgslug:p:projslug:r:resslug1",
								 "attributes": {"slug": "resslug1"}}}`,
			},
		}},
	}
}

func deleteGetResourceBranch1Endpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: `{"data": {"type": "resources",
								 "id": "o:orgslug:p:projslug:r:resslug1--branch",
								 "attributes": {"slug": "resslug1--branch"}}}`,
			},
		}},
	}
}

func deleteGetResourcesEndpoint() *jsonapi.MockEndpoint {
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

func deleteGetResourcesBranchEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{
			{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "resources",
									"id": "o:orgslug:p:projslug:r:resslug--abranch",
									"attributes": {"slug": "resslug--abranch"}}]}`,
				},
			},
			{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "resources",
									"id": "o:orgslug:p:projslug:r:resslug1--abranch",
									"attributes": {"slug": "resslug1--abranch"}}]}`,
				},
			},
		},
	}
}

func deleteGetResLangStatsEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{{
			Response: jsonapi.MockResponse{
				Text: fmt.Sprintf(
					`{"data": [{
						"type": "resource_language_stats",
						"id":"stats1",
						"attributes": {
							"last_update": "",
							"translated_strings": 2
						},
						"relationships": {
							"language": {"data": {"type": "languages",
												  "id": "l:fr"}},
							"resource": {}
						}
					}]}`,
				),
			},
		}},
	}
}
func getMockedDataForResourceDelete() jsonapi.MockData {
	return jsonapi.MockData{
		"/organizations":          deleteGetOrganizationEndpoint(),
		projectsUrlDeleteCommand:  deleteGetProjectsEndpoint(),
		resourceUrlDeleteCommand:  deleteGetResourceEndpoint(),
		resource1UrlDeleteCommand: deleteGetResourceEndpoint(),
		resourcesUrlDeleteCommand: deleteGetResourcesEndpoint(),
	}
}

func getMockedDataForBranchResourceDelete() jsonapi.MockData {
	return jsonapi.MockData{
		"/organizations":                deleteGetOrganizationEndpoint(),
		projectsUrlDeleteCommand:        deleteGetProjectsEndpoint(),
		resourceUrlBranchDeleteCommand:  deleteGetResourceBranchEndpoint(),
		resource1UrlBranchDeleteCommand: deleteGetResourceBranch1Endpoint(),
		resourcesUrlDeleteCommand:       deleteGetResourcesBranchEndpoint(),
	}
}
