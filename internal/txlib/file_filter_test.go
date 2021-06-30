package txlib

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func beforeFileFilterTest(t *testing.T) func() {
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
	return func() {
		err = os.Chdir(curDir)
		if err != nil {
			t.Error(err)
		}
		err := os.RemoveAll(tempDir)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestSearchFileFilterFiles(t *testing.T) {
	afterTest := beforeFileFilterTest(t)
	defer afterTest()

	// <curDir>/
	//   + en.txt
	//   + fr.txt
	//   + de.txt
	for _, langCode := range []string{"en", "fr", "de"} {
		file, err := os.OpenFile(fmt.Sprintf("%s.txt", langCode),
			os.O_RDWR|os.O_CREATE|os.O_TRUNC,
			0755)
		if err != nil {
			t.Error(err)
		}
		defer file.Close()
	}

	curDir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	pathToFile := filepath.Join(curDir, "en.txt")
	actual := searchFileFilter(pathToFile, "")
	expected := map[string]string{"": pathToFile}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got '%+v', expected '%+v'", actual, expected)
	}

	actual = searchFileFilter(curDir, "<lang>.txt")
	expected = map[string]string{
		"en": filepath.Join(curDir, "en.txt"),
		"fr": filepath.Join(curDir, "fr.txt"),
		"de": filepath.Join(curDir, "de.txt"),
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got '%+v', expected '%+v'", actual, expected)
	}
}

func TestSearchFileFilterDirs(t *testing.T) {
	afterTest := beforeFileFilterTest(t)
	defer afterTest()

	// <curDir>/
	//   + en/
	//   | + text.txt
	//   + fr/
	//   | + text.txt
	//   + de/
	//     + text.txt
	for _, langCode := range []string{"en", "fr", "de"} {
		err := os.Mkdir(langCode, os.ModeDir|0755)
		if err != nil {
			t.Error(err)
		}
		file, err := os.OpenFile(
			filepath.Join(langCode, "text.txt"),
			os.O_RDWR|os.O_CREATE|os.O_TRUNC,
			0755)
		if err != nil {
			t.Error(err)
		}
		defer file.Close()
	}

	curDir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	pathToFile := filepath.Join(curDir, "en", "text.txt")
	actual := searchFileFilter(pathToFile, "")
	expected := map[string]string{"": pathToFile}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got '%+v', expected '%+v'", actual, expected)
	}

	actual = searchFileFilter(curDir, filepath.Join("<lang>", "text.txt"))
	expected = map[string]string{
		"en": filepath.Join(curDir, "en", "text.txt"),
		"fr": filepath.Join(curDir, "fr", "text.txt"),
		"de": filepath.Join(curDir, "de", "text.txt"),
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Got '%+v', expected '%+v'", actual, expected)
	}
}
