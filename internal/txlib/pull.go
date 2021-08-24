package txlib

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/transifex/cli/pkg/txapi"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
)

type PullCommandArguments struct {
	FileType          string
	Mode              string
	ContentEncoding   string
	Force             bool
	Skip              bool
	Languages         []string
	Source            bool
	Translations      bool
	All               bool
	DisableOverwrite  bool
	ResourceIds       []string
	UseGitTimestamps  bool
	Branch            string
	MinimumPercentage int
}

type CreateAsyncDownloadArguments struct {
	CommandArgs *PullCommandArguments
	Project     *jsonapi.Resource
	//ResourceName string
	CfgResource *config.Resource
}

func PullCommand(
	cfg *config.Config,
	api jsonapi.Connection,
	arguments *PullCommandArguments,
) error {
	if arguments.Branch == "-1" {
		arguments.Branch = ""
	} else if arguments.Branch == "" {
		arguments.Branch = getGitBranch()
		if arguments.Branch == "" {
			pterm.Warning.Println("Couldn't find branch information")
		}
	}
	if arguments.Branch != "" {
		pterm.Info.Printf("Using branch '%s'\n", arguments.Branch)
	}
	var cfgResources map[string]*config.Resource
	if arguments.ResourceIds != nil && len(arguments.ResourceIds) != 0 {
		cfgResources = make(map[string]*config.Resource,
			len(arguments.ResourceIds))
		for _, resourceId := range arguments.ResourceIds {
			cfgResource := cfg.FindResource(resourceId)
			if cfgResource == nil {
				return fmt.Errorf(
					"could not find resource '%s' in local configuration",
					resourceId,
				)
			}
			if arguments.Branch != "" {
				key := fmt.Sprintf("%s--%s",
					cfgResource.Name(), arguments.Branch)
				cfgResources[key] = cfgResource
			} else {
				cfgResources[cfgResource.Name()] = cfgResource
			}
		}
	} else {
		cfgResources = make(map[string]*config.Resource,
			len(cfg.Local.Resources))
		for i := range cfg.Local.Resources {
			cfgResource := &cfg.Local.Resources[i]
			if arguments.Branch != "" {
				key := fmt.Sprintf("%s--%s",
					cfgResource.Name(), arguments.Branch)
				cfgResources[key] = cfgResource
			} else {
				cfgResources[cfgResource.Name()] = cfgResource
			}
		}
	}

	for i := range cfgResources {
		cfgResource := cfgResources[i]
		if arguments.Branch != "" {
			cfgResource.ResourceSlug = fmt.Sprintf("%s--%s",
				cfgResource.ResourceSlug,
				arguments.Branch)
		}
	}

	for _, cfgResource := range cfgResources {
		err := createPullResource(cfg, api, arguments, cfgResource)
		if err != nil {
			if !arguments.Skip {
				return err
			}
		}
	}

	return nil
}

func createPullResource(
	cfg *config.Config,
	api jsonapi.Connection,
	arguments *PullCommandArguments,
	cfgResource *config.Resource,
) error {
	pterm.DefaultSection.Printf("Resource %s", cfgResource.Name())
	org, err := fetchOrganization(cfgResource, api)
	if err != nil {
		return err
	}

	project, err := fetchProject(cfgResource, api, org)
	if err != nil {
		return err
	}

	asyncDownloadArgs := CreateAsyncDownloadArguments{
		CommandArgs: arguments,
		CfgResource: cfgResource,
		Project:     project,
	}

	if arguments.Source {
		fmt.Println("Downloading source files")
		// Downloads source file
		err := createResourceStringsAsyncDownloads(
			api,
			&asyncDownloadArgs,
		)
		if err != nil {
			return err
		}
	}
	if arguments.Translations || !arguments.Source {
		fmt.Println("Downloading translation files")
		// Default functionality is to download translations only
		err := createTranslationsAsyncDownloads(cfg, api, &asyncDownloadArgs)
		if err != nil {
			return err
		}
	}
	return nil
}

