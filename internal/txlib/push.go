package txlib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/transifex/cli/pkg/worker_pool"
)

type PushCommandArguments struct {
	Source               bool
	Translation          bool
	Force                bool
	Skip                 bool
	Xliff                bool
	Languages            []string
	ResourceIds          []string
	UseGitTimestamps     bool
	Branch               string
	Base                 string
	All                  bool
	Workers              int
	Silent               bool
	ReplaceEditedStrings bool
	KeepTranslations     bool
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

	if !args.Silent {
		fmt.Print("# Getting info about resources\n\n")
	}

	pool := worker_pool.New(args.Workers, len(cfgResources), args.Silent)
	sourceTaskChannel := make(chan *SourceFilePushTask)
	translationTaskChannel := make(chan *TranslationFileTask)
	targetLanguagesChannel := make(chan TargetLanguageMessage)
	for _, cfgResource := range cfgResources {
		pool.Add(
			&ResourcePushTask{
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

	var sourceFileTasks []*SourceFilePushTask
	var translationFileTasks []*TranslationFileTask
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
		return errors.New("Aborted")
	}
	if args.Silent {
		var names []string
		for _, cfgResource := range cfgResources {
			names = append(names, fmt.Sprintf(
				"%s.%s",
				cfgResource.ProjectSlug,
				cfgResource.ResourceSlug,
			))
		}
		fmt.Printf("Got info about resources: %s\n", strings.Join(names, ", "))
	}

	// Step 2: Create missing remote target languages

	if len(targetLanguages) > 0 {
		if !args.Silent {
			fmt.Print("\n# Create missing remote target languages\n\n")
		}

		pool = worker_pool.New(args.Workers, len(targetLanguages), args.Silent)
		for projectId, languages := range targetLanguages {
			sort.Slice(languages, func(i, j int) bool {
				return languages[i] < languages[j]
			})
			pool.Add(&LanguagePushTask{projects[projectId], languages, args})
		}
		pool.Start()
		<-pool.Wait()
		if pool.IsAborted {
			return errors.New("Aborted")
		}
		if args.Silent {
			var names []string
			for projectId, languages := range targetLanguages {
				parts := strings.Split(projectId, ":")
				projectSlug := parts[1]

				var languageCodes []string
				languageCodes = append(languageCodes, languages...)
				names = append(names, fmt.Sprintf(
					"%s: %s",
					projectSlug,
					strings.Join(languageCodes, ", "),
				))
			}
			fmt.Printf(
				"Created missing remote target languages: %s\n",
				strings.Join(names, ", "),
			)
		}
	}

	// Step 3: SourceFiles

	if len(sourceFileTasks) > 0 {
		if !args.Silent {
			fmt.Print("\n# Pushing source files\n\n")
		}

		sort.Slice(sourceFileTasks, func(i, j int) bool {
			return sourceFileTasks[i].resource.Id < sourceFileTasks[j].resource.Id
		})
		pool = worker_pool.New(args.Workers, len(sourceFileTasks), args.Silent)
		for _, sourceFileTask := range sourceFileTasks {
			pool.Add(sourceFileTask)
		}
		pool.Start()
		<-pool.Wait()

		if pool.IsAborted {
			return errors.New("Aborted")
		}
		if args.Silent {
			var names []string
			for _, sourceFileTask := range sourceFileTasks {
				parts := strings.Split(sourceFileTask.resource.Id, ":")
				resourceSlug := parts[5]
				names = append(names, resourceSlug)
			}
			fmt.Printf(
				"Pushed source files for: %s\n",
				strings.Join(names, ", "),
			)
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
		if !args.Silent {
			fmt.Print("\n# Pushing translations\n\n")
		}

		pool = worker_pool.New(args.Workers, len(translationFileTasks), args.Silent)
		for _, translationFileTask := range translationFileTasks {
			pool.Add(translationFileTask)
		}
		pool.Start()
		<-pool.Wait()

		if pool.IsAborted {
			return errors.New("Aborted")
		}
		if args.Silent {
			var names []string
			for _, translationFileTask := range translationFileTasks {
				parts := strings.Split(translationFileTask.resource.Id, ":")
				resourceSlug := parts[5]
				names = append(names, fmt.Sprintf(
					"%s: %s",
					resourceSlug,
					translationFileTask.languageCode,
				))
			}
			fmt.Printf("Pushed translations: %s\n", strings.Join(names, ", "))
		}
	}

	return nil
}

type TargetLanguageMessage struct {
	project    *jsonapi.Resource
	languageId string
}

type ResourcePushTask struct {
	cfg                    *config.Config
	cfgResource            *config.Resource
	sourceTaskChannel      chan *SourceFilePushTask
	translationTaskChannel chan *TranslationFileTask
	api                    *jsonapi.Connection
	args                   PushCommandArguments
	targetLanguagesChannel chan TargetLanguageMessage
}

func (task *ResourcePushTask) Run(send func(string), abort func()) {
	cfg := task.cfg
	cfgResource := task.cfgResource
	sourceTaskChannel := task.sourceTaskChannel
	translationTaskChannel := task.translationTaskChannel
	api := task.api
	args := task.args
	targetLanguagesChannel := task.targetLanguagesChannel

	sendMessage := func(body string, force bool) {
		if args.Silent && !force {
			return
		}
		send(fmt.Sprintf(
			"%s.%s - %s",
			cfgResource.ProjectSlug,
			cfgResource.ResourceSlug,
			body,
		))
	}
	sendMessage("Getting info", false)
	resource, err := txapi.GetResourceById(api, cfgResource.GetAPv3Id())
	if err != nil {
		sendMessage(fmt.Sprintf("Error while fetching resource: %s", err), true)
		if !args.Skip {
			abort()
		}
		return
	}

	resourceIsNew := resource == nil
	if resourceIsNew {
		if args.Translation && !args.Source {
			sendMessage(
				"You are attempting to push translations for a resource that doesn't "+
					"exist yet",
				true,
			)
			if !args.Skip {
				abort()
			}
			return
		}
		sendMessage("Resource does not exist; creating", false)
		if cfgResource.Type == "" {
			sendMessage("Error: Cannot create resource, i18n type is unknown", true)
			if !args.Skip {
				abort()
			}
			return
		}
		var resourceName string
		var baseResourceId string

		if args.Branch == "" {
			resourceName = cfgResource.GetName()
		} else {
			resourceName = fmt.Sprintf(
				"%s (branch %s)",
				cfgResource.GetName(),
				args.Branch,
			)

			baseResourceSlug := getBaseResourceSlug(cfgResource, args.Branch, args.Base)

			baseResourceId = fmt.Sprintf(
				"o:%s:p:%s:r:%s",
				cfgResource.OrganizationSlug,
				cfgResource.ProjectSlug,
				baseResourceSlug,
			)
			baseResource, err := txapi.GetResourceById(api, baseResourceId)
			if err != nil {
				sendMessage(fmt.Sprintf("Error while fetching base resource: %s", err), true)
				if !args.Skip {
					abort()
				}
				return
			}
			if args.Base != "-1" {
				if baseResource == nil {
					sendMessage(fmt.Sprintf("Base Resource does not exist: %s", baseResourceId), true)
					if !args.Skip {
						abort()
					}
					return
				}
			} else {
				if baseResource == nil {
					baseResourceId = ""
				}
			}
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
			cfgResource.Type,
			baseResourceId,
		)
		if err != nil {
			sendMessage(fmt.Sprintf("Error while creating resource, %s", err), true)
			if !args.Skip {
				abort()
			}
			return
		}
	} else {
		if args.Branch != "" && args.Base != "-1" {
			baseResourceSlug := getBaseResourceSlug(cfgResource, args.Branch, args.Base)

			baseResourceId := fmt.Sprintf(
				"o:%s:p:%s:r:%s",
				cfgResource.OrganizationSlug,
				cfgResource.ProjectSlug,
				baseResourceSlug,
			)

			applyBranchToResources([]*config.Resource{cfgResource}, args.Branch)
			resource.SetRelated("base", &jsonapi.Resource{Type: "resources", Id: baseResourceId})
			err = resource.Save([]string{"base"})
			if err != nil {
				sendMessage(err.Error(), true)
				if !args.Skip {
					abort()
				}
				return
			}
		}
	}

	sendMessage("Getting stats", false)
	projectRelationship, err := resource.Fetch("project")
	if err != nil {
		sendMessage(err.Error(), true)
		if !args.Skip {
			abort()
		}
		return
	}
	project := projectRelationship.DataSingular
	sourceLanguageRelationship, exists := project.Relationships["source_language"]
	if !exists {
		sendMessage(
			"Invalid API response, project does not have a 'source_language' "+
				"relationship",
			true,
		)
		if !args.Skip {
			abort()
		}
		return
	}
	sourceLanguage := sourceLanguageRelationship.DataSingular
	var remoteStats map[string]*jsonapi.Resource
	if args.Translation {
		remoteStats, err = txapi.GetResourceStats(api, resource, nil)
	} else {
		remoteStats, err = txapi.GetResourceStats(api, resource, sourceLanguage)
	}
	if err != nil {
		sendMessage(fmt.Sprintf("Error while fetching stats, %s", err), true)
		if !args.Skip {
			abort()
		}
		return
	}
	if args.Source || !args.Translation {
		sourceTaskChannel <- &SourceFilePushTask{
			api,
			resource,
			cfgResource.SourceFile,
			remoteStats[sourceLanguage.Id],
			args,
			resourceIsNew,
			args.ReplaceEditedStrings || cfgResource.ReplaceEditedStrings,
			args.KeepTranslations || cfgResource.KeepTranslations,
		}
	}
	if args.Translation { // -t flag is set
		localToRemoteLanguageMappings := makeLocalToRemoteLanguageMappings(
			*cfg,
			*cfgResource,
		)
		overrides := cfgResource.Overrides

		sendMessage("Fetching remote languages", false)
		curDir, err := os.Getwd()
		if err != nil {
			sendMessage(err.Error(), true)
			if !args.Skip {
				abort()
			}
			return
		}
		fileFilter := cfgResource.FileFilter
		err = checkFileFilter(fileFilter)
		if err != nil {
			sendMessage(err.Error(), true)
			if !args.Skip {
				abort()
			}
			return
		}
		if args.Xliff {
			fileFilter = fmt.Sprintf("%s.xlf", fileFilter)
		}

		paths, newLanguageCodes, err := getFilesToPush(
			curDir, fileFilter, localToRemoteLanguageMappings,
			remoteStats, overrides, args, resourceIsNew,
		)
		if err != nil {
			sendMessage(err.Error(), true)
			if !args.Skip {
				abort()
			}
			return
		}

		allLanguages, err := txapi.GetLanguages(api)
		if err != nil {
			sendMessage(err.Error(), true)
			abort()
			return
		}
		for _, languageCode := range newLanguageCodes {
			_, exists := allLanguages[languageCode]
			if !exists || fmt.Sprintf("l:%s", languageCode) == sourceLanguage.Id {
				continue
			}
			targetLanguagesChannel <- TargetLanguageMessage{project, languageCode}
		}
		for languageCode, path := range paths {
			_, exists := allLanguages[languageCode]
			if !exists || fmt.Sprintf("l:%s", languageCode) == sourceLanguage.Id {
				continue
			}

			translationTaskChannel <- &TranslationFileTask{
				api,
				languageCode,
				path,
				resource,
				args,
				remoteStats,
				resourceIsNew,
			}
		}
	}
	sendMessage("Done", false)
}

type LanguagePushTask struct {
	project   *jsonapi.Resource
	languages []string
	args      PushCommandArguments
}

func (task *LanguagePushTask) Run(send func(string), abort func()) {
	project := task.project
	languages := task.languages
	args := task.args

	parts := strings.Split(project.Id, ":")

	sendMessage := func(body string, force bool) {
		if args.Silent && !force {
			return
		}
		send(fmt.Sprintf(
			"%s (%s) - %s",
			parts[3],
			strings.Join(languages, ", "),
			body,
		))
	}
	sendMessage("Pushing", false)

	var payload []*jsonapi.Resource
	for _, language := range languages {
		payload = append(payload, &jsonapi.Resource{
			Type: "languages",
			Id:   fmt.Sprintf("l:%s", language),
		})
	}
	err := project.Add("languages", payload)
	if err != nil {
		sendMessage(err.Error(), true)
		abort()
		return
	}

	sendMessage("Done", false)
}

type SourceFilePushTask struct {
	api                  *jsonapi.Connection
	resource             *jsonapi.Resource
	sourceFile           string
	remoteStats          *jsonapi.Resource
	args                 PushCommandArguments
	resourceIsNew        bool
	replaceEditedStrings bool
	keepTranslations     bool
}

func (task *SourceFilePushTask) Run(send func(string), abort func()) {
	api := task.api
	resource := task.resource
	sourceFile := task.sourceFile
	remoteStats := task.remoteStats
	args := task.args
	resourceIsNew := task.resourceIsNew
	replaceEditedStrings := task.replaceEditedStrings
	keepTranslations := task.keepTranslations

	parts := strings.Split(resource.Id, ":")
	sendMessage := func(body string, force bool) {
		if args.Silent && !force {
			return
		}
		send(fmt.Sprintf("%s.%s - %s", parts[3], parts[5], body))
	}

	file, err := os.Open(sourceFile)
	if err != nil {
		sendMessage(err.Error(), true)
		if !args.Skip {
			abort()
		}
		return
	}
	defer file.Close()

	// Only check timestamps if -f isn't set and if resource isn't new
	if !args.Force && !resourceIsNew {
		// Project should already be pre-fetched
		skip, err := shouldSkipPush(
			sourceFile, remoteStats, args.UseGitTimestamps,
		)
		if skip {
			sendMessage("Skipping", false)
			return
		}
		if err != nil {
			sendMessage(err.Error(), true)
			if !args.Skip {
				abort()
			}
			return
		}
	}

	// Uploading file

	var sourceUpload *jsonapi.Resource
	err = handleThrottling(
		func() error {
			var err error
			sourceUpload, err = txapi.UploadSource(
				api, resource, file, replaceEditedStrings, keepTranslations,
			)
			return err
		},
		"Uploading file",
		func(msg string) { sendMessage(msg, false) },
	)
	if err != nil {
		sendMessage(err.Error(), true)
		if !args.Skip {
			abort()
		}
		return
	}

	// Polling

	err = handleThrottling(
		func() error {
			return txapi.PollSourceUpload(sourceUpload)
		},
		"",
		func(msg string) { sendMessage(msg, false) },
	)
	if err != nil {
		sendMessage(err.Error(), true)
		if !args.Skip {
			abort()
		}
		return
	}

	sendMessage("Done", false)
}

type TranslationFileTask struct {
	api           *jsonapi.Connection
	languageCode  string
	path          string
	resource      *jsonapi.Resource
	args          PushCommandArguments
	remoteStats   map[string]*jsonapi.Resource
	resourceIsNew bool
}

func (task *TranslationFileTask) Run(send func(string), abort func()) {
	api := task.api
	languageCode := task.languageCode
	path := task.path
	resource := task.resource
	args := task.args
	remoteStats := task.remoteStats
	resourceIsNew := task.resourceIsNew

	parts := strings.Split(resource.Id, ":")
	cyan := color.New(color.FgCyan).SprintFunc()
	sendMessage := func(body string, force bool) {
		if args.Silent && !force {
			return
		}
		send(fmt.Sprintf(
			"%s.%s %s - %s", parts[3], parts[5],
			cyan("["+languageCode+"]"), body,
		))
	}

	// Only check timestamps if -f isn't set and if resource isn't new
	if !args.Force && !resourceIsNew {
		languageId := fmt.Sprintf("l:%s", languageCode)
		remoteStat, exists := remoteStats[languageId]
		if exists {
			skip, err := shouldSkipPush(path, remoteStat, args.UseGitTimestamps)
			if err != nil {
				sendMessage(err.Error(), true)
				if !args.Skip {
					abort()
				}
				return
			}
			if skip {
				sendMessage("Skipping because remote file is newer than local", false)
				return
			}
		}
	}

	// Uploading file

	var upload *jsonapi.Resource
	err := handleThrottling(
		func() error {
			var err error
			upload, err = pushTranslation(
				api, languageCode, path, resource, args,
			)
			return err
		},
		"Uploading file",
		func(msg string) { sendMessage(msg, false) },
	)
	if err != nil {
		sendMessage(err.Error(), true)
		if !args.Skip {
			abort()
		}
		return
	}

	// Polling
	err = handleThrottling(
		func() error {
			return txapi.PollTranslationUpload(upload)
		},
		"",
		func(msg string) { sendMessage(msg, false) },
	)
	if err != nil {
		sendMessage(err.Error(), true)
		if !args.Skip {
			abort()
		}
		return
	}
	sendMessage("Done", false)
}

func getFilesToPush(
	curDir, fileFilter string,
	localToRemoteLanguageMappings map[string]string,
	remoteStats map[string]*jsonapi.Resource,
	overrides map[string]string,
	args PushCommandArguments,
	resourceIsNew bool,
) (map[string]string, []string, error) {
	paths := make(map[string]string)
	var newLanguageCodes []string

	allLocalLanguages := searchFileFilter(curDir, fileFilter)

	if len(overrides) > 0 {
		for languageCode, customPath := range overrides {
			// Add the Resource file filter overrides per lang
			path := filepath.Join(curDir, customPath)
			// In case of xliff/json add the extension
			if args.Xliff {
				path = fmt.Sprintf("%s.xlf", path)
			}
			allLocalLanguages[languageCode] = path
		}
	}

	for localLanguageCode, path := range allLocalLanguages {
		remoteLanguageCode, exists := localToRemoteLanguageMappings[localLanguageCode]
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

		remoteLanguageId := fmt.Sprintf("l:%s", remoteLanguageCode)
		_, exists = remoteStats[remoteLanguageId]
		if exists {
			paths[remoteLanguageCode] = path
		} else {
			// if --all is set or -l is set and the code is in one of the
			// languages, we need to create the remote language
			if args.All || (len(args.Languages) > 0 &&
				(stringSliceContains(args.Languages, localLanguageCode) ||
					stringSliceContains(args.Languages, remoteLanguageCode))) {
				paths[remoteLanguageCode] = path
				newLanguageCodes = append(newLanguageCodes, remoteLanguageCode)
			}
			continue
		}
	}
	return paths, newLanguageCodes, nil
}

func pushTranslation(
	api *jsonapi.Connection,
	languageCode, path string,
	resource *jsonapi.Resource,
	args PushCommandArguments,
) (*jsonapi.Resource, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	language := &jsonapi.Resource{
		API:  api,
		Type: "languages",
		Id:   fmt.Sprintf("l:%s", languageCode),
	}
	upload, err := txapi.UploadTranslation(api, resource, language, file, args.Xliff)
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
