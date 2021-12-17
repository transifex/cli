package txlib

import (
	"fmt"
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
	afterTest := beforeAddTest(t, nil, nil)
	defer afterTest()

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
	err := cfg.Local.Save()
	if err != nil {
		t.Error(err)
	}

	var args = AddCommandArguments{
		OrganizationSlug: "org",
		ProjectSlug:      "myproj",
		ResourceSlug:     "res",
		FileFilter:       "aaa<lang>.json",
		RType:            "type",
		SourceFile:       "aaa.json",
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
				FileFilter:       "aaa<lang>.json",
				Type:             "type",
				SourceFile:       "aaa.json",
			},
		},
		Path: "localconf",
	}
	if !reflect.DeepEqual(cfg.Local, expected) {
		t.Errorf("Expected addCommand to create %+v and got %+v!",
			expected, cfg.Local)
	}
}

func beforeAddTest(t *testing.T,
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