func fetchProject(
	cfgResource *config.Resource,
	api jsonapi.Connection,
	org *jsonapi.Resource,
) (*jsonapi.Resource, error) {
	msg := fmt.Sprintf("Fetching project '%s'", cfgResource.ProjectSlug)
	spinner, err := pterm.DefaultSpinner.Start(msg)
	if err != nil {
		return nil, err
	}
	project, err := txapi.GetProject(&api, org, cfgResource.ProjectSlug)
	if err != nil {
		spinner.Fail(msg + ": " + err.Error())
		return nil, err
	}
	if project == nil {
		err = fmt.Errorf("%s: Not found", msg)
		spinner.Fail(err)
		return nil, err
	}
	spinner.Success(
		fmt.Sprintf("Project '%s' fetched", cfgResource.ProjectSlug),
	)
	return project, nil
}

func fetchOrganization(
	cfgResource *config.Resource, api jsonapi.Connection,
) (*jsonapi.Resource, error) {
	msg := fmt.Sprintf("Fetching organization '%s'",
		cfgResource.OrganizationSlug)
	spinner, err := pterm.DefaultSpinner.Start(msg)
	if err != nil {
		return nil, err
	}

	org, err := txapi.GetOrganization(&api, cfgResource.OrganizationSlug)
	if err != nil {
		spinner.Fail(msg + ": " + err.Error())
		return nil, err
	}
	if org == nil {
		err = fmt.Errorf("%s: Not found", msg)
		spinner.Fail(err)
		return nil, err
	}
	spinner.Success(
		fmt.Sprintf("Organization '%s' fetched", cfgResource.OrganizationSlug),
	)
	return org, err
}

