package txlib

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/assert"
	"github.com/transifex/cli/pkg/jsonapi"
)

func TestSuccessfulFindOrganizationSlug(t *testing.T) {
	org1ProjectsUrl := "/projects?filter%5Borganization%5D=o%3Aorg&" +
		"filter%5Bslug%5D=projslug"
	org2ProjectsUrl := "/projects?filter%5Borganization%5D=o%3Aorg2&" +
		"filter%5Bslug%5D=projslug"
	mockData := jsonapi.MockData{
		"/organizations": jsonapi.GetMockTextResponse(
			`{"data": [{"type": "organizations",
			            "id": "o:org",
									"attributes": {"slug": "org"}},
			           {"type": "organizations",
								  "id": "o:org2",
									"attributes": {"slug": "org2"}}]}`,
		),
		org1ProjectsUrl: jsonapi.GetMockTextResponse(
			`{"data": [{
				"type": "projects",
				"id": "o:orgslug:p:projslug",
				"attributes": {"name": "Proj Name", "slug": "projslug"},
				"relationships": {"organization": {
					"data": {"type": "organizations", "id": "o:orgslug"},
					"links": {"related": "/organizations/o:orgslug"}
				}}
			}]}`,
		),
		org2ProjectsUrl: jsonapi.GetMockTextResponse(`{"data": []}`),
	}

	api := jsonapi.GetTestConnection(mockData)
	resource := config.Resource{
		ProjectSlug: "projslug",
	}
	res, err := getOrganizationSlug(api, &resource)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, res, "org")
}

func TestFailToFindOrganizationSlug(t *testing.T) {
	org1ProjectsUrl := "/projects?filter%5Borganization%5D=o%3Aorg&" +
		"filter%5Bslug%5D=projslug"
	org2ProjectsUrl := "/projects?filter%5Borganization%5D=o%3Aorg2&" +
		"filter%5Bslug%5D=projslug"
	mockData := jsonapi.MockData{
		"/organizations": jsonapi.GetMockTextResponse(
			`{"data": [{"type": "organizations",
			            "id": "o:org",
									"attributes": {"slug": "org"}},
			           {"type": "organizations",
								  "id": "o:org2",
									"attributes": {"slug": "org2"}}]}`,
		),
		org1ProjectsUrl: jsonapi.GetMockTextResponse(
			`{"data": [{
				"type": "projects",
				"id": "o:orgslug:p:projslug",
				"attributes": {"name": "Proj Name", "slug": "projslug"},
				"relationships": {"organization": {
					"data": {"type": "organizations", "id": "o:orgslug"},
					"links": {"related": "/organizations/o:orgslug"}
				}}
			}]}`,
		),
		org2ProjectsUrl: jsonapi.GetMockTextResponse(`{"data": []}`),
	}

	api := jsonapi.GetTestConnection(mockData)
	resource := config.Resource{
		ProjectSlug: "projslug3",
	}
	res, err := getOrganizationSlug(api, &resource)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, res, "")
}

