package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"gopkg.in/ini.v1"
)

type Config struct {
	RestHostname string
	Username     string
	Password     string
	Token        string
}

type FileMapping struct {
	Name                 string
	FileFilter           string
	SourceFile           string
	SourceLang           string
	FileType             string
	TranslationOverrides map[string]string
	LanguageOverrides    map[string]string
	LanguageMappings     map[string]string
}

func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

func getCurrentWorkingDir() string {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return path
}

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

func getConfigFilePath(configDirPath string) (string, error) {
	configFilePath := filepath.Join(configDirPath, "config")
	configFile, err := os.Stat(configFilePath)
	if os.IsNotExist(err) || configFile.IsDir() {
		return "", fmt.Errorf("Cannot find file: 'config' in the '%s' directory", configDirPath)
	}
	return configFilePath, nil
}

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
	rootCfg, err := ini.Load(rootConfigFilePath)
	if err != nil {
		return fmt.Errorf("Could not parse file: '%s'", rootConfigFilePath)
	}
	fileMappings := make(map[string]FileMapping)

	for _, section := range cfg.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}
		if section.Name() == "main" {
			hostKey := section.Key("host").String()
			context.App.Metadata["ActiveHost"] = hostKey
			hostSection := rootCfg.Section(hostKey)
			var token string
			token = hostSection.Key("token").String()
			if context.IsSet("token") {
				token = context.String("token")
			}
			context.App.Metadata["Config"] = &Config{
				RestHostname: hostSection.Key("rest_hostname").String(),
				Username:     hostSection.Key("username").String(),
				Password:     hostSection.Key("password").String(),
				Token:        token,
			}
			continue
		}

		translationOverrides := make(map[string]string)
		for _, key := range section.Keys() {
			if strings.HasPrefix(key.Name(), "trans.") {
				languageCode := strings.TrimPrefix(key.Name(), "trans.")
				translationOverrides[languageCode] = key.String()
			}
		}

		languageOvverides := make(map[string]string)
		languagePairs := strings.Split(section.Key("lang_map").String(), ",")
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

		fileFilter := section.Key("file_filter").String()
		languageMappings := getExistingLanuagePaths(projectDir, fileFilter)

		for languageCode, languagePath := range translationOverrides {
			languagePath = filepath.Join(projectDir, languagePath)
			_, err := os.Stat(languagePath)
			if !os.IsNotExist(err) {
				languageMappings[languageCode] = languagePath
			}
		}

		sourceFilePath := filepath.Join(
			projectDir, section.Key("source_file").String(),
		)
		_, err := os.Stat(sourceFilePath)
		if os.IsNotExist(err) {
			return fmt.Errorf("Could not find source_file: '%s'", sourceFilePath)
		}

		fileMappings[section.Name()] = FileMapping{
			Name:                 section.Name(),
			FileFilter:           fileFilter,
			SourceFile:           sourceFilePath,
			SourceLang:           section.Key("source_lang").String(),
			FileType:             section.Key("type").String(),
			TranslationOverrides: translationOverrides,
			LanguageOverrides:    languageOvverides,
			LanguageMappings:     languageMappings,
		}
	}
	context.App.Metadata["FileMappings"] = fileMappings

	return nil
}

func formatConfigFile(c *cli.Context) error {
	config := c.App.Metadata["Config"].(*Config)
	rootCfg, _ := ini.Load(c.App.Metadata["RootConfigFilePath"])
	section := rootCfg.Section(c.App.Metadata["ActiveHost"].(string))
	if config.Token == "" {
		if config.Username == "api" {
			fmt.Printf(
				"Found old configuration editing `%s` file\n\n",
				c.App.Metadata["RootConfigFilePath"],
			)
			section.NewKey("token", config.Password)
		} else {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("No api token found. Generate one from transifex?\n")
			fmt.Printf("Type `yes` to continue\n")
			fmt.Printf("-> ")
			text, _ := reader.ReadString('\n')
			text = strings.Replace(text, "\n", "", -1)
			if strings.Compare("yes", text) != 0 {
				return cli.Exit("Aborting...", 0)
			}
			fmt.Printf("Not implemented. Adding test token\n")
			config.Token = "TestToken"
		}
		rootCfg.SaveTo(c.App.Metadata["RootConfigFilePath"].(string))
	}
	if config.RestHostname == "" {
		fmt.Printf("No rest_hostname found adding `rest-api.transifex.com`\n")
		config.RestHostname = "https://rest-api.transifex.com"
		section.NewKey("rest_hostname", config.RestHostname)
		rootCfg.SaveTo(c.App.Metadata["RootConfigFilePath"].(string))
	}
	return nil
}