func createTranslationsAsyncDownloads(cfg *config.Config,
	api jsonapi.Connection,
	arguments *CreateAsyncDownloadArguments) error {

	project := arguments.Project
	cfgResource := arguments.CfgResource
	commandArgs := arguments.CommandArgs
	var targetLanguages map[string]*jsonapi.Resource

	resource, err := fetchResource(api, cfgResource, project)
	if err != nil {
		return err
	}

	msg := "Retrieving target languages"
	spinner, err := pterm.DefaultSpinner.Start(msg)
	if err != nil {
		return err
	}
	projectLanguages, err := txapi.GetProjectLanguages(project)
	if err != nil {
		spinner.Fail(msg + ": " + err.Error())
		return err
	}

	if !commandArgs.All {
		if len(commandArgs.Languages) > 0 {
			targetLanguages = make(map[string]*jsonapi.Resource,
				len(commandArgs.Languages))
			for _, key := range commandArgs.Languages {
				if key == cfgResource.SourceLanguage {
					pterm.Warning.Printf(
						"User defined language '%s' is source language. Skipping\n",
						key,
					)
					continue
				}
				language, exists := projectLanguages[key]
				if exists {
					targetLanguages[key] = language
				} else {
					// Skip non existing languages but do not terminate the
					// process
					pterm.Warning.Printf(
						"User defined language '%s' doesn't belong to "+
							"project '%s'\n",
						key, project.Id,
					)
				}
			}
		} else {
			allLocalLanguages := searchFileFilter(".", cfgResource.FileFilter)
			if arguments.CommandArgs.FileType == "xliff" {
				addAdditionalLocalLanguages(
					cfgResource, &allLocalLanguages, "xlf",
				)
			} else if arguments.CommandArgs.FileType == "json" {
				addAdditionalLocalLanguages(
					cfgResource, &allLocalLanguages, "json",
				)
			}

			targetLanguages = make(map[string]*jsonapi.Resource,
				len(allLocalLanguages))
			for local := range allLocalLanguages {
				key := getTxLanguageCode(
					cfg.Local.LanguageMappings, local, cfgResource,
				)
				if key == cfgResource.SourceLanguage {
					pterm.Warning.Printf(
						"Language '%s' is source language. Skipping...\n",
						key,
					)
					continue
				}
				language, exists := projectLanguages[key]
				if exists {
					targetLanguages[key] = language
				} else {
					// Skip non existing languages but do not terminate the process
					pterm.Warning.Printf(
						"User defined language '%s' doesn't belong to "+
							"project '%s'\n",
						key, project.Id)
				}
			}
		}
	} else {
		targetLanguages = projectLanguages
	}

	if len(targetLanguages) > 0 {
		spinner.Success("Target languages fetched")
	} else {
		spinner.Fail("No target languages found")
	}

	for lang, language := range targetLanguages {
		helper := txapi.CreateDownloadArguments{
			OrganizationSlug: cfgResource.OrganizationSlug,
			ProjectSlug:      cfgResource.ProjectSlug,
			ResourceSlug:     cfgResource.ResourceSlug,
			Resource:         resource,
			Language:         language,
			FileType:         commandArgs.FileType,
			Mode:             commandArgs.Mode,
			ContentEncoding:  commandArgs.ContentEncoding,
		}

		localLanguageCode, _ := txapi.CreateLanguageDirectory(
			cfg.Local.LanguageMappings, lang, cfgResource,
		)
		translationFile := strings.ReplaceAll(
			cfgResource.FileFilter, "<lang>", localLanguageCode,
		)
		translationFile = setFileTypeExtensions(commandArgs, translationFile)

		msg := "Downloading translation file " + translationFile
		spinner, err := pterm.DefaultSpinner.Start(msg)
		if err != nil {
			return err
		}

		_, err = os.Stat(translationFile)
		if err == nil && commandArgs.DisableOverwrite {
			spinner.Warning(fmt.Sprintf(
				"Disable Overwrite enabled. Skip downloading translation '%s'",
				translationFile,
			))
			continue
		}

		download, err := txapi.CreateTranslationsAsyncDownload(&api, helper)
		if err != nil {
			spinner.Fail(
				fmt.Sprintf("%s: %s", msg, err.Error()),
			)
			if arguments.CommandArgs.Skip {
				continue
			} else {
				return err
			}
		}

		if !commandArgs.Force {
			// Check timestamps only if force is not true

			remoteStats, err := txapi.GetResourceStats(&api, resource, nil)
			if err != nil {
				spinner.Fail(
					fmt.Sprintf("%s: %s", msg, err.Error()),
				)
				if arguments.CommandArgs.Skip {
					continue
				} else {
					return err
				}
			}

			localLanguageCode, _ := txapi.CreateLanguageDirectory(
				cfg.Local.LanguageMappings, lang, cfgResource,
			)
			languageFilePath := strings.ReplaceAll(
				cfgResource.FileFilter, "<lang>", localLanguageCode,
			)
			key := download.Relationships["language"].DataSingular.Id
			remoteStat := remoteStats[key]

			minimum_perc := arguments.CommandArgs.MinimumPercentage
			if minimum_perc == -1 {
				if cfgResource.MinimumPercentage > -1 {
					minimum_perc = cfgResource.MinimumPercentage
				}
			}
			skip, err := shouldSkipDownload(
				languageFilePath,
				remoteStat,
				arguments.CommandArgs.UseGitTimestamps,
				arguments.CommandArgs.Mode,
				minimum_perc,
			)
			if err != nil {
				spinner.Fail(fmt.Sprintf("%s: %s", msg, err.Error()))
				if arguments.CommandArgs.Skip {
					continue
				} else {
					return err
				}
			}
			if skip {
				spinner.Warning(fmt.Sprintf(
					"Skipping download translation for resource '%s - %s'",
					resource.Id, lang,
				))
				continue
			}
		}

		duration, _ := time.ParseDuration("2s")
		err = txapi.PollTranslationDownload(
			cfg.Local.LanguageMappings,
			download,
			duration,
			cfgResource,
			arguments.CommandArgs.FileType,
		)
		if err != nil {
			spinner.Fail(
				fmt.Sprintf("%s: %s", msg, err.Error()),
			)
			if arguments.CommandArgs.Skip {
				continue
			} else {
				return err
			}
		} else {
			spinner.Success(fmt.Sprintf(
				"Translation file '%s' downloaded", translationFile,
			))
		}
	}

	return nil
}

