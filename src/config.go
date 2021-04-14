package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
	"gopkg.in/ini.v1"
)

// Config holds the root configuration
type Config struct {
	ActiveHost string
	Hostname   string
	Username   string
	Password   string
	Token      string
}

// FileMapping holds the file configuration
type FileMapping struct {
	ID                   string
	OrganizationSlug     string
	ProjectSlug          string
	ResourceSlug         string
	FileFilter           string
	SourceFile           string
	SourceLang           string
	FileType             string
	TranslationOverrides map[string]string
	LanguageOverrides    map[string]string
	LanguageMappings     map[string]string
}

// Returns the current working directory as a string.
func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

// Returns the absolute path of current working directory.
func getCurrentWorkingDir() string {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return path
}

// Try to find the config directory '.tx/' working backwards from the
// current working directory. If found it returns the absolute path.
func getConfigDirPath() (string, error) {
	path := getCurrentWorkingDir()
	for {
		configDir, err := os.Stat(filepath.Join(path, ".tx"))
		if os.IsNotExist(err) || !configDir.IsDir() {
			if path == filepath.Dir(path) {
				return "", fmt.Errorf("Cannot find directory: '.tx'")
			}
			path = filepath.Dir(path)
			continue
		}
		return filepath.Join(path, ".tx"), nil
	}
}

func getProjectDir(configDirPath string) string {
	parent := filepath.Dir(configDirPath)
	return parent
}

// Checks if the config file exists in the given configDirPath.
// If found it returns the absolute path.
func getConfigFilePath(configDirPath string) (string, error) {
	configFilePath := filepath.Join(configDirPath, "config")
	configFile, err := os.Stat(configFilePath)
	if os.IsNotExist(err) || configFile.IsDir() {
		return "", fmt.Errorf("Cannot find file: 'config' in the '%s' directory", configDirPath)
	}
	return configFilePath, nil
}

// Checks if the root config file exists.
// The file can exist in the config directory or in the user's home directory.
// Both are checked in that order.
// If found it returns the absolute path.
func getRootConfigFilePath(configDirPath string) (string, error) {
	rootConfPath := filepath.Join(configDirPath, ".transifexrc")
	rcFile, err := os.Stat(rootConfPath)
	if !os.IsNotExist(err) && !rcFile.IsDir() {
		return rootConfPath, nil
	}
	rootConfPath = filepath.Join(getHomeDir(), ".transifexrc")
	rcFile, err = os.Stat(rootConfPath)
	if !os.IsNotExist(err) && !rcFile.IsDir() {
		return rootConfPath, nil
	}
	return "", fmt.Errorf("Cannot find file: '.transifexrc'")
}

// Parses the root config file, creates a Config object
// and adds it to the cli context metadata as "Config"
func loadRootConfig(c *cli.Context, rootConfigFilePath string, activeHost string) error {
	rootCfg, err := ini.Load(rootConfigFilePath)
	if err != nil {
		return fmt.Errorf("Could not parse file: '%s'", rootConfigFilePath)
	}
	hostSection := rootCfg.Section(activeHost)
	token := hostSection.Key("token").String()
	if c.IsSet("token") {
		token = c.String("token")
	}
	hostname := hostSection.Key("rest_hostname").String()
	if c.IsSet("hostname") {
		hostname = c.String("hostname")
	}
	c.App.Metadata["Config"] = &Config{
		ActiveHost: activeHost,
		Hostname:   hostname,
		Username:   hostSection.Key("username").String(),
		Password:   hostSection.Key("password").String(),
		Token:      token,
	}
	return nil
}

// Given a language mapping (`lang_map = pt_PT: pt-pt, pt_BR: pt-br`)
// create a map txLanguageCode -> localLanguageCode
func getLanguageOverrides(langMappings string) map[string]string {
	languageOvverides := make(map[string]string)
	languagePairs := strings.Split(langMappings, ",")
	for _, element := range languagePairs {
		pair := strings.Split(element, ":")
		if len(pair) != 2 {
			continue
		}
		remoteCode := strings.TrimSpace(pair[0])
		localCode := strings.TrimSpace(pair[1])
		if len(remoteCode) == 0 || len(localCode) == 0 {
			continue
		}
		languageOvverides[remoteCode] = localCode
	}
	return languageOvverides
}

