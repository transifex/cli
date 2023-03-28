package txlib

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/transifex/cli/pkg/worker_pool"
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
	Workers           int
	Silent            bool
	Pseudo            bool
}

func PullCommand(
	cfg *config.Config,
	api *jsonapi.Connection,
	args *PullCommandArguments,
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

	if !args.Silent {
		fmt.Print("# Getting info about resources\n\n")
	}

	filePullTaskChannel := make(chan *FilePullTask)
	var filePullTasks []*FilePullTask
	pool := worker_pool.New(args.Workers, len(cfgResources), args.Silent)
	for _, cfgResource := range cfgResources {
		pool.Add(&ResourcePullTask{cfgResource, api, args, filePullTaskChannel, cfg})
	}
	pool.Start()

	waitChanel := pool.Wait()
	exitfor := false
	for !exitfor {
		select {
		case task := <-filePullTaskChannel:
			filePullTasks = append(filePullTasks, task)
		case <-waitChanel:
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

	if len(filePullTasks) > 0 {
		sort.Slice(filePullTasks, func(i, j int) bool {
			left := filePullTasks[i]
			right := filePullTasks[j]
			if left.resource.Id != right.resource.Id {
				return left.resource.Id < right.resource.Id
			} else {
				return left.languageCode < right.languageCode
			}
		})

		if !args.Silent {
			fmt.Print("\n# Pulling files\n\n")
		}
		pool = worker_pool.New(args.Workers, len(filePullTasks), args.Silent)
		for _, task := range filePullTasks {
			pool.Add(task)
		}
		pool.Start()
		<-pool.Wait()

		if pool.IsAborted {
			return errors.New("Aborted")
		}
		if args.Silent {
			var names []string
			for _, filePullTask := range filePullTasks {
				var languageCode string
				if filePullTask.languageCode == "" {
					languageCode = "source"
				} else {
					languageCode = filePullTask.languageCode
				}
				names = append(names, fmt.Sprintf(
					"%s: %s",
					filePullTask.cfgResource.ResourceSlug,
					languageCode,
				))
			}
			fmt.Printf("Pulled files: %s\n", strings.Join(names, ", "))
		}
	}

	return nil
}

type ResourcePullTask struct {
	cfgResource         *config.Resource
	api                 *jsonapi.Connection
	args                *PullCommandArguments
	filePullTaskChannel chan *FilePullTask
	cfg                 *config.Config
}

func (task *ResourcePullTask) Run(send func(string), abort func()) {
	cfgResource := task.cfgResource
	api := task.api
	args := task.args
	filePullTaskChannel := task.filePullTaskChannel
	cfg := task.cfg

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

	localToRemoteLanguageMappings := makeLocalToRemoteLanguageMappings(
		*cfg,
		*cfgResource,
	)
	remoteToLocalLanguageMappings := makeRemoteToLocalLanguageMappings(
		localToRemoteLanguageMappings,
	)

	var err error
	resource, err := txapi.GetResourceById(api, cfgResource.GetAPv3Id())
	if err != nil {
		sendMessage(err.Error(), true)
		if !args.Skip {
			abort()
		}
		return
	}
	if resource == nil {
		sendMessage(
			fmt.Sprintf(
				"Resource %s.%s does not exist",
				cfgResource.ProjectSlug,
				cfgResource.ResourceSlug,
			),
			true,
		)
		return
	}

	projectRelationship, err := resource.Fetch("project")
	if err != nil {
		sendMessage(err.Error(), true)
		if !args.Skip {
			abort()
		}
		return
	}
	project := projectRelationship.DataSingular
	sourceLanguage := project.Relationships["source_language"].DataSingular

	var stats map[string]*jsonapi.Resource
	if args.Source && !args.Translations {
		stats, err = txapi.GetResourceStats(api, resource, sourceLanguage)
	} else {
		stats, err = txapi.GetResourceStats(api, resource, nil)
	}
	if err != nil {
		sendMessage(err.Error(), true)
		if !args.Skip {
			abort()
		}
		return
	}

	if args.Source {
		filePullTaskChannel <- &FilePullTask{
			cfgResource,
			"",
			args,
			api,
			resource,
			stats[sourceLanguage.Id],
			"",
			remoteToLocalLanguageMappings,
		}
	}

	if args.Translations || !args.Source {
		languageInfo := make(map[string]*struct {
			filePath string
			stats    *jsonapi.Resource
		})

		// Local stuff
		err = checkFileFilter(cfgResource.FileFilter)
		if err != nil {
			sendMessage(err.Error(), true)
			if !args.Skip {
				abort()
			}
			return
		}
		fileFilter := setFileTypeExtensions(args.FileType, cfgResource.FileFilter)
		if args.Pseudo {
			fileFilter = strings.Replace(
				fileFilter,
				"<lang>",
				"<lang>_pseudo",
				-1,
			)
		}
		localFiles := searchFileFilter(".", fileFilter)

		for localLanguageCode, filePath := range cfgResource.Overrides {
			filePath = setFileTypeExtensions(args.FileType, filePath)
			remoteLanguageCode, exists := localToRemoteLanguageMappings[localLanguageCode]
			if !exists {
				remoteLanguageCode = localLanguageCode
			}
			if len(args.Languages) > 0 &&
				(!stringSliceContains(args.Languages, remoteLanguageCode) &&
					!stringSliceContains(args.Languages, localLanguageCode)) {
				continue
			}
			localFiles[remoteLanguageCode] = filePath
		}

		for localLanguageCode, filePath := range localFiles {
			remoteLanguageCode, exists := localToRemoteLanguageMappings[localLanguageCode]
			if !exists {
				remoteLanguageCode = localLanguageCode
			}
			if len(args.Languages) > 0 &&
				(!stringSliceContains(args.Languages, remoteLanguageCode) &&
					!stringSliceContains(args.Languages, localLanguageCode)) {
				continue
			}
			languageId := fmt.Sprintf("l:%s", remoteLanguageCode)
			languageInfo[languageId] = &struct {
				filePath string
				stats    *jsonapi.Resource
			}{filePath: filePath}
		}

		// Remote stuff
		for languageId, stat := range stats {
			parts := strings.Split(languageId, ":")
			remoteLanguageCode := parts[1]
			localLanguageCode, exists := remoteToLocalLanguageMappings[remoteLanguageCode]
			if !exists {
				localLanguageCode = remoteLanguageCode
			}
			if len(args.Languages) > 0 &&
				(!stringSliceContains(args.Languages, remoteLanguageCode) &&
					!stringSliceContains(args.Languages, localLanguageCode)) {
				continue
			}
			info, exists := languageInfo[languageId]
			if exists {
				info.stats = stat
			} else {
				languageInfo[languageId] = &struct {
					filePath string
					stats    *jsonapi.Resource
				}{stats: stat}
			}
		}

		for languageId, info := range languageInfo {
			if languageId == sourceLanguage.Id || info.stats == nil {
				continue
			}
			parts := strings.Split(languageId, ":")
			languageCode := parts[1]
			filePullTaskChannel <- &FilePullTask{
				cfgResource,
				languageCode,
				args,
				api,
				resource,
				info.stats,
				info.filePath,
				remoteToLocalLanguageMappings,
			}
		}
	}
	sendMessage("Done", false)
}

type FilePullTask struct {
	cfgResource                   *config.Resource
	languageCode                  string
	args                          *PullCommandArguments
	api                           *jsonapi.Connection
	resource                      *jsonapi.Resource
	stats                         *jsonapi.Resource
	filePath                      string
	remoteToLocalLanguageMappings map[string]string
}

func (task *FilePullTask) Run(send func(string), abort func()) {
	cfgResource := task.cfgResource
	languageCode := task.languageCode
	args := task.args
	api := task.api
	resource := task.resource
	stats := task.stats
	filePath := task.filePath
	remoteToLocalLanguageMapping := task.remoteToLocalLanguageMappings

	sendMessage := func(body string, force bool) {
		if args.Silent && !force {
			return
		}
		var code string
		if languageCode == "" {
			code = "source"
		} else {
			code = languageCode
		}

		cyan := color.New(color.FgCyan).SprintFunc()
		send(fmt.Sprintf(
			"%s.%s %s - %s",
			cfgResource.ProjectSlug,
			cfgResource.ResourceSlug,
			cyan("["+code+"]"),
			body,
		))
	}
	sendMessage("Pulling file", false)

	if languageCode == "" {
		sourceFile := setFileTypeExtensions(args.FileType, cfgResource.SourceFile)

		_, err := os.Stat(sourceFile)
		if err == nil && args.DisableOverwrite {
			sendMessage("Disable overwrite enabled, creating a .new file", false)
			sourceFile = sourceFile + ".new"
		}

		if !args.Force {
			shouldSkip, err := shouldSkipResourceDownload(
				sourceFile,
				resource,
				args.UseGitTimestamps,
			)
			if err != nil {
				sendMessage(err.Error(), true)
				if !args.Skip {
					abort()
				}
				return
			}
			if shouldSkip {
				sendMessage("Local file is newer than remote, skipping", false)
				return
			}
		}

		// Creating download job

		var download *jsonapi.Resource
		err = handleThrottling(
			func() error {
				var err error
				download, err = txapi.CreateResourceStringsAsyncDownload(
					api,
					resource,
					args.ContentEncoding,
					args.FileType,
					args.Pseudo,
				)
				return err
			},
			"Creating download job",
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
				return txapi.PollResourceStringsDownload(download, sourceFile)
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
	} else {
		if filePath != "" {
			// Remote language file exists and so does local
			if args.DisableOverwrite {
				sendMessage("Disable overwrite enabled, creating a .new file", false)
				filePath = filePath + ".new"
			}
		} else {
			// Remote language file exists but local does not
			remoteLanguageCode := languageCode
			localLanguageCode, exists := remoteToLocalLanguageMapping[remoteLanguageCode]
			if !exists {
				localLanguageCode = remoteLanguageCode
			}
			if !args.All &&
				(!stringSliceContains(args.Languages, remoteLanguageCode) &&
					!stringSliceContains(args.Languages, localLanguageCode)) {
				sendMessage("File was not found locally, skipping", false)
				return
			}
			pseudo_postfix := ""
			if args.Pseudo {
				pseudo_postfix = "_pseudo"
			}
			filePath = strings.Replace(
				cfgResource.FileFilter,
				"<lang>",
				localLanguageCode+pseudo_postfix,
				-1,
			)
			filePath = setFileTypeExtensions(args.FileType, filePath)
		}
		minimumPerc := args.MinimumPercentage
		if minimumPerc == -1 {
			if cfgResource.MinimumPercentage > -1 {
				minimumPerc = cfgResource.MinimumPercentage
			}
		}
		shouldSkip, feedbackMessage, err := shouldSkipDownload(
			filePath,
			stats,
			args.UseGitTimestamps,
			args.Mode,
			minimumPerc,
			args.Force,
		)
		if err != nil {
			sendMessage(err.Error(), true)
			if !args.Skip {
				abort()
			}
			return
		}
		if shouldSkip {
			sendMessage(feedbackMessage, false)
			return
		}

		// Creating download job

		var download *jsonapi.Resource
		err = handleThrottling(
			func() error {
				var err error
				if args.Pseudo {
					download, err = txapi.CreateResourceStringsAsyncDownload(
						api,
						resource,
						args.ContentEncoding,
						args.FileType,
						args.Pseudo,
					)
				} else {
					download, err = txapi.CreateTranslationsAsyncDownload(
						api,
						resource,
						languageCode,
						args.ContentEncoding,
						args.FileType,
						args.Mode,
					)
				}
				return err
			},
			"Creating download job",
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
				return txapi.PollTranslationDownload(download, filePath)
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
	}
	if !args.DisableOverwrite {
		sendMessage("Done", false)
	} else {
		sendMessage("Done. File created with a .new extension", false)
	}

}

func shouldSkipDownload(
	path string, remoteStat *jsonapi.Resource, useGitTimestamps bool,
	mode string, minimum_perc int, force bool,
) (bool, string, error) {
	var localTime time.Time
	var feedbackMessage = ""
	var remoteStatAttributes txapi.ResourceLanguageStatsAttributes
	err := remoteStat.MapAttributes(&remoteStatAttributes)
	if err != nil {
		return false, feedbackMessage, err
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
			feedbackMessage = "Minimum translation completion threshold not satisfied, skipping"
			return true, feedbackMessage, nil
		}
	}

	if !force {
		if useGitTimestamps {
			// TODO: check if parent folder is repo
			localTime = getLastCommitDate(path)
			if localTime == (time.Time{}) {
				return shouldSkipDownload(path, remoteStat,
					false, mode, minimum_perc, force)
			}
		} else {
			localStat, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					return false, feedbackMessage, nil
				}
				return false, feedbackMessage, err
			}
			localTime = localStat.ModTime().UTC()
		}

		remoteTime, err := time.Parse(time.RFC3339,
			remoteStatAttributes.LastUpdate)
		if err != nil {
			return false, feedbackMessage, err
		}

		// Don't pull if local file is newer than remote
		// resource-language
		if remoteTime.Before(localTime) {
			feedbackMessage = "Local file is newer than remote, skipping"
		}
		return remoteTime.Before(localTime), feedbackMessage, nil
	}
	return false, feedbackMessage, nil
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

	return actedOnStringsPerc < minimum_percFloat
}

func getActedOnStringsPercentage(
	actedOnStrings float32,
	totalStrings float32) float32 {

	actedOnStringsPerc := (actedOnStrings * 100) / totalStrings
	return actedOnStringsPerc
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

func setFileTypeExtensions(fileType string, translationFile string) string {
	if fileType == "xliff" {
		translationFile = fmt.Sprintf("%s.xlf", translationFile)
	} else if fileType == "json" {
		translationFile = fmt.Sprintf("%s.json", translationFile)
	}
	return translationFile
}
