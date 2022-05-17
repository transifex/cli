package txlib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/pterm/pterm"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
)

type PushCommandArguments struct {
	Source           bool
	Translation      bool
	Force            bool
	Skip             bool
	Xliff            bool
	Languages        []string
	ResourceIds      []string
	UseGitTimestamps bool
	Branch           string
	All              bool
	Workers          int
}

func PushCommand(
	cfg *config.Config, api jsonapi.Connection, args PushCommandArguments,
) error {
	if args.Branch == "-1" {
		args.Branch = ""
	} else if args.Branch == "" {
		args.Branch = getGitBranch()
		if args.Branch == "" {
			pterm.Warning.Println("Couldn't find branch information")
		}
	}
	if args.Branch != "" {
		pterm.Info.Printf("Using branch '%s'\n", args.Branch)
	}
	var cfgResources []*config.Resource
	if args.ResourceIds != nil && len(args.ResourceIds) != 0 {
		cfgResources = make([]*config.Resource, 0, len(args.ResourceIds))
		for _, resourceId := range args.ResourceIds {
			cfgResource := cfg.FindResource(resourceId)
			if cfgResource == nil {
				fmt.Println(pterm.Error.Sprintf(
					"could not find resource '%s' in local configuration or your resource slug is invalid",
					resourceId,
				))
				return fmt.Errorf(
					"could not find resource '%s' in local configuration or your resource slug is invalid",
					resourceId,
				)
			}

			_, err := os.Stat(cfgResource.SourceFile)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println(pterm.Error.Sprintf(
						"could not find file '%s'. Aborting.",
						cfgResource.SourceFile,
					))
					return fmt.Errorf(
						"could not find file '%s'. Aborting",
						cfgResource.SourceFile,
					)
				} else {
					fmt.Println(pterm.Error.Sprintf(
						"something went wrong while examining the source " +
							"file path",
					))
					return fmt.Errorf(
						"something went wrong while examining the source " +
							"file path",
					)
				}
			}
			cfgResources = append(cfgResources, cfgResource)
		}
	} else {
		for i := range cfg.Local.Resources {
			cfgResource := &cfg.Local.Resources[i]
			_, err := os.Stat(cfgResource.SourceFile)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println(pterm.Error.Sprintf(
						"could not find file '%s'. Aborting",
						cfgResource.SourceFile,
					))
					return fmt.Errorf(
						"could not find file '%s'. Aborting",
						cfgResource.SourceFile,
					)
				} else {
					fmt.Println(pterm.Error.Sprintf(
						"something went wrong while examining the source " +
							"file path",
					))
					return fmt.Errorf(
						"something went wrong while examining the source " +
							"file path",
					)
				}
			}
			cfgResources = append(cfgResources, cfgResource)
		}
	}

	for i := range cfgResources {
		cfgResource := cfgResources[i]
		if args.Branch != "" {
			cfgResource.ResourceSlug = fmt.Sprintf("%s--%s",
				slug.Make(args.Branch),
				cfgResource.ResourceSlug)
		}
	}

	for _, cfgResource := range cfgResources {
		err := pushResource(&api, cfg, *cfgResource, args)
		if err != nil {
			if !args.Skip {
				return err
			}
		}
	}
	return nil
}

