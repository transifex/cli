package txlib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/pterm/pterm"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/transifex/cli/pkg/worker_pool"
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
	cfg *config.Config,
	api jsonapi.Connection,
	args PushCommandArguments,
) error {
	args.Branch = figureOutBranch(args.Branch)

	cfgResources, err := figureOutResources(args.ResourceIds, cfg)
	if err != nil {
		return err
	}

	applyBranchToResources(cfgResources, args.Branch)

	sort.Slice(cfgResources, func(i, j int) bool {
		return cfgResources[i].GetAPv3Id() < cfgResources[j].GetAPv3Id()
	})

	// Step 1: Resources

	fmt.Print("# Getting info about resources\n\n")

	pool := worker_pool.New(args.Workers, len(cfgResources))
	sourceTaskChannel := make(chan SourceFileTask)
	translationTaskChannel := make(chan TranslationFileTask)
	targetLanguagesChannel := make(chan TargetLanguageMessage)
	for _, cfgResource := range cfgResources {
		pool.Add(
			ResourceTask{
				cfg,
				cfgResource,
				sourceTaskChannel,
				translationTaskChannel,
				&api,
				args,
				targetLanguagesChannel,
			},
		)
	}
	pool.Start()

	var sourceFileTasks []SourceFileTask
	var translationFileTasks []TranslationFileTask
	projects := make(map[string]*jsonapi.Resource)
	targetLanguages := make(map[string][]string)

	waitChannel := pool.Wait()
	exitfor := false
	for !exitfor {
		select {
		case sourceFileTask := <-sourceTaskChannel:
			sourceFileTasks = append(sourceFileTasks, sourceFileTask)

		case translationFileTask := <-translationTaskChannel:
			translationFileTasks = append(translationFileTasks, translationFileTask)

		case targetLanguageMessage := <-targetLanguagesChannel:
			project := targetLanguageMessage.project
			languageId := targetLanguageMessage.languageId

			_, exists := projects[project.Id]
			if !exists {
				projects[project.Id] = project
			}

			languages, exists := targetLanguages[project.Id]
			if !exists {
				targetLanguages[project.Id] = []string{}
				languages = targetLanguages[project.Id]
			}
			if !stringSliceContains(languages, languageId) {
				targetLanguages[project.Id] = append(
					targetLanguages[project.Id],
					languageId,
				)
			}

		case <-waitChannel:
			exitfor = true
		}
	}

	if pool.IsAborted {
		fmt.Println("Aborted")
		return errors.New("Aborted")
	}

	// Step 2: Create missing remote target languages
	if len(targetLanguages) > 0 {
		fmt.Print("\n# Create missing remote target languages\n\n")

		pool = worker_pool.New(args.Workers, len(targetLanguages))
		for projectId, languages := range targetLanguages {
			sort.Slice(languages, func(i, j int) bool {
				return languages[i] < languages[j]
			})
			pool.Add(LanguagePushTask{projects[projectId], languages})
		}
		pool.Start()
		<-pool.Wait()
		if pool.IsAborted {
			fmt.Println("Aborted")
			return errors.New("Aborted")
		}
	}

	// Step 3: SourceFiles

	if len(sourceFileTasks) > 0 {
		fmt.Print("\n# Pushing source files\n\n")

		sort.Slice(sourceFileTasks, func(i, j int) bool {
			return sourceFileTasks[i].resource.Id < sourceFileTasks[j].resource.Id
		})
		pool = worker_pool.New(args.Workers, len(sourceFileTasks))
		for _, sourceFileTask := range sourceFileTasks {
			pool.Add(sourceFileTask)
		}
		pool.Start()
		<-pool.Wait()

		if pool.IsAborted {
			fmt.Println("Aborted")
			return errors.New("Aborted")
		}
	}

	// Step 4: Translations

	if len(translationFileTasks) > 0 {
		sort.Slice(translationFileTasks, func(i, j int) bool {
			left := translationFileTasks[i]
			right := translationFileTasks[j]
			if left.resource.Id != right.resource.Id {
				return left.resource.Id < right.resource.Id
			} else {
				return left.languageCode < right.languageCode
			}
		})
		fmt.Print("\n# Pushing translations\n\n")

		pool = worker_pool.New(args.Workers, len(translationFileTasks))
		for _, translationFileTask := range translationFileTasks {
			pool.Add(translationFileTask)
		}
		pool.Start()
		<-pool.Wait()

		if pool.IsAborted {
			fmt.Println("Aborted")
			return errors.New("Aborted")
		}
	}

	return nil
}

type TargetLanguageMessage struct {
	project    *jsonapi.Resource
	languageId string
}