func addAdditionalLocalLanguages(
	cfgResource *config.Resource,
	allLocalLanguages *map[string]string,
	extension string,
) {
	additionalLocalLanguages := searchFileFilter(
		".", fmt.Sprintf("%s.%s", cfgResource.FileFilter, extension),
	)

	for key, value := range additionalLocalLanguages {
		if _, ok := (*allLocalLanguages)[key]; !ok {
			// Key found with xlf file does not exist yet in
			// allLocalLanguages"). Adding it now
			(*allLocalLanguages)[key] = value
		}
	}
}

func createResourceStringsAsyncDownloads(
	api jsonapi.Connection,
	arguments *CreateAsyncDownloadArguments) error {
	project := arguments.Project
	cfgResource := arguments.CfgResource
	commandArgs := arguments.CommandArgs
	msg := "Downloading source file " + cfgResource.SourceFile

	resource, err := fetchResource(api, cfgResource, project)
	if err != nil {
		return err
	}

	helper := txapi.CreateResourceStringDownloadArguments{
		OrganizationSlug: cfgResource.OrganizationSlug,
		ProjectSlug:      cfgResource.ProjectSlug,
		ResourceSlug:     cfgResource.ResourceSlug,
		Resource:         resource,
		FileType:         commandArgs.FileType,
		ContentEncoding:  commandArgs.ContentEncoding,
	}

	spinner, err := pterm.DefaultSpinner.Start(msg)
	if err != nil {
		return err
	}

	sourceFile := setFileTypeExtensions(commandArgs, cfgResource.SourceFile)
	_, err = os.Stat(sourceFile)
	if err == nil && commandArgs.DisableOverwrite {
		spinner.Warning(fmt.Sprintf("Disable Overwrite is enabled. "+
			"Skip downloading source file '%s'", cfgResource.SourceFile))
		return nil
	}

	download, err := txapi.CreateResourceStringsAsyncDownload(&api, helper)
	if err != nil {
		spinner.Fail(msg + ": " + err.Error())
		return err
	}

	if !commandArgs.Force {
		// Check timestamps only if force is not true
		skip, err := shouldSkipResourceDownload(cfgResource.SourceFile,
			resource,
			arguments.CommandArgs.UseGitTimestamps)
		if err != nil {
			spinner.Fail(msg + ": " + err.Error())
		}
		if skip {
			spinner.Success()
			return nil
		}
	}
	duration, _ := time.ParseDuration("2s")
	err = txapi.PollResourceStringsDownload(
		download,
		duration,
		cfgResource,
		arguments.CommandArgs.FileType)
	if err != nil {
		if !arguments.CommandArgs.Skip {
			spinner.Fail(msg + ": " + err.Error())
			return err
		}
		spinner.Warning(
			fmt.Sprintf("Couldn't downloaded source file '%s'",
				cfgResource.SourceFile),
		)
	} else {
		spinner.Success(
			fmt.Sprintf("Source file '%s' downloaded",
				cfgResource.SourceFile),
		)
	}

	return nil
}

func fetchResource(
	api jsonapi.Connection,
	cfgResource *config.Resource,
	project *jsonapi.Resource) (*jsonapi.Resource, error) {
	msg := fmt.Sprintf("Searching for resource '%s'", cfgResource.ResourceSlug)
	spinner, err := pterm.DefaultSpinner.Start(msg)
	if err != nil {
		return nil, err
	}
	resource, err := txapi.GetResource(&api, project, cfgResource.ResourceSlug)
	if err != nil {
		spinner.Fail(msg + ": " + err.Error())
		return nil, err
	}
	if resource == nil {
		err = fmt.Errorf("%s: Not found", msg)
		spinner.Fail(err)
		return nil, err
	}
	spinner.Success(
		fmt.Sprintf("Resource %s fetched", cfgResource.ResourceSlug),
	)
	return resource, nil
}