func TestSuccessfulMigration(t *testing.T) {
	var afterTest = func(pkgDir string, tmpDir string) {
		err := os.Chdir(pkgDir)
		if err != nil {
			t.Error(err)
		}
		err = os.RemoveAll(tmpDir)
		if err != nil {
			fmt.Println("Delete error:", err)
		}
	}

	// Requests Data
	org1ProjectsUrl := "/projects?filter%5Borganization%5D=o%3Aorg&" +
		"filter%5Bslug%5D=projslug"
	org2ProjectsUrl := "/projects?filter%5Borganization%5D=o%3Aorg2&" +
		"filter%5Bslug%5D=projslug"
	mockData := jsonapi.MockData{
		"/organizations": jsonapi.GetMockTextResponse(
			`{"data": [{"type": "organizations",
			            "id": "o:org",
									"attributes": {"slug": "org"}},
			           {"type": "organizations",
								  "id": "o:org2",
									"attributes": {"slug": "org2"}}]}`,
		),
		org1ProjectsUrl: jsonapi.GetMockTextResponse(
			`{"data": [{
				"type": "projects",
				"id": "o:orgslug:p:projslug",
				"attributes": {"name": "Proj Name", "slug": "projslug"},
				"relationships": {"organization": {
					"data": {"type": "organizations", "id": "o:orgslug"},
					"links": {"related": "/organizations/o:orgslug"}
				}}
			}]}`,
		),
		org2ProjectsUrl: jsonapi.GetMockTextResponse(`{"data": []}`),
	}

	// Create deprecated config & .transifexrc
	pkgDir, _ := os.Getwd()
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Error(err)
	}
	defer afterTest(pkgDir, tmpDir)

	f, err := os.Create(".transifexrc")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 := f.WriteString(`
		[https://www.transifex.com]
		api_hostname  = https://api.transifex.com
		hostname      = https://www.transifex.com
		username      = api
		password      = apassword
	`)

	if err2 != nil {
		log.Fatal(err2)
	}

	f, err = os.Create("config")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 = f.WriteString(`
		[main]
		host = https://www.transifex.com
		[projslug.ares]
		file_filter = locale/<lang>.po
		minimum_perc = 0
		source_file = locale/en.po
		source_lang = en
		type = PO
		trans.pt-pt = locale/other/pt_PT/en.po
		trans.fr_CA = locale/other/fr_CA/ui.po
	`)
	if err2 != nil {
		log.Fatal(err2)
	}

	// Load for the first time configs
	cfg, err := config.LoadFromPaths(
		filepath.Join(tmpDir, ".transifexrc"), filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Error(err)
	}

	api := jsonapi.GetTestConnection(mockData)

	assert.Equal(t, cfg.GetActiveHost().Name, "https://www.transifex.com")
	assert.Equal(t, cfg.GetActiveHost().Token, "")
	assert.Equal(t, cfg.GetActiveHost().RestHostname, "")
	assert.Equal(t, cfg.Local.Resources[0].OrganizationSlug, "")

	_, _, err = MigrateLegacyConfigFile(&cfg, api)
	if err != nil {
		t.Error(err)
	}

	// Load for the first time configs
	cfgReloaded, err := config.LoadFromPaths(
		filepath.Join(tmpDir, ".transifexrc"), filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, cfgReloaded.GetActiveHost().Name, "https://app.transifex.com")
	assert.Equal(t, cfgReloaded.GetActiveHost().Token, "apassword")
	assert.Equal(t, cfgReloaded.GetActiveHost().RestHostname,
		"https://rest.api.transifex.com")
	assert.Equal(t, cfgReloaded.Local.Resources[0].OrganizationSlug, "org")
	assert.Equal(t, len(cfgReloaded.Local.Resources[0].Overrides), 2)
}

func TestSuccessfulMigrationWithSourceFileConstruction(t *testing.T) {
	var afterTest = func(pkgDir string, tmpDir string) {
		err := os.Chdir(pkgDir)
		if err != nil {
			t.Error(err)
		}
		err = os.RemoveAll(tmpDir)
		if err != nil {
			fmt.Println("Delete error:", err)
		}
	}

	// Requests Data
	org1ProjectsUrl := "/projects?filter%5Borganization%5D=o%3Aorg&" +
		"filter%5Bslug%5D=projslug"
	org2ProjectsUrl := "/projects?filter%5Borganization%5D=o%3Aorg2&" +
		"filter%5Bslug%5D=projslug"
	mockData := jsonapi.MockData{
		"/organizations": jsonapi.GetMockTextResponse(
			`{"data": [{"type": "organizations",
			            "id": "o:org",
									"attributes": {"slug": "org"}},
			           {"type": "organizations",
								  "id": "o:org2",
									"attributes": {"slug": "org2"}}]}`,
		),
		org1ProjectsUrl: jsonapi.GetMockTextResponse(
			`{"data": [{
				"type": "projects",
				"id": "o:orgslug:p:projslug",
				"attributes": {"name": "Proj Name", "slug": "projslug"},
				"relationships": {"organization": {
					"data": {"type": "organizations", "id": "o:orgslug"},
					"links": {"related": "/organizations/o:orgslug"}
				}}
			}]}`,
		),
		org2ProjectsUrl: jsonapi.GetMockTextResponse(`{"data": []}`),
	}

	// Create deprecated config & .transifexrc
	pkgDir, _ := os.Getwd()
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Error(err)
	}
	defer afterTest(pkgDir, tmpDir)

	f, err := os.Create(".transifexrc")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 := f.WriteString(`
		[https://www.transifex.com]
		api_hostname  = https://api.transifex.com
		hostname      = https://www.transifex.com
		username      = api
		password      = apassword
	`)

	if err2 != nil {
		log.Fatal(err2)
	}

	f, err = os.Create("config")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 = f.WriteString(`
		[main]
		host = https://www.transifex.com
		[projslug.ares]
		file_filter = locale/<lang>.po
		minimum_perc = 0
		source_lang = en
		type = PO
		trans.pt-pt = locale/other/pt_PT/en.po
		trans.fr_CA = locale/other/fr_CA/ui.po
	`)
	if err2 != nil {
		log.Fatal(err2)
	}

	// Load for the first time configs
	cfg, err := config.LoadFromPaths(
		filepath.Join(tmpDir, ".transifexrc"), filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Error(err)
	}

	api := jsonapi.GetTestConnection(mockData)

	assert.Equal(t, cfg.GetActiveHost().Token, "")
	assert.Equal(t, cfg.GetActiveHost().RestHostname, "")
	assert.Equal(t, cfg.Local.Resources[0].OrganizationSlug, "")

	_, _, err = MigrateLegacyConfigFile(&cfg, api)
	if err != nil {
		t.Error(err)
	}

	// Load for the first time configs
	cfgReloaded, err := config.LoadFromPaths(
		filepath.Join(tmpDir, ".transifexrc"), filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, cfgReloaded.Local.Resources[0].OrganizationSlug, "org")
	assert.Equal(t, cfgReloaded.Local.Resources[0].SourceFile, "locale/en.po")
	assert.Equal(t, len(cfgReloaded.Local.Resources[0].Overrides), 2)
}