type ResourceTask struct {
	cfg                    *config.Config
	cfgResource            *config.Resource
	sourceTaskChannel      chan SourceFileTask
	translationTaskChannel chan TranslationFileTask
	api                    *jsonapi.Connection
	args                   PushCommandArguments
	targetLanguagesChannel chan TargetLanguageMessage
}

func (task ResourceTask) Run(send func(string), abort func()) {
	cfg := task.cfg
	cfgResource := task.cfgResource
	sourceTaskChannel := task.sourceTaskChannel
	translationTaskChannel := task.translationTaskChannel
	api := task.api
	args := task.args
	targetLanguagesChannel := task.targetLanguagesChannel

	sendMessage := func(body string) {
		send(fmt.Sprintf(
			"%s.%s - %s",
			cfgResource.ProjectSlug,
			cfgResource.ResourceSlug,
			body,
		))
	}
	sendMessage("Getting info")
	resource, err := txapi.GetResourceById(api, cfgResource.GetAPv3Id())
	if err != nil {
		sendMessage(fmt.Sprintf("Error while fetching resource: %s", err))
		return
	}

	resourceIsNew := resource == nil
	if resourceIsNew {
		sendMessage("Resource does not exist; creating")
		if cfgResource.Type == "" {
			sendMessage("Error: Cannot create resource, i18n type is unknown")
			if !args.Skip {
				abort()
			}
			return
		}
		var resourceName string
		if args.Branch == "" {
			resourceName = cfgResource.ResourceName()
		} else {
			resourceName = fmt.Sprintf(
				"%s (branch %s)",
				cfgResource.ResourceName(),
				args.Branch,
			)
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
			sendMessage(fmt.Sprintf("Error while creating resource, %s", err))
			if !args.Skip {
				abort()
			}
			return
		}
	}

	sendMessage("Getting stats")
	remoteStats, err := getRemoteStats(api, resource, args)
	if err != nil {
		sendMessage(fmt.Sprintf("Error while fetching stats, %s", err))
		if !args.Skip {
			abort()
		}
		return
	}
	if args.Source || !args.Translation {
		sourceTaskChannel <- SourceFileTask{
			api,
			resource,
			cfgResource.SourceFile,
			remoteStats,
			args,
			resourceIsNew,
		}
	}
	if args.Translation { // -t flag is set
		reverseLanguageMappings := makeReverseLanguageMappings(*cfg, *cfgResource)
		overrides := cfgResource.Overrides
		projectRelationship, err := resource.Fetch("project")
		if err != nil {
			sendMessage(fmt.Sprintf("Error while fetching project, %s", err))
			if !args.Skip {
				abort()
			}
			return
		}
		project := projectRelationship.DataSingular

		sendMessage("Fetching remote languages")
		// TODO see if we can figure our remote languages from stats
		remoteLanguages, err := txapi.GetProjectLanguages(project)
		if err != nil {
			sendMessage(fmt.Sprintf("Error while fetching remote languages, %s", err))
			if !args.Skip {
				abort()
			}
			return
		}
		curDir, err := os.Getwd()
		if err != nil {
			sendMessage(err.Error())
			if !args.Skip {
				abort()
			}
			return
		}
		fileFilter := cfgResource.FileFilter
		err = isFileFilterValid(fileFilter)
		if err != nil {
			sendMessage(err.Error())
			if !args.Skip {
				abort()
			}
			return
		}
		if args.Xliff {
			fileFilter = fmt.Sprintf("%s.xlf", fileFilter)
		}

		languageCodesToPush, pathsToPush, newLanguageCodes, err := getFilesToPush(
			curDir, fileFilter, reverseLanguageMappings, remoteLanguages,
			remoteStats, overrides, args, resourceIsNew,
		)
		if err != nil {
			sendMessage(err.Error())
			if !args.Skip {
				abort()
			}
			return
		}

		allLanguages, err := txapi.GetLanguages(api)
		if err != nil {
			sendMessage(err.Error())
			abort()
			return
		}
		sourceLanguageId := project.Relationships["source_language"].DataSingular.Id
		for _, languageCode := range newLanguageCodes {
			language, exists := allLanguages[languageCode]
			if !exists || fmt.Sprintf("l:%s", languageCode) == sourceLanguageId {
				continue
			}
			remoteLanguages[languageCode] = language
			targetLanguagesChannel <- TargetLanguageMessage{project, languageCode}
		}
		for i := range languageCodesToPush {
			languageCode := languageCodesToPush[i]
			path := pathsToPush[i]

			_, exists := allLanguages[languageCode]
			if !exists || fmt.Sprintf("l:%s", languageCode) == sourceLanguageId {
				continue
			}

			translationTaskChannel <- TranslationFileTask{
				api,
				languageCode,
				path,
				resource,
				remoteLanguages,
				args,
				remoteStats,
				resourceIsNew,
			}
		}
	}
	sendMessage("Done")
}