func getActedOnStringsPercentage(
	actedOnStrings float32,
	totalStrings float32) float32 {

	actedOnStringsPerc := (actedOnStrings * 100) / totalStrings
	return actedOnStringsPerc
}

func shouldSkipDueToStringPercentage(
	minimum_perc int,
	actedOnStrings int,
	totalStrings int) bool {

	minimum_percFloat := float32(minimum_perc)
	actedOnStringsFloat := float32(actedOnStrings)
	totalStringsFloat := float32(totalStrings)

	actedOnStringsPerc := getActedOnStringsPercentage(
		actedOnStringsFloat, totalStringsFloat)

	if actedOnStringsPerc < minimum_percFloat {
		return true
	}
	return false
}
func shouldSkipDownload(
	path string, remoteStat *jsonapi.Resource, useGitTimestamps bool,
	mode string, minimum_perc int,
) (bool, error) {
	var localTime time.Time

	var remoteStatAttributes txapi.ResourceLanguageStatsAttributes
	err := remoteStat.MapAttributes(&remoteStatAttributes)
	if err != nil {
		return false, err
	}
	remoteTime, err := time.Parse(time.RFC3339,
		remoteStatAttributes.LastUpdate)
	if err != nil {
		return false, err
	}

	if minimum_perc > 0 {
		actedOnStrings := remoteStatAttributes.TranslatedStrings
		switch mode {
		case "reviewed", "onlyreviewed":
			actedOnStrings = remoteStatAttributes.ReviewedStrings
		case "proofread", "onlyproofread":
			actedOnStrings = remoteStatAttributes.ProofreadStrings
		}

		totalStrings := remoteStatAttributes.TotalStrings

		skipDueToStringPercentage := shouldSkipDueToStringPercentage(
			minimum_perc, actedOnStrings, totalStrings,
		)
		if skipDueToStringPercentage {
			return true, nil
		}
	}
	if useGitTimestamps {
		// TODO: check if parent folder is repo
		localTime = getLastCommitDate(path)
		if localTime == (time.Time{}) {
			return shouldSkipDownload(path, remoteStat,
				false, mode, minimum_perc)
		}
	} else {
		localStat, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
		localTime = localStat.ModTime().UTC()
	}

	// Don't pull if local file is newer than remote
	// resource-language
	return remoteTime.Before(localTime), nil
}

func shouldSkipResourceDownload(
	path string, resource *jsonapi.Resource, useGitTimestamps bool,
) (bool, error) {
	var localTime time.Time

	if useGitTimestamps {
		localTime = getLastCommitDate(path)
		if localTime == (time.Time{}) {
			return shouldSkipResourceDownload(path, resource, false)
		}
	} else {
		localStat, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			} else {
				return false, err
			}
		}
		localTime = localStat.ModTime().UTC()
	}

	var resourceAttributes txapi.ResourceAttributes

	err := resource.MapAttributes(&resourceAttributes)
	if err != nil {
		return false, err
	}
	remoteTime, err := time.Parse(time.RFC3339,
		resourceAttributes.DatetimeModified)
	if err != nil {
		return false, err
	}

	// Don't pull if local file is newer than remote
	// resource-language
	return remoteTime.Before(localTime), nil
}

func setFileTypeExtensions(
	commandArgs *PullCommandArguments, translationFile string,
) string {
	if commandArgs.FileType == "xliff" {
		translationFile = fmt.Sprintf("%s.xlf", translationFile)
	} else if commandArgs.FileType == "json" {
		translationFile = fmt.Sprintf("%s.json", translationFile)
	}
	return translationFile
}