func TestNeedsTokenInRootConfig(t *testing.T) {
	var afterTest = func(pkgDir string, tmpDir string) {
		err := os.Chdir(pkgDir)
		if err != nil {
			t.Error(err)
		}
		err = os.RemoveAll(tmpDir)
		if err != nil {
			fmt.Println("Delete error:", err)
		}
	}
	// Requests Data
	mockData := jsonapi.MockData{
		"/organizations": jsonapi.GetMockTextResponse(`{"data": []}`),
	}

	// Create deprecated config & .transifexrc
	pkgDir, _ := os.Getwd()
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Error(err)
	}
	defer afterTest(pkgDir, tmpDir)

	f, err := os.Create(".transifexrc")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 := f.WriteString(`
		[https://www.transifex.com]
		api_hostname  = https://api.transifex.com
		hostname      = https://www.transifex.com
		username      = tk
		password      = apassword
	`)

	if err2 != nil {
		log.Fatal(err2)
	}

	f, err = os.Create("config")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 = f.WriteString(`
		[main]
		host = https://www.transifex.com
		[projslug.ares]
		file_filter = locale/<lang>.po
		minimum_perc = 0
		source_file = locale/en.po
		source_lang = en
		type = PO
	`)
	if err2 != nil {
		log.Fatal(err2)
	}

	// Load for the first time configs
	cfg, err := config.LoadFromPaths(
		filepath.Join(tmpDir, ".transifexrc"), filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Error(err)
	}

	api := jsonapi.GetTestConnection(mockData)

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, _ = MigrateLegacyConfigFile(&cfg, api)

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	assert.True(t, strings.Contains(string(out), "API token not found."))
}

func TestNoTransifexRcFile(t *testing.T) {
	var afterTest = func(pkgDir string, tmpDir string) {
		err := os.Chdir(pkgDir)
		if err != nil {
			t.Error(err)
		}
		err = os.RemoveAll(tmpDir)
		if err != nil {
			fmt.Println("Delete error:", err)
		}
	}

	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	userTransifexRcPath := filepath.Join(homeDir, ".transifexrc")
	userTransifexRcBackupPath := filepath.Join(homeDir, ".transifexrc.testbak")

	// Check if user's .transifexrc exists and create a backup if needed
	if _, err := os.Stat(userTransifexRcPath); err == nil {
		err = os.Rename(userTransifexRcPath, userTransifexRcBackupPath)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Clean up any ~/.transifexrc that got created during the test, and restore the user's original
	defer func() {
		os.Remove(userTransifexRcPath)
		if _, err = os.Stat(userTransifexRcBackupPath); err == nil {
			err = os.Rename(userTransifexRcBackupPath, userTransifexRcPath)
			if err != nil {
				t.Fatal(err)
			}
		}
	}()

	// Requests Data
	mockData := jsonapi.MockData{
		"/organizations": jsonapi.GetMockTextResponse(`{"data": []}`),
	}

	// Create deprecated config without .transifexrc
	pkgDir, _ := os.Getwd()
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Error(err)
	}
	defer afterTest(pkgDir, tmpDir)

	f, err := os.Create("config")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 := f.WriteString(`
		[main]
		host = https://www.transifex.com
		[projslug.ares]
		file_filter = locale/<lang>.po
		minimum_perc = 0
		source_file = locale/en.po
		source_lang = en
		type = PO
	`)
	if err2 != nil {
		log.Fatal(err2)
	}

	// Load for the first time configs
	cfg, err := config.LoadFromPaths("", filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Error(err)
	}

	api := jsonapi.GetTestConnection(mockData)

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, _ = MigrateLegacyConfigFile(&cfg, api)

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	assert.True(t, strings.Contains(string(out), "Root configuration file not found"))
	assert.True(t, strings.Contains(string(out), "Please provide an API token to continue."))
}

