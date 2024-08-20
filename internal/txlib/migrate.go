package txlib

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
	"os"

	"github.com/transifex/cli/pkg/txapi"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
)

/*
MigrateLegacyConfigFile
Edits legacy config files so they contain all the necessary information
to use the 3rd version of the API.
Steps taken:
1. Update 'host' field in 'main' section of local configuration to use app.transifex.com
2. If root configuration has www.transifex.com, update it to app.transifex.com instead
3. Check for token setting.
   If not found check for API token in the old configuration.
   If not found generate one.
4. Check for rest_hostname setting. If not found add it.
5. Check the section keys are using the legacy format
   (`<project_slug>.<resource_slug>`)
   If yes find the organization for each section key and reformat the
   section key to conform to the new format
   (o:<organization_slug>:p:<project_slug>:r:<resource_slug>)
*/
func MigrateLegacyConfigFile(
	cfg *config.Config, api jsonapi.Connection,
) (string, string, error) {
	// Backup previous file before doing anything

	//Read all the contents of the original config file
	bytesRead, err := ioutil.ReadFile(cfg.Local.Path)
	if err != nil {
		return "", "", fmt.Errorf("aborting, could not read local configuration %w", err)
	}

	//Copy all the contents to the destination file
	currentTime := time.Now()

	backUpFilePath := filepath.Join(filepath.Dir(cfg.Local.Path),
		"config_"+currentTime.Format("20060102150405")+".bak")

	err = ioutil.WriteFile(backUpFilePath, bytesRead, 0755)

	if err != nil {
		return "", "", fmt.Errorf("aborting, could not create backup file %w", err)
	}

	// Also backup the root configuration file, if it exists
	backUpRootFilePath := ""
	rootFileCreated := false
	if _, err = os.Stat(cfg.Root.Path); err == nil {
		bytesRead, err = ioutil.ReadFile(cfg.Root.Path)
		if err != nil {
			return "", "", fmt.Errorf("aborting, could not read root configuration %w", err)
		}
		backUpRootFilePath = filepath.Join(filepath.Dir(cfg.Root.Path),
			".transifexrc_"+currentTime.Format("20060102150405")+".bak")
		err = ioutil.WriteFile(backUpRootFilePath, bytesRead, 0755)
		if err != nil {
			return "", "", fmt.Errorf("aborting, could not create backup file %w", err)
		}
	} else if os.IsNotExist(err) {
		fmt.Printf("Root configuration file not found -- creating it at `%s`.\n", cfg.Root.Path)
		f, err := os.Create(cfg.Root.Path)
		if err != nil {
			return "", "", fmt.Errorf("aborting, could not create root configuration %w", err)
		}
		rootFileCreated = true
		defer f.Close()
	} else {
		return "", "", fmt.Errorf("aborting, could not read root configuration %w", err)
	}

	// Update 'host' field in 'main' section of local config to use app.transifex.com
	cfg.Local.Host = strings.ReplaceAll(cfg.Local.Host, "www.transifex.com", "app.transifex.com")

	// Update existing root config to use app.transifex.com
	for i := range cfg.Root.Hosts {
		host := &cfg.Root.Hosts[i]
		host.Name = strings.ReplaceAll(host.Name, "www.transifex.com", "app.transifex.com")
	}

	// Get the current host
	activeHost := cfg.GetActiveHost()

	if activeHost == nil {
		activeHost = &config.Host{}
		activeHost.Name = "https://app.transifex.com"
		activeHost.RestHostname = ""
		activeHost.Token = ""
	}

	if activeHost.Token == "" {
		if activeHost.Username == "api" {
			// Use the current password as token
			fmt.Printf("Found API token in `%s` file\n", cfg.Root.Path)
			activeHost.Token = activeHost.Password
		} else {
			// No token for some reason get a new one
			if cfg.GetActiveHost() != nil {
				fmt.Println("API token not found. Please provide it and it will " +
					"be saved in '~/.transifexrc'.")
			} else {
				fmt.Println("Please provide an API token to continue.")
			}

			fmt.Println("If you don't have an API token, you can generate " +
				"one in https://app.transifex.com/user/settings/api/")
			fmt.Print("> ")
			var token string
			_, err := fmt.Scanln(&token)
			if err != nil {
				return "", "", err
			}
			activeHost.Token = token
		}
	}

	// Save the new rest url
	if activeHost.RestHostname == "" {
		fmt.Println("No rest_hostname found. Adding `rest.api.transifex.com`")
		activeHost.RestHostname = "https://rest.api.transifex.com"
	}

	// Save the new root config if we created the file
	if rootFileCreated {
		cfg.Root.Hosts = append(cfg.Root.Hosts, *activeHost)
	}

	// Try to update resources currently in config
	// Internally if config finds a resource without ":" it will treat it as
	// a migration, read the resource in a special way and create a temp
	// Resource that has no organizationSlug -> "o::p:<project>:r:<resource>"
	var resources = cfg.Local.Resources

	api.Host = activeHost.RestHostname
	api.Token = activeHost.Token
	for i, resource := range resources {
		if resource.OrganizationSlug == "" {
			organizationSlug, err := getOrganizationSlug(api, &resource)
			if err != nil {
				return "", "", err
			}
			if organizationSlug == "" {
				fmt.Printf(
					"Could not migrate resource `%s`\n\n",
					resource.ResourceSlug,
				)
			} else {
				resource.OrganizationSlug = organizationSlug
				if resource.SourceFile == "" &&
					resource.SourceLanguage != "" &&
					resource.FileFilter != "" {
					resource.SourceFile = strings.ReplaceAll(
						resource.FileFilter, "<lang>", resource.SourceLanguage,
					)
				}
				resources[i] = resource

			}

		}
	}
	cfg.Local.Resources = resources
	err = cfg.Save()
	if err != nil {
		return "", "", fmt.Errorf("%w", err)
	}
	return backUpFilePath, backUpRootFilePath, nil
}

func getOrganizationSlug(
	api jsonapi.Connection, resource *config.Resource,
) (string, error) {

	organizations, _ := txapi.GetOrganizations(&api)

	for _, organization := range organizations {
		project, err := txapi.GetProject(
			&api, organization, resource.ProjectSlug,
		)

		if err == nil && project != nil {
			var orgAttributes txapi.OrganizationAttributes
			err = organization.MapAttributes(&orgAttributes)
			return orgAttributes.Slug, err
		}
	}
	return "", nil
}
