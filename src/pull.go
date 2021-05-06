package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

// PullFlags represent the flags that can be passed in the pull command.
type PullFlags struct {
	AllFlag           bool     // whenther to download non existing files as well
	ForceFlag         bool     // whether to skip comparing timestamps
	SkipExisting      bool     // whether to skip downloading existing files
	UseGitTimestamps  bool     // whether to use git intead of local file timestamps
	IgnoreErrors      bool     // whether to write successful downloads even if one has failed
	Xliff             bool     // whether to download the xliff file format
	MaxDownlads       int      // how many downloads to do concurrently
	ResourceRegex     string   // the final regex that will be checked against a resource slug
	MinimumPercentage *float64 //
}

// PullCommand will pull files from the transifex API.
// First it will retrieve each resource's `resource_language_stats`
// The for each language it will download the file when the following criteria are met:
// 1. The file exist localy
// 2. The current file timestamp is older than the last resource language statistic update
// 3. The language is not the source language
// The files will be downloaded serially and will be written to disk only if they are all successful
// The FileMapping configuration will be used to derive the filepath locations for each resource and language
func PullCommand(c *cli.Context) error {
	// Get config
	config := c.App.Metadata["Config"].(*Config)
	fileMappings := c.App.Metadata["FileMappings"].(map[string]FileMapping)

	// If a resource regex is passed validate and prepare it
	resourceRegex := c.String("resource")
	if resourceRegex != "" {
		if isValid, _ := regexp.MatchString(`^[-\w\*]+$`, resourceRegex); !isValid {
			return fmt.Errorf("Not valid resource regex `%s`", resourceRegex)
		}
		resourceRegex = strings.Replace(resourceRegex, "*", `[-\w]+`, -1) + "$"
	}

	var minimumPercentage *float64
	if c.IsSet("minimum-perc") {
		tmp := c.Float64("minimum-perc")
		minimumPercentage = &tmp
	}
	// Get flags
	pullFlags := PullFlags{
		AllFlag:           c.Bool("all"),
		ForceFlag:         c.Bool("force"),
		SkipExisting:      c.Bool("disable-overwrite"),
		UseGitTimestamps:  c.Bool("use-git-timestamps"),
		IgnoreErrors:      c.Bool("skip"),
		Xliff:             c.Bool("xliff"),
		MaxDownlads:       c.Int("parallel"),
		ResourceRegex:     resourceRegex,
		MinimumPercentage: minimumPercentage,
	}

	client := NewClient(config.Token, config.Hostname)

	// Context will be used to cancel downloads.
	// When the --skip flag is not passed when a download fails no files are written
	// to disk. When a download fails it calls the cancelFunc() func which aborts other downloads.
	cancelContext, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Controls when the command will exit.
	var globalLock sync.WaitGroup

	// The write file lock synchronizes the downloads. When a download finishes successfully
	// it will mark it as finished `writeFileLock.Done()`.
	var writeFileLock sync.WaitGroup
	writeFileLockCh := make(chan struct{})

	// Controls how many downloads will be performed at the same time.
	guardMaxDownloads := make(chan struct{}, pullFlags.MaxDownlads)
	defer close(guardMaxDownloads)

	for _, fileMapping := range fileMappings {

		projectID := "o:" + fileMapping.OrganizationSlug + ":p:" + fileMapping.ProjectSlug
		resourceID := fileMapping.ID

		resourceLanguageStats, err := client.getProjectLanguageStats(projectID, &resourceID, nil)
		if err != nil {
			return err
		}
		for _, resourceLanguageStat := range *resourceLanguageStats {
			// <-cancelContext.Done() means a download has failed (`cancelFunc()` was called)
			// by default continue execution
			select {
			case <-cancelContext.Done():
				return nil
			default:
			}

			shouldSkip, err := skipDownload(&pullFlags, &fileMapping, &resourceLanguageStat)
			if err != nil {
				return err
			}
			if shouldSkip {
				continue
			}

			// Will block if `maxDownloads` is reached (channel is filled)
			guardMaxDownloads <- struct{}{}

			// Add 1 to wait groups
			globalLock.Add(1)
			writeFileLock.Add(1)
			languageID := resourceLanguageStat.Relationships.Language.Data.ID
			go func() {
				// Substracts 1 from wait group when goroutine exits
				defer globalLock.Done()

				languagePath := getLanguagePathFromID(languageID, &fileMapping)

				content, err := downloadFile(client, resourceID, languageID, &pullFlags)

				if err != nil {
					fmt.Printf(err.Error())
					// If the `--skip` flag is not passed cancel all downloads
					if !pullFlags.IgnoreErrors {
						cancelFunc()
					}
				}

				fmt.Printf("Download completed for resource `%s` and language `%s`\n", resourceID, languageID)

				// Mark download complete
				writeFileLock.Done()
				// Read one entry
				// Will make another download possible if `maxDownloads` is reached
				<-guardMaxDownloads

				// If we are not ignoring errors we have to wait
				// for all downloads to be finished successfully or receive the cancel signal.
				if !pullFlags.IgnoreErrors {
					select {
					// If the `cancelFunc()` func is called it will exit.
					case <-cancelContext.Done():
						return
					// If the channel `writeFileLockCh` closes the file will can be written to disk.
					case <-writeFileLockCh:
					}
				}

				fmt.Printf("Writing resource `%s` and language `%s` to path `%s`\n", resourceID, languageID, languagePath)

				if pullFlags.Xliff {
					ext := path.Ext(languagePath)
					languagePath = languagePath[0:len(languagePath)-len(ext)] + ".xlf"
				}

				if _, err = os.Stat(languagePath); os.IsNotExist(err) {
					os.MkdirAll(filepath.Dir(languagePath), os.ModePerm)
					if _, err = os.Create(languagePath); err != nil {
						return
					}
				}

				err = ioutil.WriteFile(languagePath, *content, 0644)
				if err != nil {
					fmt.Println(err.Error())
				}
			}()
		}
	}
	// When all files finished downloading close the `writeFileLockCh` channel
	// This will allow writing to disk when the `skip` flag is NOT passed
	go func() {
		writeFileLock.Wait()
		close(writeFileLockCh)
	}()

	// Wait for all downloads to finish
	globalLock.Wait()
	return nil
}

