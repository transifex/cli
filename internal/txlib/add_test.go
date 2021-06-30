package txlib

import (
	"os"
	"reflect"
	"testing"

	"github.com/transifex/cli/internal/txlib/config"
)

func TestNoSourceFileErrorAddCommand(t *testing.T) {
	cfg := config.Config{
		Local: &config.LocalConfig{},
		Root:  &config.RootConfig{},
	}
	var args = AddCommandArguments{
		OrganizationSlug: "org",
		ProjectSlug:      "proj",
		ResourceSlug:     "res",
		FileFilter:       "ffilter",
		RType:            "type",
		SourceFile:       "",
	}
	err := AddCommand(&cfg, &args)
	if err == nil {
		t.Errorf("No source file should return an error when trying to add")
	}
}

func TestSuccessfulAddForAddCommand(t *testing.T) {
	testDir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error(err)
	}
	defer func() { os.RemoveAll(tempDir) }()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err = os.Chdir(testDir)
		if err != nil {
			t.Error(err)
		}
	}()

	cfg := config.Config{
		Local: &config.LocalConfig{
			Host: "host",
			Resources: []config.Resource{
				{ProjectSlug: "aaa", ResourceSlug: "bbb"},
				{ProjectSlug: "ccc", ResourceSlug: "ddd"},
			},
			Path: "localconf",
		},
		Root: &config.RootConfig{Path: "rootconf"},
	}
	err = cfg.Local.Save()
	if err != nil {
		t.Error(err)
	}

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
		Host: "host",
		Resources: []config.Resource{
			{ProjectSlug: "aaa", ResourceSlug: "bbb"},
			{ProjectSlug: "ccc", ResourceSlug: "ddd"},
			{
				OrganizationSlug: "org",
				ProjectSlug:      "myproj",
				ResourceSlug:     "res",
				FileFilter:       "f<lang>filter",
				Type:             "type",
				SourceFile:       "mysourcefile.po",
			},
		},
		Path: "localconf",
	}
	if !reflect.DeepEqual(cfg.Local, expected) {
		t.Errorf("Expected addCommand to create %+v and got %+v!",
			expected, cfg.Local)
	}
}