func TestResourceMigrationFailed(t *testing.T) {
	var afterTest = func(pkgDir string, tmpDir string) {
		err := os.Chdir(pkgDir)
		if err != nil {
			t.Error(err)
		}
		err = os.RemoveAll(tmpDir)
		if err != nil {
			fmt.Println("Delete error:", err)
		}
	}
	// Requests Data
	project1Url := "/projects?filter%5Borganization%5D=o%3Aorg&" +
		"filter%5Bslug%5D=projslug"
	project2Url := "/projects?filter%5Borganization%5D=o%3Aorg&" +
		"filter%5Bslug%5D=projslug2"
	org2ProjectsUrl := "/projects?filter%5Borganization%5D=o%3Aorg2&" +
		"filter%5Bslug%5D=projslug"
	mockData := jsonapi.MockData{
		"/organizations": &jsonapi.MockEndpoint{
			Requests: []jsonapi.MockRequest{
				{
					Response: jsonapi.MockResponse{
						Text: `{"data": [
							{"type": "organizations",
							 "id": "o:org",
							 "attributes": {"slug": "org"}},
							{"type": "organizations",
							 "id": "o:org2",
							 "attributes": {"slug": "org2"}}
						]}`,
					},
				},
				{
					Response: jsonapi.MockResponse{
						Text: `{"data": [
							{"type": "organizations",
							 "id": "o:org",
							 "attributes": {"slug": "org"}},
							{"type": "organizations",
							 "id": "o:org2",
							 "attributes": {"slug": "org2"}}
						]}`,
					},
				},
			},
		},
		project1Url: jsonapi.GetMockTextResponse(
			`{"data": [{
				"type": "projects",
				"id": "o:orgslug:p:projslug",
				"attributes": {"name": "Proj Name", "slug": "projslug"},
				"relationships": {"organization": {
					"data": {"type": "organizations", "id": "o:orgslug"},
					"links": {"related": "/organizations/o:orgslug"}
				}}
			}]}`,
		),
		project2Url: jsonapi.GetMockTextResponse(
			`{"data": [{
				"type": "projects",
				"id": "o:orgslug:p:projslug2",
				"attributes": {"name": "Proj Name 2", "slug": "projslug2"},
				"relationships": {"organization": {
					"data": {"type": "organizations", "id": "o:orgslug"},
					"links": {"related": "/organizations/o:orgslug"}
				}}
			}]}`,
		),
		org2ProjectsUrl: jsonapi.GetMockTextResponse(`{"data": []}`),
	}

	// Create deprecated config & .transifexrc
	pkgDir, _ := os.Getwd()
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Error(err)
	}
	defer afterTest(pkgDir, tmpDir)

	f, err := os.Create(".transifexrc")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 := f.WriteString(`
		[https://www.transifex.com]
		api_hostname  = https://api.transifex.com
		hostname      = https://www.transifex.com
		username      = api
		password      = apassword
	`)

	if err2 != nil {
		log.Fatal(err2)
	}

	f, err = os.Create("config")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 = f.WriteString(`
		[main]
		host = https://www.transifex.com
		[projslug1.ares]
		file_filter = locale/<lang>.po
		minimum_perc = 10
		source_file = locale/en.po
		source_lang = en
		type = PO
		[projslug2.ares2]
		file_filter = locale/<lang>.po
		minimum_perc = 0
		source_file = locale/en.po
		source_lang = en
		type = PO
	`)
	if err2 != nil {
		log.Fatal(err2)
	}

	// Load for the first time configs
	cfg, err := config.LoadFromPaths(
		filepath.Join(tmpDir, ".transifexrc"), filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Error(err)
	}

	api := jsonapi.GetTestConnection(mockData)

	assert.Equal(t, cfg.Local.Resources[0].OrganizationSlug, "")
	assert.Equal(t, cfg.Local.Resources[1].OrganizationSlug, "")

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_, _, err = MigrateLegacyConfigFile(&cfg, api)
	if err != nil {
		t.Error(err)
	}

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout
	assert.True(t, strings.Contains(
		string(out), "Could not migrate resource `ares`"))

	content, err := ioutil.ReadFile(filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Error(err)
	}
	assert.True(t, strings.Contains(
		string(content), "projslug1.ares"))
	assert.True(t, strings.Contains(
		string(content), "o:org:p:projslug2:r:ares2"))
	assert.True(t, strings.Contains(
		string(content), "minimum_perc = 10"))
	assert.True(t, strings.Contains(
		string(content), "minimum_perc = 0"))
}