func pushResource(
	api *jsonapi.Connection, cfg *config.Config, cfgResource config.Resource,
	args PushCommandArguments,
) error {
	pterm.DefaultSection.Printf("Resource %s\n", cfgResource.Name())
	duration, _ := time.ParseDuration("1s")

	msg := fmt.Sprintf("Searching for resource '%s'", cfgResource.ResourceSlug)
	spinner, err := pterm.DefaultSpinner.Start(msg)
	if err != nil {
		return err
	}

	resource, err := txapi.GetResourceById(api, cfgResource.GetAPv3Id())
	if err != nil {
		spinner.Fail(msg + ": " + err.Error())
		return err
	}
	resourceIsNew := false
	if resource != nil {
		spinner.Success(
			fmt.Sprintf("Resource with slug '%s' found", cfgResource.ResourceSlug),
		)
	} else {
		spinner.Warning(
			fmt.Sprintf(
				"Resource with slug '%s' not found. We will try to create it for you.",
				cfgResource.ResourceSlug,
			),
		)
		if cfgResource.Type == "" {
			return fmt.Errorf(
				"resource '%s - %s - %s' does not exist; cannot create "+
					"because the configuration is missing the 'type' field",
				cfgResource.OrganizationSlug,
				cfgResource.ProjectSlug,
				cfgResource.ResourceSlug)
		}
		var resourceName string
		if args.Branch == "" {
			resourceName = cfgResource.ResourceName()
		} else {
			resourceName = fmt.Sprintf("(branch %s) %s",
				args.Branch,
				cfgResource.ResourceName())
		}
		resource, err = txapi.CreateResource(
			api,
			fmt.Sprintf(
				"o:%s:p:%s",
				cfgResource.OrganizationSlug,
				cfgResource.ProjectSlug,
			),
			resourceName,
			cfgResource.ResourceSlug,
			cfgResource.Type)
		if err != nil {
			spinner.Fail(msg + ": " + err.Error())
			return err
		}
		spinner.Success(
			fmt.Sprintf("Created resource with name '%s' and slug '%s'",
				resourceName,
				cfgResource.ResourceSlug,
			))
		resourceIsNew = true
	}

	msg = "Fetching stats"
	spinner, err = pterm.DefaultSpinner.Start(msg)
	if err != nil {
		return err
	}
	remoteStats, err := getRemoteStats(api, resource, args)
	if err != nil {
		spinner.Fail(msg + ": " + err.Error())
		return err
	}
	spinner.Success("Stats fetched")

	var sourceUpload *jsonapi.Resource
	// should push source either when -s flag is set or when neither -s, -t are
	if args.Source || !args.Translation {
		fmt.Print("\nPushing source:\n\n")
		msg = fmt.Sprintf("Pushing source file '%s'", cfgResource.SourceFile)
		spinner, err = pterm.DefaultSpinner.Start(msg)
		if err != nil {
			return err
		}
		sourceUpload, err = pushSource(
			api, resource, cfgResource.SourceFile, remoteStats, args,
			resourceIsNew,
		)
		if err != nil {
			spinner.Fail(msg + ": " + err.Error())
			if !args.Skip {
				return err
			}
		}
		if sourceUpload != nil {
			spinner.Success(
				fmt.Sprintf("Source file '%s' pushed", cfgResource.SourceFile),
			)
		} else {
			spinner.Warning(
				fmt.Sprintf(
					"Source file '%s' skipped because remote file is newer "+
						"than local",
					cfgResource.SourceFile,
				),
			)
		}
	}

	var translationUploads []*jsonapi.Resource
	if args.Translation { // -t flag is set
		translationUploads, err = pushTranslations(api, resource, cfg,
			cfgResource, remoteStats, args, resourceIsNew)
		if err != nil {
			if !args.Skip {
				return err
			}
		}
	}

	if sourceUpload != nil || len(translationUploads) > 0 {
		fmt.Print("\nPolling for upload completion:\n\n")
	}
	if sourceUpload != nil {
		msg = "Polling for source upload"
		spinner, err := pterm.DefaultSpinner.Start(msg)
		if err != nil {
			return err
		}
		err = txapi.PollSourceUpload(sourceUpload, duration)
		if err != nil {
			spinner.Fail(msg + ": " + err.Error())
			if !args.Skip {
				return err
			}
		}
		spinner.Success("Source language upload verified")
	}

	for _, upload := range translationUploads {
		languageRelationship, err := upload.Fetch("language")
		if err != nil {
			return err
		}
		language := languageRelationship.DataSingular
		var languageAttributes txapi.LanguageAttributes
		err = language.MapAttributes(&languageAttributes)
		if err != nil {
			return err
		}

		msg = fmt.Sprintf("Polling for language '%s' upload",
			languageAttributes.Code)
		spinner, err := pterm.DefaultSpinner.Start(msg)
		if err != nil {
			return err
		}
		err = txapi.PollTranslationUpload(upload, duration)
		if err != nil {
			spinner.Fail(msg + ": " + err.Error())
			if !args.Skip {
				return err
			}
		}
		spinner.Success(
			fmt.Sprintf("Language '%s' upload verified",
				languageAttributes.Code),
		)
	}

	return nil
}

