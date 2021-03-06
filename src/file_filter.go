package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func getPathForLanguage(projectDir string, expression string, languageCode string) string {
	languagePath := strings.Replace(expression, "<lang>", languageCode, -1)
	return filepath.Join(projectDir, languagePath)
}

func getExistingLanuagePaths(projectDir string, expression string) map[string]string {
	languageRegex := strings.Replace(expression, "<lang>", "([^\\/]+?)", -1)
	re, err := regexp.Compile(languageRegex)
	if err != nil {
		return make(map[string]string)
	}
	filePaths, err := searchDir(re, projectDir)
	if err != nil {
		return make(map[string]string)
	}
	languagePaths := make(map[string]string)
	for _, filePath := range filePaths {
		matches := re.FindStringSubmatch(filePath)
		if len(matches) != 2 {
			continue
		}
		languagePaths[matches[1]] = filePath

	}
	return languagePaths
}

func searchDir(re *regexp.Regexp, dir string) ([]string, error) {
	files := []string{}

	walk := func(fn string, fileInfo os.FileInfo, err error) error {
		if re.MatchString(fn) == false {
			return nil
		}
		if !fileInfo.IsDir() {
			files = append(files, fn)
		}
		return nil
	}
	err := filepath.Walk(dir, walk)
	if err != nil {
		return files, err
	}

	return files, nil
}
