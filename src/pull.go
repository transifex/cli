package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

func pullCommand(c *cli.Context) error {
	// Get config
	config := c.App.Metadata["Config"].(*Config)
	projectDir := c.App.Metadata["ProjectDir"].(string)
	fileMappings := c.App.Metadata["FileMappings"].(map[string]FileMapping)

	// Get flags
	allFlag := c.Bool("all")
	forceFlag := c.Bool("force")
	skipExisting := c.Bool("disable-overwrite")
	useGitTimestamps := c.Bool("use-git-timestamps")
	ignoreErrors := c.Bool("skip")
	xliff := c.Bool("xliff")
	maxDownlads := c.Int("parallel")

	// Init client
	client := NewClient(config.Token, config.Hostname)

	// Context will be used to cancel downloads.
	// When the --skip flag is not passed when a download fails no files are written
	// to disk. When a download fails it calls the cancel() func which aborts other downloads.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// The write file lock synchronizes the downloads. When a download finishes successfully
	// it will mark it as finished `writeFileLock.Done()`.
	var writeFileLock sync.WaitGroup
	writeFileLockCh := make(chan struct{})

	// Controls how many downloads will be performed at the same time.
	guardMaxDownloads := make(chan struct{}, maxDownlads)

	for _, fileMapping := range fileMappings {

		projectID := "o:" + fileMapping.OrganizationSlug + ":p:" + fileMapping.ProjectSlug
		resourceID := fileMapping.ID

		resourceLanguageStats, err := client.getProjectLanguageStats(projectID, &resourceID, nil)
		if err != nil {
			return err
		}
		for _, resourceLanguageStat := range *resourceLanguageStats {
			// <-ctx.Done() means a download has failed (`cancel()` was called)
			// default continue execution
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			languageID := resourceLanguageStat.Relationships.Language.Data.ID
			// It blows my mind that we don't have access to the language code anywhere in the response.
			txLanguageCode := strings.Split(languageID, ":")[1]
			if txLanguageCode == fileMapping.SourceLang {
				continue
			}

			localLanguageCode := getLocalLanguageCode(txLanguageCode, &fileMapping)

			_, isExistingFile := fileMapping.LanguageMappings[localLanguageCode]

			// If no --all flag was provided we update only files that exist locally
			if !allFlag && !isExistingFile {
				continue
			}
			// if the --disable-overwrite flag is passed skip existing files
			if skipExisting && isExistingFile {
				continue
			}

			languagePath := getLanguagePath(localLanguageCode, projectDir, &fileMapping)

			// Check timestamp only for existing files
			if isExistingFile {
				var localLastUpdate time.Time
				var txLastUpdate time.Time
				if useGitTimestamps {
					localLastUpdate, err = lastCommitDate(projectDir, languagePath)
					if err != nil {
						return err
					}
				} else {
					fileInfo, err := os.Stat(languagePath)
					if err != nil {
						return err
					}
					localLastUpdate = fileInfo.ModTime()
				}

				datelayout := "2006-01-02T15:04:05Z"
				txLastUpdate, err = time.Parse(datelayout, resourceLanguageStat.Attributes.LastUpdate)
				if err != nil {
					return err
				}
				hasUpdate := localLastUpdate.Before(txLastUpdate)

				// If no --force flag was provided we update only local files
				// that before the last update in transifex
				if !hasUpdate && !forceFlag {
					continue
				}
			}

			// Will block if `maxDownloads` is reached (channel is filled)
			guardMaxDownloads <- struct{}{}

			// Add 1 to wait groups
			wg.Add(1)
			writeFileLock.Add(1)
			go func() {
				// Substracts 1 from wait group when goroutine exits
				defer wg.Done()

				content, err := downloadFile(client, resourceID, languageID, xliff, languagePath)
				if err != nil {
					fmt.Printf(err.Error())
					// Cancel signal. Cancels other downloads
					cancel()
					return
				}

				fmt.Printf("Download completed for resource `%s` and language `%s`\n", resourceID, languageID)
				// Mark download complete
				writeFileLock.Done()
				// Read one entry
				// Will make another download possible if `maxDownloads` is reached
				<-guardMaxDownloads

				// If the `skip` flag was passed all files or no files will be downloaded
				if !ignoreErrors {
					// Wait for all downloads to be finished or check for cancel signal.
					// If the channel `writeFileLockCh` closes the file will be written to disk.
					// If the `cancel()` func is called it will exit.
					select {
					case <-ctx.Done():
						return
					case <-writeFileLockCh:
					}
				}

				if xliff {
					ext := path.Ext(languagePath)
					languagePath = languagePath[0:len(languagePath)-len(ext)] + ".xlf"

				}

				fmt.Printf("Writing resource `%s` and language `%s` to path `%s`\n", resourceID, languageID, languagePath)

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
	wg.Wait()
	return nil
}

func downloadFile(client *Client, resourceID string, languageID string, xliff bool, path string) (*[]byte, error) {
	resourceTranslationDownload, err := client.createResourceTranslationsDownload(
		resourceID, languageID, "default", xliff,
	)
	fmt.Printf("Downloading resource `%s` and language `%s`\n", resourceID, languageID)
	if err != nil {
		return nil, err
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancelFunc()

	result, err := client.pollResourceTranslationsDownload(
		ctx, *resourceTranslationDownload.ID, 1*time.Second,
	)

	if result.Attributes.Content != nil {
		return result.Attributes.Content, nil
	} else {
		return nil, fmt.Errorf("Error compiling file for resource `%s` and language `%s`", resourceID, languageID)
	}
}
