package txlib

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/gosuri/uilive"
	"github.com/pterm/pterm"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/transifex/cli/pkg/worker_pool"
)

type Message struct {
	i    int
	body string
}

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

	// Resources

	fmt.Print("# Getting info about resources\n\n")

	messages := make([]string, len(cfgResources))
	messageChannel := make(chan Message)

	pool := worker_pool.NewWorkerPool(args.Workers, len(cfgResources))
	sourceTaskChannel := make(chan SourceFileTask)
	translationTaskChannel := make(chan TranslationFileTask)
	for i, cfgResource := range cfgResources {
		pool.AddTask(
			ResourceTask{
				i,
				cfg,
				cfgResource,
				sourceTaskChannel,
				translationTaskChannel,
				&api,
				args,
				messageChannel,
			},
		)
	}
	pool.Start()

	var sourceFileTasks []SourceFileTask
	var translationFileTasks []TranslationFileTask

	w := uilive.New()
	w.Start()

	waitChannel := pool.Wait()
	exitfor := false
	for !exitfor {
		select {
		case sourceFileTask := <-sourceTaskChannel:
			sourceFileTask.i = len(sourceFileTasks)
			sourceFileTasks = append(sourceFileTasks, sourceFileTask)
		case translationFileTask := <-translationTaskChannel:
			translationFileTask.i = len(translationFileTasks)
			translationFileTasks = append(translationFileTasks, translationFileTask)
		case msg := <-messageChannel:
			messages[msg.i] = msg.body
			fmt.Fprintln(w, strings.Join(filterStringSlice(messages), "\n"))
			w.Flush()
		case <-waitChannel:
			exitfor = true
		}
	}
	w.Stop()
	fmt.Print("\n")

	if pool.IsAborted {
		fmt.Println("Aborted")
		return errors.New("Aborted")
	}

	// SourceFiles

	if len(sourceFileTasks) > 0 {
		fmt.Print("# Pushing source files\n\n")

		messages = make([]string, len(sourceFileTasks))

		w = uilive.New()
		w.Start()
		pool = worker_pool.NewWorkerPool(args.Workers, len(sourceFileTasks))
		for _, sourceFileTask := range sourceFileTasks {
			pool.AddTask(sourceFileTask)
		}
		pool.Start()

		exitfor = false
		waitChannel = pool.Wait()
		for !exitfor {
			select {
			case msg := <-messageChannel:
				messages[msg.i] = msg.body
				fmt.Fprintln(w, strings.Join(filterStringSlice(messages), "\n"))
				w.Flush()
			case <-waitChannel:
				exitfor = true
			}
		}
		w.Stop()
		fmt.Print("\n")

		if pool.IsAborted {
			fmt.Println("Aborted")
			return errors.New("Aborted")
		}
	}

	// Translations

	if len(translationFileTasks) > 0 {
		fmt.Print("# Pushing translations\n\n")

		messages = make([]string, len(translationFileTasks))

		w = uilive.New()
		w.Start()
		pool = worker_pool.NewWorkerPool(args.Workers, len(translationFileTasks))
		for _, translationFileTask := range translationFileTasks {
			pool.AddTask(translationFileTask)
		}
		pool.Start()

		exitfor = false
		waitChannel = pool.Wait()
		for !exitfor {
			select {
			case msg := <-messageChannel:
				messages[msg.i] = msg.body
				fmt.Fprintln(w, strings.Join(filterStringSlice(messages), "\n"))
				w.Flush()
			case <-waitChannel:
				exitfor = true
			}
		}
		w.Stop()

		if pool.IsAborted {
			fmt.Println("Aborted")
			return errors.New("Aborted")
		}
	}

	return nil
}

type ResourceTask struct {
	i                      int
	cfg                    *config.Config
	cfgResource            *config.Resource
	sourceTaskChannel      chan SourceFileTask
	translationTaskChannel chan TranslationFileTask
	api                    *jsonapi.Connection
	args                   PushCommandArguments
	messageChannel         chan Message
}

