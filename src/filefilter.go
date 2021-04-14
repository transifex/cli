package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Find all exsting translation files.
// To do this use the file filter regex.
// After extracting all the files use the translation overrides
// to override / find the remaining files.
// NOTE: The keys of the map refer to the local language codes.
func getExistingLanuagePaths(projectDir string, fileMapping *FileMapping) map[string]string {
	languageRegex := strings.Replace(
		fileMapping.FileFilter, "<lang>", "([^\\/]+?)", -1,
	)

	languagePaths := make(map[string]string)
	re, err := regexp.Compile(languageRegex)
	if err != nil {
		return languagePaths
	}

	filePaths, err := searchDir(re, projectDir)
	if err != nil {
		return languagePaths
	}

	for _, filePath := range filePaths {
		matches := re.FindStringSubmatch(filePath)
		if len(matches) != 2 {
			continue
		}
		languagePaths[matches[1]] = filePath
	}

	// Ensure language overrides take precedence over files found
	// via the language filter
	for localLanguageCode, filePath := range fileMapping.TranslationOverrides {
		filePath = filepath.Join(projectDir, filePath)
		rcFile, err := os.Stat(filePath)
		if !os.IsNotExist(err) && !rcFile.IsDir() {
			languagePaths[localLanguageCode] = filePath
		} else {
			delete(languagePaths, localLanguageCode)
		}
	}
	// TODO: If the found path clashes with the LanguageMappings maybe remove it?
	// TODO: Random non language file location tha matches the regex? Should we validate?
	return languagePaths
}

// Takes a transifex language and converts it to a local language code
func getLocalLanguageCode(txLanguageCode string, fileMapping *FileMapping) string {
	if val, ok := fileMapping.LanguageOverrides[txLanguageCode]; ok {
		return val
	}
	return txLanguageCode
}

// Takes a local language and converts it to a transifex language code
func getTxLanguageCode(localLanguageCode string, fileMapping *FileMapping) string {
	reverseLanguageOverrides := make(map[string]string)
	for k, v := range fileMapping.LanguageOverrides {
		reverseLanguageOverrides[v] = k
	}
	if val, ok := reverseLanguageOverrides[localLanguageCode]; ok {
		return val
	}
	return localLanguageCode
}

// Given the local language code return the absoloute path to the file
// even if the file does not currently exist
func getLanguagePath(localLanguageCode string, projectDir string, fileMapping *FileMapping) string {
	if val, ok := fileMapping.TranslationOverrides[localLanguageCode]; ok {
		return filepath.Join(projectDir, val)
	}
	return filepath.Join(projectDir, strings.Replace(fileMapping.FileFilter, "<lang>", localLanguageCode, -1))
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
