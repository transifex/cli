package txlib

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
)

type DeleteCommandArguments struct {
	ResourceIds []string
	Force       bool
	Skip        bool
}

func DeleteCommand(
	cfg *config.Config,
	api jsonapi.Connection,
	arguments *DeleteCommandArguments,
) error {
	var cfgResources []*config.Resource

	pterm.Info.Println("Initiating Delete")

	for _, resourceId := range arguments.ResourceIds {

		// Split resource id to check if it is a bulk delete
		parts := strings.Split(resourceId, ".")
		if len(parts) != 2 {
			if !arguments.Skip {
				return fmt.Errorf(
					"Wrong resource id for %s. Aborting",
					resourceId,
				)
			} else {
				fmt.Printf(
					"Wrong resource id for %s. Aborting",
					resourceId,
				)
			}
		}

		projectSlug := parts[0]
		resourceSlug := parts[1]

		// Find Resources for delete in config
		if resourceSlug != "*" {
			cfgResource := cfg.FindResource(resourceId)
			if cfgResource == nil {
				if !arguments.Skip {
					return fmt.Errorf(
						"could not find resource '%s' in local configuration. Aborting",
						resourceId,
					)
				} else {
					fmt.Printf(
						"could not find resource '%s' in local configuration.",
						resourceId,
					)
					break
				}
			}
			cfgResources = append(cfgResources, cfgResource)
		} else {
			batchedResources := cfg.FindResourcesByProject(projectSlug)
			cfgResources = append(cfgResources, batchedResources...)
		}

	}
	// If there are no resources found stop
	if len(cfgResources) == 0 {
		pterm.Error.Println("Given resources not found in config file.")
		return nil
	}

	// Delete each resource
	for _, cfgResource := range cfgResources {
		// Delete Resource from Server
		err := deleteResource(&api, cfg, *cfgResource, *arguments)
		if err != nil {
			if !arguments.Skip {
				return err
			} else {
				pterm.Error.Println("Given resources not found in config file.")
			}
		} else {
			// Remove successful deletes from config
			cfg.RemoveResource(*cfgResource)
			err := cfg.Save()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func deleteResource(
	api *jsonapi.Connection, cfg *config.Config, cfgResource config.Resource,
	args DeleteCommandArguments,
) error {

	// Get Organization from Server
	organization, err := txapi.GetOrganization(api,
		cfgResource.OrganizationSlug)
	if err != nil {
		return err
	}

	if organization == nil {
		return fmt.Errorf("Organization '%s' not found",
			cfgResource.OrganizationSlug)
	}

	// Get Project from Server
	project, err := txapi.GetProject(api, organization,
		cfgResource.ProjectSlug)
	if err != nil {
		return err
	}

	if project == nil {
		return fmt.Errorf("Project '%s - %s' not found",
			cfgResource.OrganizationSlug,
			cfgResource.ProjectSlug)

	}

	// Get Resource from Server
	resource, err := txapi.GetResource(api, project, cfgResource.ResourceSlug)

	if err != nil {
		return err
	}

	if resource == nil {
		return fmt.Errorf("Resource '%s - %s - %s' not found",
			cfgResource.OrganizationSlug,
			cfgResource.ProjectSlug,
			cfgResource.ResourceSlug)
	}

	msg := fmt.Sprintf("Deleting resource '%s'",
		cfgResource.ResourceSlug)

	spinner, err := pterm.DefaultSpinner.Start(msg)

	remoteStats, err := txapi.GetResourceStats(api, resource, nil)
	if args.Force != true {
		for i := range remoteStats {
			var remoteStatAttributes txapi.ResourceLanguageStatsAttributes
			err := remoteStats[i].MapAttributes(&remoteStatAttributes)
			if err != nil {
				spinner.Fail(err)
				return err
			}
			if remoteStatAttributes.TranslatedStrings > 0 {
				spinner.Fail(err)
				return fmt.Errorf("Aborting due to translations in %s",
					cfgResource.ResourceSlug)
			}
		}
	}

	err = txapi.DeleteResource(api, resource)

	if err != nil {
		spinner.Fail(
			fmt.Sprintf("Resource deletion for '%s' failed",
				cfgResource.ResourceSlug),
		)
		return err
	} else {
		spinner.Success(
			fmt.Sprintf("Resource '%s' deleted", cfgResource.ResourceSlug),
		)
	}
	return nil
}