func (task ResourceTask) Run(pool *worker_pool.WorkerPool) {
	i := task.i
	cfg := task.cfg
	cfgResource := task.cfgResource
	sourceTaskChannel := task.sourceTaskChannel
	translationTaskChannel := task.translationTaskChannel
	api := task.api
	args := task.args
	messageChannel := task.messageChannel

	sendMessage := func(body string) {
		messageChannel <- Message{
			i,
			fmt.Sprintf(
				"%s.%s - %s",
				cfgResource.ProjectSlug,
				cfgResource.ResourceSlug,
				body,
			),
		}
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
			return
		}
	}

	sendMessage("Getting stats")
	remoteStats, err := getRemoteStats(api, resource, args)
	if err != nil {
		sendMessage(fmt.Sprintf("Error while fetching stats, %s", err))
		return
	}
	if args.Source || !args.Translation {
		sourceTaskChannel <- SourceFileTask{
			-1,
			api,
			resource,
			cfgResource.SourceFile,
			remoteStats,
			args,
			resourceIsNew,
			messageChannel,
		}
	}
	if args.Translation { // -t flag is set
		reverseLanguageMappings := makeReverseLanguageMappings(*cfg, *cfgResource)
		overrides := cfgResource.Overrides
		projectRelationship, err := resource.Fetch("project")
		if err != nil {
			sendMessage(fmt.Sprintf("Error while fetching project, %s", err))
			return
		}
		project := projectRelationship.DataSingular

		sendMessage("Fetching remote languages")
		// TODO see if we can figure our remote languages from stats
		remoteLanguages, err := txapi.GetProjectLanguages(project)
		if err != nil {
			sendMessage(fmt.Sprintf("Error while fetching remote languages, %s", err))
			return
		}
		curDir, err := os.Getwd()
		if err != nil {
			return
		}
		fileFilter := cfgResource.FileFilter
		if err := isFileFilterValid(fileFilter); err != nil {
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
			return
		}
		if len(newLanguageCodes) > 0 {
			sendMessage("Creating new remote languages")
			remoteLanguages, err = createNewLanguages(
				api, project, remoteLanguages, newLanguageCodes,
			)
			if err != nil {
				return
			}
		}
		for i := range languageCodesToPush {
			languageCode := languageCodesToPush[i]
			path := pathsToPush[i]
			_, exists := remoteLanguages[languageCode]
			if !exists {
				continue
			}

			translationTaskChannel <- TranslationFileTask{
				-1,
				api,
				languageCode,
				path,
				resource,
				remoteLanguages,
				args,
				messageChannel,
			}
		}
	}
	sendMessage("Done")
}

type SourceFileTask struct {
	i              int
	api            *jsonapi.Connection
	resource       *jsonapi.Resource
	sourceFile     string
	remoteStats    map[string]*jsonapi.Resource
	args           PushCommandArguments
	resourceIsNew  bool
	messageChannel chan Message
}

func (task SourceFileTask) Run(pool *worker_pool.WorkerPool) {
	i := task.i
	api := task.api
	resource := task.resource
	sourceFile := task.sourceFile
	remoteStats := task.remoteStats
	args := task.args
	resourceIsNew := task.resourceIsNew
	messageChannel := task.messageChannel

	sendMessage := func(body string) {
		messageChannel <- Message{i, fmt.Sprintf("%s - %s", resource.Id, body)}
	}

	doError := func(err error) {
		sendMessage(fmt.Sprintf("Error: %s", err))
		if !args.Skip {
			pool.Abort()
		}
	}

	if sourceFile == "" {
		return
	}
	file, err := os.Open(sourceFile)
	if err != nil {
		doError(err)
		return
	}
	defer file.Close()

	// Only check timestamps if -f isn't set and if resource isn't new
	if !args.Force && !resourceIsNew {
		// Project should already be pre-fetched
		projectRelationship, err := resource.Fetch("project")
		if err != nil {
			doError(err)
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
			doError(err)
			return
		}
	}
	sendMessage("Uploading file")
	sourceUpload, err := txapi.UploadSource(api, resource, file)
	if err != nil {
		doError(err)
		return
	}
	if sourceUpload == nil {
		return
	}
	duration := time.Duration(1) * time.Second
	sendMessage("Polling")
	err = txapi.PollSourceUpload(sourceUpload, duration)
	if err != nil {
		doError(err)
		return
	}
	sendMessage("Done")
}

type TranslationFileTask struct {
	i               int
	api             *jsonapi.Connection
	languageCode    string
	path            string
	resource        *jsonapi.Resource
	remoteLanguages map[string]*jsonapi.Resource
	args            PushCommandArguments
	messageChannel  chan Message
}

func (task TranslationFileTask) Run(pool *worker_pool.WorkerPool) {
	i := task.i
	api := task.api
	languageCode := task.languageCode
	path := task.path
	resource := task.resource
	remoteLanguages := task.remoteLanguages
	args := task.args
	messageChannel := task.messageChannel

	sendMessage := func(body string) {
		messageChannel <- Message{
			i, fmt.Sprintf("%s (%s) - %s", resource.Id, languageCode, body),
		}
	}

	sendMessage("Uploading file")
	upload, err := pushTranslation(
		api, languageCode, path, resource, remoteLanguages, args,
	)
	if err != nil {
		return
	}

	duration := time.Duration(1) * time.Second
	sendMessage("Polling")
	err = txapi.PollTranslationUpload(upload, duration)
	if err != nil {
		if !args.Skip {
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

func filterStringSlice(src []string) []string {
	var dst []string
	for _, msg := range src {
		if len(msg) > 0 {
			dst = append(dst, msg)
		}
	}
	return dst
}
