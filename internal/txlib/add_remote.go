package txlib

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
)

func AddRemoteCommand(
	cfg *config.Config,
	api *jsonapi.Connection,
	projectUrl,
	fileFilter string,
	minimumPerc int,
) error {
	// "/org/proj/whatever..." => ["", "org", "proj", whatever...]
	//                             ↑   ↑      ↑       ↑
	//                             0   1      2       3
	parsed, err := url.Parse(projectUrl)
	if err != nil {
		return fmt.Errorf("invalid project URL '%s'", projectUrl)
	}
	parts := strings.Split(parsed.Path, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid project URL '%s'", projectUrl)
	}
	organizationSlug := parts[1]
	projectSlug := parts[2]

	// Get stuff from API
	project, err := txapi.GetProjectById(
		api,
		fmt.Sprintf("o:%s:p:%s", organizationSlug, projectSlug),
	)
	if err != nil {
		return fmt.Errorf("unable to fetch project: %s", err)
	}
	if project == nil {
		return fmt.Errorf("project not found at '%s'", projectUrl)
	}
	resources, err := txapi.GetResources(api, project)
	if err != nil {
		return fmt.Errorf("unable to fetch resources: %s", err)
	}
	if len(resources) == 0 {
		// Printing instead of returning an error because the command may have been
		// supplied with multiple project URLs
		fmt.Printf("Project at '%s' does not have any resources", projectUrl)
	}
	organization := &jsonapi.Resource{
		API:  api,
		Type: "organizations",
		Id:   fmt.Sprintf("o:%s", organizationSlug),
	}
	i18nFormats, err := txapi.GetI18nFormats(api, organization)
	if err != nil {
		return fmt.Errorf("unable to fetch i18n formats: %s", err)
	}

	for _, resource := range resources {
		// Find i18n format data
		i18nFormatRelationship, exists := resource.Relationships["i18n_format"]
		if !exists {
			return errors.New(
				"resource doest not have an 'i18n_format' relationship",
			)
		}
		i18nFormat, exists := i18nFormats[i18nFormatRelationship.DataSingular.Id]
		if !exists {
			return fmt.Errorf(
				"could not find file Format: %s",
				resource.Relationships["i18n_format"].DataSingular.Id,
			)
		}

		// Construct file-filter
		resourceFileFilter := strings.ReplaceAll(
			fileFilter,
			"<project_slug>",
			projectSlug,
		)

		var resourceAttributes txapi.ResourceAttributes
		resource.MapAttributes(&resourceAttributes)
		resourceFileFilter = strings.ReplaceAll(
			resourceFileFilter,
			"<resource_slug>",
			resourceAttributes.Slug,
		)

		if strings.Contains(resourceFileFilter, "<ext>") {
			var i18nFormatAttributes txapi.I18nFormatsAttributes
			i18nFormat.MapAttributes(&i18nFormatAttributes)
			ext := i18nFormatAttributes.FileExtensions[0][1:]
			resourceFileFilter = strings.ReplaceAll(resourceFileFilter, "<ext>", ext)
		}

		// Construct source file
		sourceLanguageRelationship, exists := project.Relationships["source_language"]
		if !exists {
			return errors.New("project does not have a 'source_language' relationship")
		}
		sourceLanguage := sourceLanguageRelationship.DataSingular
		sourceLanguageCode := sourceLanguage.Id[2:]
		sourceFile := strings.ReplaceAll(
			resourceFileFilter,
			"<lang>",
			sourceLanguageCode,
		)

		// Construct minimum percentage
		if minimumPerc == -1 {
			minimumPerc = 0
		}

		// Add to local config (in RAM, will save to disk later)
		cfg.Local.Resources = append(cfg.Local.Resources, config.Resource{
			OrganizationSlug:  organizationSlug,
			ProjectSlug:       projectSlug,
			ResourceSlug:      resourceAttributes.Slug,
			FileFilter:        resourceFileFilter,
			SourceFile:        sourceFile,
			SourceLanguage:    sourceLanguageCode,
			Type:              i18nFormat.Id,
			MinimumPercentage: minimumPerc,
		})
		fmt.Printf(
			"Added '%s.%s' to configuration\n",
			projectSlug,
			resourceAttributes.Slug,
		)
	}

	return nil
}