// Parses the config file, creates a map[string]FileMapping with the object
// with "<project_slug>.<resource_slug>" as keys
// and adds it to the cli context metadata as "FileMappings".
// In addition it adds the following usefull info in the cli context:
// ProjectDir, RootConfigFilePath, ConfigFilePath
func setMetadata(context *cli.Context) error {
	var configDirPath string
	var err error

	if context.IsSet("config") {
		configDirPath, err = filepath.Abs(context.String("config"))
		if os.IsNotExist(err) {
			return fmt.Errorf("Cannot find directory: '%s'", configDirPath)
		}
		_, err = os.Stat(configDirPath)
		if os.IsNotExist(err) {
			return fmt.Errorf("Cannot find directory: '%s'", configDirPath)
		}
	} else {
		configDirPath, err = getConfigDirPath()
		if err != nil {
			return err
		}
	}
	projectDir := getProjectDir(configDirPath)
	context.App.Metadata["ProjectDir"] = projectDir

	rootConfigFilePath, err := getRootConfigFilePath(configDirPath)
	if err != nil {
		return err
	}
	context.App.Metadata["RootConfigFilePath"] = rootConfigFilePath

	configFilePath, err := getConfigFilePath(configDirPath)
	if err != nil {
		return err
	}
	context.App.Metadata["ConfigFilePath"] = configFilePath

	cfg, err := ini.Load(configFilePath)
	if err != nil {
		return fmt.Errorf("Could not parse file: '%s'", configFilePath)
	}

	mainSection := cfg.Section("main")
	activeHost := mainSection.Key("host").String()
	if err = loadRootConfig(context, rootConfigFilePath, activeHost); err != nil {
		return err
	}
	langMappings := mainSection.Key("lang_map").String()
	globalLanguageOverrides := getLanguageOverrides(langMappings)

	fileMappings := make(map[string]FileMapping)
	for _, section := range cfg.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}
		if section.Name() == "main" {
			continue
		}

		translationOverrides := make(map[string]string)
		for _, key := range section.Keys() {
			if strings.HasPrefix(key.Name(), "trans.") {
				languageCode := strings.TrimPrefix(key.Name(), "trans.")
				translationOverrides[languageCode] = key.String()
			}
		}

		langMappings := section.Key("lang_map").String()
		languageOverrides := getLanguageOverrides(langMappings)
		for txLanguageCode, localLanguageCode := range globalLanguageOverrides {
			if _, ok := languageOverrides[txLanguageCode]; ok {
				continue
			}
			languageOverrides[txLanguageCode] = localLanguageCode
		}

		sourceFilePath := filepath.Join(
			projectDir, section.Key("source_file").String(),
		)
		_, err := os.Stat(sourceFilePath)
		if os.IsNotExist(err) {
			return fmt.Errorf("Could not find source_file: '%s'", sourceFilePath)
		}

		resourceID := section.Name()
		var organizationSlug string
		var projectSlug string
		var resourceSlug string
		if match, _ := regexp.MatchString(ResourceIDRegex, section.Name()); match {
			idParts := strings.Split(section.Name(), ":")
			organizationSlug = idParts[1]
			projectSlug = idParts[3]
			resourceSlug = idParts[5]
		} else {
			idParts := strings.Split(section.Name(), ".")
			projectSlug = idParts[0]
			resourceSlug = idParts[1]
		}
		fileFilter := section.Key("file_filter").String()
		fileMapping := FileMapping{
			ID:                   resourceID,
			OrganizationSlug:     organizationSlug,
			ProjectSlug:          projectSlug,
			ResourceSlug:         resourceSlug,
			FileFilter:           fileFilter,
			SourceFile:           sourceFilePath,
			SourceLang:           section.Key("source_lang").String(),
			FileType:             section.Key("type").String(),
			TranslationOverrides: translationOverrides,
			LanguageOverrides:    languageOverrides,
		}
		languageMappings := getExistingLanuagePaths(projectDir, &fileMapping)
		fileMapping.LanguageMappings = languageMappings
		fileMappings[section.Name()] = fileMapping
	}
	context.App.Metadata["FileMappings"] = fileMappings

	return nil
}