func TestBackUpFilesCreated(t *testing.T) {
	var afterTest = func(pkgDir string, tmpDir string) {
		err := os.Chdir(pkgDir)
		if err != nil {
			t.Error(err)
		}
		err = os.RemoveAll(tmpDir)
		if err != nil {
			fmt.Println("Delete error:", err)
		}
	}

	// Requests Data
	mockData := jsonapi.MockData{
		"/organizations": jsonapi.GetMockTextResponse(
			`{"data": [{"type": "organizations",
			            "id": "o:org",
									"attributes": {"slug": "org"}},
			           {"type": "organizations",
								  "id": "o:org2",
									"attributes": {"slug": "org2"}}]}`,
		),
		"/projects?filter%5Borganization%5D=o%3Aorg&filter%5Bslug%5D=projslug": jsonapi.GetMockTextResponse(
			`{"data": [{
				"type": "projects",
				"id": "o:orgslug:p:projslug",
				"attributes": {"name": "Proj Name", "slug": "projslug"},
				"relationships": {"organization": {
					"data": {"type": "organizations", "id": "o:orgslug"},
					"links": {"related": "/organizations/o:orgslug"}
				}}
			}]}`,
		),
		"/projects?filter%5Borganization%5D=o%3Aorg2&filter%5Bslug%5D=projslug": jsonapi.GetMockTextResponse(`{"data": []}`),
	}

	// Create deprecated config & .transifexrc
	pkgDir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Error(err)
	}
	defer afterTest(pkgDir, tmpDir)

	f, err := os.Create(".transifexrc")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 := f.WriteString(`
		[https://www.transifex.com]
		api_hostname  = https://api.transifex.com
		hostname      = https://www.transifex.com
		username      = api
		password      = apassword
	`)

	if err2 != nil {
		log.Fatal(err2)
	}

	f, err = os.Create("config")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 = f.WriteString(`
		[main]
		host = https://www.transifex.com
		[projslug.ares]
		file_filter = locale/<lang>.po
		minimum_perc = 0
		source_file = locale/en.po
		source_lang = en
		type = PO
	`)
	if err2 != nil {
		log.Fatal(err2)
	}

	// Load for the first time configs
	cfg, err := config.LoadFromPaths(
		filepath.Join(tmpDir, ".transifexrc"), filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Error(err)
	}

	api := jsonapi.GetTestConnection(mockData)

	backupFilePath, backupRootFilePath, _ := MigrateLegacyConfigFile(&cfg, api)

	newLocalContent, err := ioutil.ReadFile(filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Error(err)
	}
	buLocalContent, err := ioutil.ReadFile(filepath.Join(backupFilePath))
	if err != nil {
		t.Error(err)
	}

	newRootContent, err2 := ioutil.ReadFile(filepath.Join(tmpDir, ".transifexrc"))
	if err2 != nil {
		t.Error(err2)
	}
	buRootContent, err2 := ioutil.ReadFile(filepath.Join(backupRootFilePath))
	if err2 != nil {
		t.Error(err2)
	}

	if err != nil {
		t.Errorf("A backup local config file was expected %s", err.Error())
	}

	if err2 != nil {
		t.Errorf("A backup root config file was expected %s", err2.Error())
	}

	assert.True(t, strings.Contains(string(buLocalContent), "[projslug.ares]"))
	assert.True(t, strings.Contains(string(newLocalContent),
		"o:org:p:projslug:r:ares"))
	assert.True(t, strings.Contains(string(buRootContent), "[https://www.transifex.com]"))
	assert.True(t, strings.Contains(string(newRootContent), "[https://app.transifex.com]"))
}
