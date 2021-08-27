package txlib

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/assert"
)

func beforeTest() (string, string) {
	pkgDir, _ := os.Getwd()
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}
	_ = os.Chdir(tmpDir)
	return pkgDir, tmpDir
}

func afterTest(pkgDir string, tmpDir string) {
	_ = os.Chdir(pkgDir)
	err := os.RemoveAll(tmpDir)
	if err != nil {
		fmt.Println("Delete error:", err)
	}
}

func TestInitCreateFile(t *testing.T) {
	var pkgDir, tmpDir = beforeTest()
	defer afterTest(pkgDir, tmpDir)

	err := InitCommand()
	if err != nil {
		t.Error(err)
	}

	_, err = os.Stat(filepath.Join(tmpDir, ".tx", "config"))
	if err != nil {
		t.Errorf("Config should exist: %s", err)
	}

}

func TestInitCreateFileContents(t *testing.T) {
	var pkgDir, tmpDir = beforeTest()
	defer afterTest(pkgDir, tmpDir)

	err := InitCommand()
	if err != nil {
		t.Error(err)
	}

	var filePath = filepath.Join(tmpDir, ".tx", "config")
	cfg, _ := config.LoadFromPaths("", filePath)

	res := cfg.Local.Host

	assert.Equal(t, res, "https://www.transifex.com")
}

func TestDoesNotChangeConfigWhenAbort(t *testing.T) {
	var pkgDir, tmpDir = beforeTest()
	defer afterTest(pkgDir, tmpDir)

	err := InitCommand()
	if err != nil {
		t.Error(err)
	}

	var filePath = filepath.Join(tmpDir, ".tx", "config")
	cfg, _ := config.LoadFromPaths("", filePath)

	// Add a Resource to check if the file is the same after init cancellation
	var args = AddCommandArguments{
		OrganizationSlug: "org",
		ProjectSlug:      "myproj",
		ResourceSlug:     "res",
		FileFilter:       "f<lang>filter",
		RType:            "type",
		SourceFile:       "mysourcefile.po",
	}
	err = AddCommand(&cfg, &args)
	if err != nil {
		t.Error(err)
	}

	expected := &config.LocalConfig{
		Resources: []config.Resource{
			{
				OrganizationSlug: "org",
				ProjectSlug:      "myproj",
				ResourceSlug:     "res",
				FileFilter:       "f<lang>filter",
				Type:             "type",
				SourceFile:       "mysourcefile.po",
			},
		}}

	err = InitCommand()
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(cfg.Local.Resources, expected.Resources) {
		t.Errorf("Expected config not to be changed: %+v and got %+v!",
			expected.Resources, cfg.Local.Resources)
	}
}
