package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"gopkg.in/ini.v1"
)

// Edits legacy config files so they contain all the necessary information
// to use the 3rd version of the API.
// Steps taken:
// 1. Check for token setting.
//    If not found check for API token in the old configuration.
//    If not found generate one.
// 2. Check for rest_hostname setting. If not found add it.
// 3. Check the section keys are using the legacy format (`<project_slug>.<resource_slug>`)
//    If yes find the organization for each section key and reformat the section key
//    to conform to the new format (o:<organization_slug>:p:<project_slug>:r:<resource_slug>)
func migrateLegacyConfigFile(c *cli.Context) error {
	config := c.App.Metadata["Config"].(*Config)
	rootCfg, _ := ini.Load(c.App.Metadata["RootConfigFilePath"])
	section := rootCfg.Section(config.ActiveHost)
	if config.Token == "" {
		if config.Username == "api" {
			fmt.Printf(
				"Found API token in `%s` file\n",
				c.App.Metadata["RootConfigFilePath"],
			)
			section.NewKey("token", config.Password)
			config.Token = config.Password
		} else {
			// Likely this will change and the user will be prompted to
			// enter the API token since having generate API token endpoints
			// using username:password is questionable.
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("No api token found. Input your API token: ")
			apiToken, _ := reader.ReadString('\n')
			apiToken = strings.Replace(apiToken, "\n", "", -1)
			config.Token = apiToken
			section.NewKey("token", apiToken)
			fmt.Printf("Token saved\n")
		}
		rootCfg.SaveTo(c.App.Metadata["RootConfigFilePath"].(string))
	}
	if config.Hostname == "" {
		fmt.Printf("No rest_hostname found adding `rest.api.transifex.com `\n")
		config.Hostname = "https://rest.api.transifex.com"
		section.NewKey("rest_hostname", config.Hostname)
		rootCfg.SaveTo(c.App.Metadata["RootConfigFilePath"].(string))
	}

	fileMappings := c.App.Metadata["FileMappings"].(map[string]FileMapping)
	for _, fileMapping := range fileMappings {
		if fileMapping.OrganizationSlug == "" {
			resourceID := getResourceID(config, &fileMapping)
			if resourceID == "" {
				fmt.Printf(
					"Could not migrate resource `%s`\n\n", fileMapping.ID,
				)
			} else {
				oldResourceID := fileMapping.ID
				organizationSlug := strings.Split(resourceID, ":")[1]
				fileMapping.OrganizationSlug = organizationSlug
				delete(fileMappings, oldResourceID)
				fileMapping.ID = resourceID
				fileMappings[resourceID] = fileMapping
				configFilePath := c.App.Metadata["ConfigFilePath"].(string)
				read, err := ioutil.ReadFile(configFilePath)
				if err != nil {
					return err
				}
				newContents := strings.Replace(string(read), oldResourceID, resourceID, -1)
				err = ioutil.WriteFile(configFilePath, []byte(newContents), 0)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func getResourceID(config *Config, fileMapping *FileMapping) string {
	client := NewClient(config.Token, config.Hostname)
	// TODO: Maybe handle possible error here?
	organizations, _ := client.getOrganizations()
	for _, organization := range *organizations {
		projectID := organization.ID + ":p:" + fileMapping.ProjectSlug
		_, err := client.getProject(projectID)
		if err == nil {
			return projectID + ":r:" + fileMapping.ResourceSlug
		}
	}
	return ""
}