func pushSource(
	api *jsonapi.Connection,
	resource *jsonapi.Resource,
	sourceFile string,
	remoteStats map[string]*jsonapi.Resource,
	args PushCommandArguments,
	resourceIsNew bool,
) (*jsonapi.Resource, error) {
	if sourceFile == "" {
		return nil, errors.New("cannot push source file because the " +
			"configuration is missing the 'source_file' field")
	}
	file, err := os.Open(sourceFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Only check timestamps if -f isn't set and if resource isn't new
	if !args.Force && !resourceIsNew {
		// Project should already be pre-fetched
		projectRelationship, err := resource.Fetch("project")
		if err != nil {
			return nil, err
		}
		project := projectRelationship.DataSingular
		sourceLanguageRelationship := project.Relationships["source_language"]
		remoteStat := remoteStats[sourceLanguageRelationship.DataSingular.Id]
		skip, err := shouldSkipPush(sourceFile, remoteStat,
			args.UseGitTimestamps)
		if err != nil {
			return nil, err
		}
		if skip {
			return nil, nil
		}
	}
	return txapi.UploadSource(api, resource, file)
}

func pushTranslations(
	api *jsonapi.Connection,
	resource *jsonapi.Resource,
	cfg *config.Config,
	cfgResource config.Resource,
	remoteStats map[string]*jsonapi.Resource,
	args PushCommandArguments,
	resourceIsNew bool,
) ([]*jsonapi.Resource, error) {
	fmt.Print("\nPushing translations:\n\n")

	reverseLanguageMappings := makeReverseLanguageMappings(*cfg, cfgResource)
	overrides := cfgResource.Overrides
	projectRelationship, err := resource.Fetch("project")
	if err != nil {
		return nil, err
	}
	project := projectRelationship.DataSingular

	msg := "Fetching project's remote languages"
	spinner, err := pterm.DefaultSpinner.Start(msg)
	if err != nil {
		return nil, err
	}
	remoteLanguages, err := txapi.GetProjectLanguages(project)
	if err != nil {
		spinner.Fail(msg + ": " + err.Error())
		return nil, err
	}
	var languageCodes []string
	for languageCode := range remoteLanguages {
		languageCodes = append(languageCodes, languageCode)
	}
	spinner.Success(
		fmt.Sprintf("Project's remote languages fetched: %s",
			strings.Join(languageCodes, ", ")),
	)

	curDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	fileFilter := cfgResource.FileFilter
	if err := isFileFilterValid(fileFilter); err != nil {
		return nil, err
	}
	if args.Xliff {
		fileFilter = fmt.Sprintf("%s.xlf", fileFilter)
	}

	languageCodesToPush, pathsToPush, newLanguageCodes, err := getFilesToPush(
		curDir, fileFilter, reverseLanguageMappings, remoteLanguages,
		remoteStats, overrides, args, resourceIsNew,
	)
	if err != nil {
		return nil, err
	}

	if len(newLanguageCodes) > 0 {
		remoteLanguages, err = createNewLanguages(
			api, project, remoteLanguages, newLanguageCodes,
		)
		if err != nil {
			return nil, err
		}
	}
	if len(languageCodesToPush) < 1 {
		spinner.Warning("No language files found to push. Aborting")
	}
	var uploads []*jsonapi.Resource
	for i := range languageCodesToPush {
		languageCode := languageCodesToPush[i]
		path := pathsToPush[i]
		_, exists := remoteLanguages[languageCode]
		if !exists {
			continue
		}

		msg := fmt.Sprintf("Uploading '%s'", path)
		spinner, err := pterm.DefaultSpinner.Start(msg)
		if err != nil {
			return nil, err
		}
		upload, err := pushTranslation(
			api, languageCode, path, resource, remoteLanguages, args,
		)
		if err != nil {
			spinner.Fail(msg + ": " + err.Error())
			if !args.Skip {
				return nil, err
			}
		}
		spinner.Success(fmt.Sprintf("'%s' uploaded", path))
		uploads = append(uploads, upload)
	}
	return uploads, nil
}

func getFilesToPush(
	curDir, fileFilter string,
	reverseLanguageMappings map[string]string,
	remoteLanguages, remoteStats map[string]*jsonapi.Resource,
	overrides map[string]string,
	args PushCommandArguments,
	resourceIsNew bool,
) ([]string, []string, []string, error) {
	var languageCodesToPush []string
	var pathsToPush []string
	var newLanguageCodes []string

	allLocalLanguages := searchFileFilter(curDir, fileFilter)

	if len(overrides) > 0 {
		for langOverride := range overrides {
			// Add the Resource file filter overrides per lang
			allLocalLanguages[langOverride] = filepath.
				Join(curDir, overrides[langOverride])
			// In case of xliff add the extension
			if args.Xliff {
				allLocalLanguages[langOverride] = fmt.Sprintf("%s.xlf",
					allLocalLanguages[langOverride])
			}
		}
	}

	for localLanguageCode, path := range allLocalLanguages {
		remoteLanguageCode, exists := reverseLanguageMappings[localLanguageCode]
		if !exists {
			remoteLanguageCode = localLanguageCode
		}

		// if -l is set and the language is not in one of the languages, we
		// must skip
		if len(args.Languages) > 0 &&
			(!contains(args.Languages, localLanguageCode) &&
				!contains(args.Languages, remoteLanguageCode)) {
			continue
		}

		language, exists := remoteLanguages[remoteLanguageCode]
		if !exists {
			// if --all is set or -l is set and the code is in one of the
			// languages, we need to create the remote language
			if args.All || (len(args.Languages) > 0 &&
				(contains(args.Languages, localLanguageCode) ||
					contains(args.Languages, remoteLanguageCode))) {
				languageCodesToPush = append(languageCodesToPush,
					remoteLanguageCode)
				pathsToPush = append(pathsToPush, path)
				newLanguageCodes = append(newLanguageCodes, remoteLanguageCode)
			}
			continue
		}

		// Only check timestamps if -f isn't set and if resource isn't new
		if !args.Force && !resourceIsNew {
			remoteStat, exists := remoteStats[language.Id]
			if !exists {
				return nil, nil, nil, fmt.Errorf(
					"couldn't find matching stats for the language %s",
					language.Id,
				)
			}
			skip, err := shouldSkipPush(path, remoteStat, args.UseGitTimestamps)
			if err != nil {
				return nil, nil, nil, err
			}
			if skip {
				continue
			}
		}
		languageCodesToPush = append(languageCodesToPush, remoteLanguageCode)
		pathsToPush = append(pathsToPush, path)
	}
	return languageCodesToPush, pathsToPush, newLanguageCodes, nil
}

func pushTranslation(
	api *jsonapi.Connection,
	languageCode, path string,
	resource *jsonapi.Resource,
	remoteLanguages map[string]*jsonapi.Resource,
	args PushCommandArguments,
) (*jsonapi.Resource, error) {
	language, exists := remoteLanguages[languageCode]
	if !exists {
		return nil, fmt.Errorf("language '%s' not found (unreachable code)",
			languageCode)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	upload, err := txapi.UploadTranslation(
		api, resource, language, file, args.Xliff,
	)
	if err != nil {
		return nil, err
	}
	return upload, nil
}

func shouldSkipPush(
	path string, remoteStat *jsonapi.Resource, useGitTimestamps bool,
) (bool, error) {
	var localTime time.Time

	if useGitTimestamps {
		localTime = getLastCommitDate(path)
		if localTime == (time.Time{}) {
			return shouldSkipPush(path, remoteStat, false)
		}
	} else {
		localStat, err := os.Stat(path)
		if err != nil {
			return false, err
		}
		localTime = localStat.ModTime().UTC()
	}

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

	// Don't push if local file is older than remote
	// resource-language
	return localTime.Before(remoteTime), nil
}

// Trivial contains function
func contains(pool []string, target string) bool {
	for _, candidate := range pool {
		if target == candidate {
			return true
		}
	}
	return false
}

func isFileFilterValid(fileFilter string) error {
	if fileFilter == "" {
		return errors.New("cannot push translations because the " +
			"configuration file is missing the 'file_filter' field")
	} else if strings.Count(fileFilter, "<lang>") != 1 {
		return errors.New(
			"cannot push translations because the file_filter' field " +
				"doesn't have exactly one occurrence of '<lang>'",
		)
	} else {
		return nil
	}
}

func makeReverseLanguageMappings(
	cfg config.Config, cfgResource config.Resource,
) map[string]string {
	// In the configuration, the language mappings are "remote code -> local
	// code" (eg 'pt_BT: pt-br'). Looking into the filesystem, we get the local
	// language codes; so if we need to find the remote codes, we need to
	// reverse the maps

	result := make(map[string]string)
	for key, value := range cfg.Local.LanguageMappings {
		result[value] = key
	}
	for key, value := range cfgResource.LanguageMappings {
		// Resource language mappings overwrite "global" language mappings
		result[value] = key
	}
	return result
}

func getRemoteStats(
	api *jsonapi.Connection,
	resource *jsonapi.Resource,
	args PushCommandArguments,
) (map[string]*jsonapi.Resource, error) {
	var result map[string]*jsonapi.Resource
	// We don't need remote stats if -f isn't set
	if !args.Force {
		var err error
		if args.Translation {
			// We need all stats
			result, err = txapi.GetResourceStats(api, resource, nil)
			if err != nil {
				return nil, err
			}
		} else {
			projectRelationship, err := resource.Fetch("project")
			if err != nil {
				return nil, err
			}
			project := projectRelationship.DataSingular
			// We only need stats for the source language
			result, err = txapi.GetResourceStats(
				api, resource,
				project.Relationships["source_language"].DataSingular,
			)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func createNewLanguages(
	api *jsonapi.Connection,
	project *jsonapi.Resource,
	remoteLanguages map[string]*jsonapi.Resource,
	newLanguageCodes []string,
) (map[string]*jsonapi.Resource, error) {
	// msg := fmt.Sprintf("Adding '%s' to the project's remote languages",
	// 	strings.Join(newLanguageCodes, ", "))
	// spinner, err := pterm.DefaultSpinner.Start(msg)
	// if err != nil {
	// 	return nil, err
	// }

	allLanguages, err := txapi.GetLanguages(api)
	if err != nil {
		// spinner.Fail(msg + ": " + err.Error())
		return nil, errors.New("failed to fetch languages")
	}
	// var skippedLanguageCodes []string
	var languagesToCreate []*jsonapi.Resource
	for _, languageCode := range newLanguageCodes {
		language, exists := allLanguages[languageCode]
		if !exists {
			// skippedLanguageCodes = append(skippedLanguageCodes, languageCode)
			continue
		}
		sourceLanguageRelationship, err := project.Fetch("source_language")
		if err != nil {
			// spinner.Fail(msg + ": " + err.Error())
			return nil, err
		}
		sourceLanguage := sourceLanguageRelationship.DataSingular
		var sourceLanguageAttributes txapi.LanguageAttributes
		err = sourceLanguage.MapAttributes(&sourceLanguageAttributes)
		if err != nil {
			// spinner.Fail(msg + ": " + err.Error())
			return nil, err
		}
		if languageCode == sourceLanguageAttributes.Code {
			// skippedLanguageCodes = append(skippedLanguageCodes, languageCode)
			continue
		}

		languagesToCreate = append(languagesToCreate, language)
	}
	// newLanguageCodes = make([]string, 0, len(languagesToCreate))
	if len(languagesToCreate) > 0 {
		for _, language := range languagesToCreate {
			var languageAttributes txapi.LanguageAttributes
			err := language.MapAttributes(&languageAttributes)
			if err != nil {
				// spinner.Fail(msg + ": " + err.Error())
				return nil, err
			}
			// newLanguageCodes = append(newLanguageCodes,
			// 	languageAttributes.Code)
		}
		err := project.Add("languages", languagesToCreate)
		if err != nil {
			// spinner.Fail(msg + ": " + err.Error())
			return nil, err
		}
		remoteLanguages, err = txapi.GetProjectLanguages(project)
		if err != nil {
			// spinner.Fail(msg + ": " + err.Error())
			return nil, err
		}
	}
	// msg = fmt.Sprintf("Added languages: '%s'",
	// 	strings.Join(newLanguageCodes, ", "))
	// if len(skippedLanguageCodes) > 0 {
	// 	msg = msg + fmt.Sprintf(", skipped: '%s'",
	// 		strings.Join(skippedLanguageCodes, ", "))
	// }
	// spinner.Success(msg)
	return remoteLanguages, nil
}