type LanguagePushTask struct {
	project   *jsonapi.Resource
	languages []string
}

func (task LanguagePushTask) Run(send func(string), abort func()) {
	project := task.project
	languages := task.languages

	parts := strings.Split(project.Id, ":")

	sendMessage := func(body string) {
		send(fmt.Sprintf(
			"%s (%s) - %s",
			parts[3],
			strings.Join(languages, ", "),
			body,
		))
	}
	sendMessage("Pushing")

	var payload []*jsonapi.Resource
	for _, language := range languages {
		payload = append(payload, &jsonapi.Resource{
			Type: "languages",
			Id:   fmt.Sprintf("l:%s", language),
		})
	}
	err := project.Add("languages", payload)
	if err != nil {
		sendMessage(err.Error())
		abort()
		return
	}

	sendMessage("Done")
}

type SourceFileTask struct {
	api           *jsonapi.Connection
	resource      *jsonapi.Resource
	sourceFile    string
	remoteStats   map[string]*jsonapi.Resource
	args          PushCommandArguments
	resourceIsNew bool
}

func (task SourceFileTask) Run(send func(string), abort func()) {
	api := task.api
	resource := task.resource
	sourceFile := task.sourceFile
	remoteStats := task.remoteStats
	args := task.args
	resourceIsNew := task.resourceIsNew

	parts := strings.Split(resource.Id, ":")

	sendMessage := func(body string) {
		send(fmt.Sprintf("%s.%s - %s", parts[3], parts[5], body))
	}

	if sourceFile == "" {
		return
	}
	file, err := os.Open(sourceFile)
	if err != nil {
		sendMessage(err.Error())
		if !args.Skip {
			abort()
		}
		return
	}
	defer file.Close()

	// Only check timestamps if -f isn't set and if resource isn't new
	if !args.Force && !resourceIsNew {
		// Project should already be pre-fetched
		projectRelationship, err := resource.Fetch("project")
		if err != nil {
			sendMessage(err.Error())
			if !args.Skip {
				abort()
			}
			return
		}
		project := projectRelationship.DataSingular
		sourceLanguageRelationship := project.Relationships["source_language"]
		remoteStat := remoteStats[sourceLanguageRelationship.DataSingular.Id]
		skip, err := shouldSkipPush(
			sourceFile, remoteStat, args.UseGitTimestamps,
		)
		if skip {
			sendMessage("Skipping")
			return
		}
		if err != nil {
			sendMessage(err.Error())
			if !args.Skip {
				abort()
			}
			return
		}
	}

	sendMessage("Uploading file")

	var sourceUpload *jsonapi.Resource
	err = handleThrottling(
		func() error {
			var err error
			sourceUpload, err = txapi.UploadSource(api, resource, file)
			return err
		},
		sendMessage,
	)
	if err != nil {
		sendMessage(err.Error())
		if !args.Skip {
			abort()
		}
		return
	}

	sendMessage("Polling")

	err = handleThrottling(
		func() error {
			return txapi.PollSourceUpload(sourceUpload, time.Second)
		},
		sendMessage,
	)
	if err != nil {
		sendMessage(err.Error())
		if !args.Skip {
			abort()
		}
		return
	}

	sendMessage("Done")
}

type TranslationFileTask struct {
	api             *jsonapi.Connection
	languageCode    string
	path            string
	resource        *jsonapi.Resource
	remoteLanguages map[string]*jsonapi.Resource
	args            PushCommandArguments
	remoteStats     map[string]*jsonapi.Resource
	resourceIsNew   bool
}

func (task TranslationFileTask) Run(send func(string), abort func()) {
	api := task.api
	languageCode := task.languageCode
	path := task.path
	resource := task.resource
	remoteLanguages := task.remoteLanguages
	args := task.args
	remoteStats := task.remoteStats
	resourceIsNew := task.resourceIsNew

	parts := strings.Split(resource.Id, ":")

	sendMessage := func(body string) {
		send(fmt.Sprintf("%s.%s [%s] - %s", parts[3], parts[5], languageCode, body))
	}

	sendMessage("Uploading file")

	// Only check timestamps if -f isn't set and if resource isn't new
	if !args.Force && !resourceIsNew {
		languageId := fmt.Sprintf("l:%s", languageCode)
		remoteStat, exists := remoteStats[languageId]
		if exists {
			skip, err := shouldSkipPush(path, remoteStat, args.UseGitTimestamps)
			if err != nil {
				sendMessage(err.Error())
				if !args.Skip {
					abort()
				}
				return
			}
			if skip {
				sendMessage("Skipping because remote file is newer than local")
				return
			}
		}
	}

	var upload *jsonapi.Resource
	err := handleThrottling(
		func() error {
			var err error
			upload, err = pushTranslation(
				api, languageCode, path, resource, remoteLanguages, args,
			)
			return err
		},
		sendMessage,
	)
	if err != nil {
		sendMessage(err.Error())
		if !args.Skip {
			abort()
		}
		return
	}

	sendMessage("Polling")
	err = handleThrottling(
		func() error {
			return txapi.PollTranslationUpload(upload, time.Second)
		},
		sendMessage,
	)
	if err != nil {
		sendMessage(err.Error())
		if !args.Skip {
			abort()
		}
		return
	}
	sendMessage("Done")
}

