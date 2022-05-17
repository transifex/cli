package txlib

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/pterm/pterm"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/transifex/cli/pkg/worker_pool"
)

func PushParallelCommand(
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
		if err := isFileFilterValid(fileFilter); err != nil {
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
	sourceUpload, err := txapi.UploadSource(api, resource, file)
	if err != nil {
		sendMessage(err.Error())
		if !args.Skip {
			abort()
		}
		return
	}
	if sourceUpload == nil {
		return
	}
	duration := time.Duration(1) * time.Second
	sendMessage("Polling")
	err = txapi.PollSourceUpload(sourceUpload, duration)
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
}

func (task TranslationFileTask) Run(send func(string), abort func()) {
	api := task.api
	languageCode := task.languageCode
	path := task.path
	resource := task.resource
	remoteLanguages := task.remoteLanguages
	args := task.args

	parts := strings.Split(resource.Id, ":")

	sendMessage := func(body string) {
		send(fmt.Sprintf("%s.%s [%s] - %s", parts[3], parts[5], languageCode, body))
	}

	sendMessage("Uploading file")
	upload, err := pushTranslation(
		api, languageCode, path, resource, remoteLanguages, args,
	)
	if err != nil {
		sendMessage(err.Error())
		if !args.Skip {
			abort()
		}
		return
	}

	duration := time.Duration(1) * time.Second
	sendMessage("Polling")
	err = txapi.PollTranslationUpload(upload, duration)
	if err != nil {
		sendMessage(err.Error())
		if !args.Skip {
			abort()
			return
		}
		return
	}
	sendMessage("Done")
}

func figureOutBranch(branch string) string {
	var result string
	if branch == "-1" {
		result = ""
	} else if branch == "" {
		result = getGitBranch()
		if result == "" {
			pterm.Warning.Println("Couldn't find branch information")
		}
	}
	if result != "" {
		pterm.Info.Printf("Using branch '%s'\n", result)
	}
	return result
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
				cfgResource.ResourceSlug,
				slug.Make(branch))
		}
	}
}

func stringSliceContains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