func downloadFile(client *Client, resourceID string, languageID string, pullFlags *PullFlags) (*[]byte, error) {
	resourceTranslationDownload, err := client.createResourceTranslationsDownload(
		resourceID, languageID, "default", pullFlags.Xliff,
	)
	fmt.Printf("Downloading resource `%s` and language `%s`\n", resourceID, languageID)
	if err != nil {
		return nil, err
	}

	cancelContext, cancelFunc := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancelFunc()

	result, err := client.pollResourceTranslationsDownload(
		cancelContext, *resourceTranslationDownload.ID, 1*time.Second,
	)

	if result.Attributes.Content != nil {
		return result.Attributes.Content, nil
	} else {
		return nil, fmt.Errorf("Error compiling file for resource `%s` and language `%s`", resourceID, languageID)
	}
}

func skipDownload(pullArgs *PullFlags, fileMapping *FileMapping, resourceLanguageStat *ResourceLanguageStats) (bool, error) {

	if pullArgs.ResourceRegex != "" {
		isMatch, err := regexp.MatchString(pullArgs.ResourceRegex, fileMapping.ResourceSlug)
		if !isMatch || err != nil {
			return true, err
		}
	}

	// It blows my mind that we don't have access to the language code anywhere in the response.
	txLanguageCode := strings.Split(resourceLanguageStat.Relationships.Language.Data.ID, ":")[1]
	if txLanguageCode == fileMapping.SourceLang {
		return true, nil
	}

	localLanguageCode := getLocalLanguageCode(txLanguageCode, fileMapping)

	_, isExistingFile := fileMapping.LanguageMappings[localLanguageCode]

	// If no --all flag was provided we update only files that exist locally
	if !pullArgs.AllFlag && !isExistingFile {
		return true, nil
	}
	// if the --disable-overwrite flag is passed skip existing files
	if pullArgs.SkipExisting && isExistingFile {
		return true, nil
	}

	if pullArgs.MinimumPercentage != nil {
		part := resourceLanguageStat.Attributes.TranslatedStrings
		total := resourceLanguageStat.Attributes.TotalStrings
		currentPercentage := (float64(part) * float64(100)) / float64(total)
		if currentPercentage < *pullArgs.MinimumPercentage {
			return true, nil
		}
	}

	// Check timestamp only for existing files
	if isExistingFile {
		languagePath := getLanguagePathFromID(
			resourceLanguageStat.Relationships.Language.Data.ID,
			fileMapping,
		)
		var localLastUpdate time.Time
		var err error
		if pullArgs.UseGitTimestamps {
			localLastUpdate, err = lastCommitDate(fileMapping.ProjectDir, languagePath)
			if err != nil {
				return true, err
			}
		} else {
			fileInfo, err := os.Stat(languagePath)
			if err != nil {
				return true, err
			}
			localLastUpdate = fileInfo.ModTime()
		}

		datelayout := "2006-01-02T15:04:05Z"
		txLastUpdate, err := time.Parse(datelayout, resourceLanguageStat.Attributes.LastUpdate)
		if err != nil {
			return true, err
		}
		hasUpdate := localLastUpdate.Before(txLastUpdate)

		// If no --force flag was provided we update only local files
		// that were last updated before the last update in transifex
		if !hasUpdate && !pullArgs.ForceFlag {
			return true, nil
		}
	}
	return false, nil
}
