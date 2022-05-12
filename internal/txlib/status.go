package txlib

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
)

type StatusCommandArguments struct {
	ResourceIds []string
}

func StatusCommand(
	cfg *config.Config,
	api jsonapi.Connection,
	arguments *StatusCommandArguments,
) error {
	var cfgResources []config.Resource

	pterm.Info.Println("Gathering data for resources")

	for _, resourceId := range arguments.ResourceIds {
		// Find Resources for delete in config
		cfgResource := cfg.FindResource(resourceId)
		if cfgResource == nil {
			return fmt.Errorf(
				"could not find resource '%s' in local configuration.",
				resourceId,
			)
		}
        cfgResources = append(cfgResources, *cfgResource)
	}

	cfgResourcesLen := len(cfgResources)

	if cfgResourcesLen == 0 {
		// cfgResources = append(cfgResources, cfg.Local.Resources)
		cfgResources = cfg.Local.Resources
		cfgResourcesLen = len(cfgResources)
	}
	// If there are no resources found stop
	if cfgResourcesLen == 0 {
		pterm.Error.Println("Given resources not found in config file.")
		return nil
	}

	for i, cfgResource := range cfgResources {
		sourceLang, err := getSourceLanguage(cfg, &api, &cfgResource)
		if err != nil {
			fmt.Print(err)
		}

		fmt.Printf("%s -> %s (%d of %d)\n", cfgResource.ProjectSlug,
			cfgResource.ResourceSlug, i+1, cfgResourcesLen)
		localLanguages := searchFileFilter(".", cfgResource.FileFilter)
		overrides := cfgResource.Overrides
		if len(overrides) > 0 {
			for langOverride := range overrides {
				localLanguages[langOverride] = overrides[langOverride]
			}
		}
		for language := range localLanguages {
			source := ""
			if sourceLang == language {
				source = " (source)"
			}
			fmt.Printf("- %s: %s %s\n", language,
				localLanguages[language], source)
		}
	}
	return nil
}

func getSourceLanguage(
	cfg *config.Config,
	api *jsonapi.Connection,
	cfgResource *config.Resource,
) (string, error) {

	if cfgResource.SourceLanguage != "" {
		return cfgResource.SourceLanguage, nil
	}

	msg := fmt.Sprintf("Fetching information for '%s'",
		cfgResource.OrganizationSlug)

	spinner, err := pterm.DefaultSpinner.Start(msg)

	if err != nil {
		return "", err
	}
	organization, err := txapi.GetOrganization(api,
		cfgResource.OrganizationSlug)
	if err != nil {
		spinner.Fail(msg + ": " + err.Error())
		return "", err
	}

	if organization == nil {
		err = fmt.Errorf("%s: Not found", msg)
		spinner.Fail(err)
		return "", err
	}

	project, err := txapi.GetProject(api, organization,
		cfgResource.ProjectSlug)
	if err != nil {
		spinner.Fail(msg + ": " + err.Error())
		return "", err
	}
	sourceLanguageRelationship, err := project.Fetch("source_language")
	if err != nil {
		spinner.Fail(msg + ": " + err.Error())
		return "", err
	}
	sourceLanguage := sourceLanguageRelationship.DataSingular
	var sourceLanguageAttributes txapi.LanguageAttributes
	err = sourceLanguage.MapAttributes(&sourceLanguageAttributes)

	if err != nil {
		spinner.Fail(msg + ": " + err.Error())
		return "", err
	}
	spinner.Success(
		fmt.Sprintf("Got information for %s", cfgResource.ResourceSlug),
	)

	return sourceLanguageAttributes.Code, nil
}