func figureOutBranch(branch string) string {
	if branch == "-1" {
		return ""
	} else if branch == "" {
		return getGitBranch()
	} else {
		return branch
	}
}

func figureOutResources(
	resourceIds []string,
	cfg *config.Config,
) ([]*config.Resource, error) {
	var result []*config.Resource

	if len(resourceIds) != 0 {
		result = make([]*config.Resource, 0, len(resourceIds))
		for _, resourceId := range resourceIds {
			cfgResource := cfg.FindResource(resourceId)
			if cfgResource == nil {
				fmt.Println(pterm.Error.Sprintf(
					"could not find resource '%s' in local configuration or your resource slug is invalid",
					resourceId,
				))
				return nil, fmt.Errorf(
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
					return nil, fmt.Errorf(
						"could not find file '%s'. Aborting",
						cfgResource.SourceFile,
					)
				} else {
					fmt.Println(pterm.Error.Sprintf(
						"something went wrong while examining the source " +
							"file path",
					))
					return nil, fmt.Errorf(
						"something went wrong while examining the source " +
							"file path",
					)
				}
			}
			result = append(result, cfgResource)
		}
	} else {
		for i := range cfg.Local.Resources {
			cfgResource := &cfg.Local.Resources[i]
			_, err := os.Stat(cfgResource.SourceFile)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println(pterm.Error.Sprintf(
						"could not find file '%s'. Aborting.",
						cfgResource.SourceFile,
					))
					return nil, fmt.Errorf(
						"could not find file '%s'. Aborting",
						cfgResource.SourceFile,
					)
				} else {
					fmt.Println(pterm.Error.Sprintf(
						"something went wrong while examining the source " +
							"file path",
					))
					return nil, fmt.Errorf(
						"something went wrong while examining the source " +
							"file path",
					)
				}
			}
			result = append(result, cfgResource)
		}
	}
	return result, nil
}

func applyBranchToResources(cfgResources []*config.Resource, branch string) {
	for i := range cfgResources {
		cfgResource := cfgResources[i]
		if branch != "" {
			cfgResource.ResourceSlug = fmt.Sprintf(
				"%s--%s",
				slug.Make(branch),
				cfgResource.ResourceSlug,
			)
		}
	}
}

// Trivial contains function
func stringSliceContains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}

/*
Run 'do'. If the error returned by 'do' is a jsonapi.ThrottleError, sleep the number of
seconds indicated by the error and try again. Meanwhile, inform the user of
what's going on using 'send'.
*/
func handleThrottling(do func() error, send func(string)) error {
	for {
		err := do()
		if err == nil {
			return nil
		} else {
			var e *jsonapi.ThrottleError
			if errors.As(err, &e) {
				retryAfter := e.RetryAfter
				for retryAfter > 0 {
					send(fmt.Sprintf(
						"Throttled, will retry after %d seconds",
						retryAfter,
					))
					time.Sleep(time.Second)
					retryAfter -= 1
				}
			} else {
				return err
			}
		}
	}
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
			(!stringSliceContains(args.Languages, localLanguageCode) &&
				!stringSliceContains(args.Languages, remoteLanguageCode)) {
			continue
		}

		_, exists = remoteLanguages[remoteLanguageCode]
		if exists {
			languageCodesToPush = append(languageCodesToPush, remoteLanguageCode)
			pathsToPush = append(pathsToPush, path)
		} else {
			// if --all is set or -l is set and the code is in one of the
			// languages, we need to create the remote language
			if args.All || (len(args.Languages) > 0 &&
				(stringSliceContains(args.Languages, localLanguageCode) ||
					stringSliceContains(args.Languages, remoteLanguageCode))) {
				languageCodesToPush = append(languageCodesToPush,
					remoteLanguageCode)
				pathsToPush = append(pathsToPush, path)
				newLanguageCodes = append(newLanguageCodes, remoteLanguageCode)
			}
			continue
		}
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
