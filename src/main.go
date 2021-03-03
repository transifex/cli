package main

import (
	"encoding/json"
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
	Hostname string
	Username string
	Password string
	Token    string
}

type FileMapping struct {
	Name                 string
	FileFilter           string
	SourceFile           string
	SourceLang           string
	FileType             string
	TranslationOverrides map[string]string
}

func setMetadata(context *cli.Context) error {
	var configDirPath string
	var err error

	if context.IsSet("config") {
		configDirPath = context.String("config")
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
			hostSection := rootCfg.Section(hostKey)
			context.App.Metadata["Config"] = Config{
				Hostname: hostSection.Key("hostname").String(),
				Username: hostSection.Key("username").String(),
				Password: hostSection.Key("password").String(),
				Token:    "",
			}
			continue
		}

		translationOverrides := make(map[string]string)
		for _, key := range section.Keys() {
			if strings.HasPrefix(key.Name(), "trans.") {
				translationOverrides[key.Name()] = key.String()
			}
		}

		fileMappings[section.Name()] = FileMapping{
			Name:                 section.Name(),
			FileFilter:           section.Key("file_filter").String(),
			SourceFile:           section.Key("source_file").String(),
			SourceLang:           section.Key("source_lang").String(),
			FileType:             section.Key("type").String(),
			TranslationOverrides: translationOverrides,
		}
	}
	context.App.Metadata["FileMappings"] = fileMappings

	return nil
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
		_, err := os.Stat(filepath.Join(path, ".tx"))
		if os.IsNotExist(err) {
			if path == filepath.Dir(path) {
				return "", fmt.Errorf("Cannot find dir: '.tx'")
			}
			path = filepath.Dir(path)
			continue
		}
		return filepath.Join(path, ".tx"), nil
	}
}

func getConfigFilePath(configDirPath string) (string, error) {
	configFilePath := filepath.Join(configDirPath, "config")
	_, err := os.Stat(configFilePath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("Cannot find file: 'config' in the '%s' directory", configDirPath)
	}
	return configFilePath, nil
}

func getRootConfigFilePath(configDirPath string) (string, error) {
	rootConfPath := filepath.Join(configDirPath, ".transifexrc")
	_, err := os.Stat(rootConfPath)
	if !os.IsNotExist(err) {
		return rootConfPath, nil
	}
	rootConfPath = filepath.Join(getHomeDir(), ".transifexrc")
	_, err = os.Stat(rootConfPath)
	if !os.IsNotExist(err) {
		return rootConfPath, nil
	}
	return "", fmt.Errorf("Cannot find file: '.transifexrc'")
}

func main() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			//Value:   filepath.Join(getCurrentWorkingDir(), ".tx/config"),
			Usage: "Load configuration from `FILE`",
		},
	}

	app := &cli.App{
		Before: setMetadata,
		Commands: []*cli.Command{
			{
				Name:    "showconf",
				Aliases: []string{"sc"},
				Usage:   "Print the active configuration",
				Action: func(c *cli.Context) error {
					fmt.Printf("Root config file: %s\n", c.App.Metadata["RootConfigFilePath"])
					fmt.Printf("Config file : %s\n", c.App.Metadata["ConfigFilePath"])
					configJSON, _ := json.MarshalIndent(c.App.Metadata["Config"], "", "  ")
					fmt.Printf("Config:\n%s\n\n", string(configJSON))

					// I wanted to see how to transform an interface to a map
					fileMappings := c.App.Metadata["FileMappings"].(map[string]FileMapping)
					// Now for example you could delete a key since it is a map
					// delete(fileMappings, "DEFAULT")
					fileMappingsJSON, _ := json.MarshalIndent(fileMappings, "", "  ")
					fmt.Printf("FileMappings:\n%s\n", string(fileMappingsJSON))
					return nil
				},
			},
			{
				Name:    "complete",
				Aliases: []string{"c"},
				Usage:   "complete a task on the list",
				Action: func(c *cli.Context) error {
					fmt.Println("completed task: ", c.Args().First())
					return nil
				},
			},
			{
				Name:    "template",
				Aliases: []string{"t"},
				Usage:   "options for task templates",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "add a new template",
						Action: func(c *cli.Context) error {
							fmt.Println("new task template: ", c.Args().First())
							return nil
						},
					},
					{
						Name:  "remove",
						Usage: "remove an existing template",
						Action: func(c *cli.Context) error {
							fmt.Println("removed task template: ", c.Args().First())
							return nil
						},
					},
				},
			},
		},
		Flags: flags,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
