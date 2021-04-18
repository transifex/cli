package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

func pullCommand(ctx *cli.Context) error {
	config := ctx.App.Metadata["Config"].(*Config)
	projectDir := ctx.App.Metadata["ProjectDir"].(string)
	fileMappings := ctx.App.Metadata["FileMappings"].(map[string]FileMapping)

	allFlag := ctx.Bool("all")
	forceFlag := ctx.Bool("force")
	skipExisting := ctx.Bool("disable-overwrite")
	useGitTimestamps := ctx.Bool("use-git-timestamps")

	client := NewClient(config.Token, config.Hostname)

	var wg sync.WaitGroup
	for _, fileMapping := range fileMappings {

		projectID := "o:" + fileMapping.OrganizationSlug + ":p:" + fileMapping.ProjectSlug
		resourceID := fileMapping.ID

		resourceLanguageStats, err := client.getProjectLanguageStats(projectID, &resourceID, nil)
		if err != nil {
			return err
		}
		for _, resourceLanguageStat := range *resourceLanguageStats {
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

			// Async download
			wg.Add(1)
			go func() {
				downloadFile(client, resourceID, languageID, languagePath)
				wg.Done()
			}()
		}
	}
	wg.Wait()
	return nil
}

func downloadFile(client *Client, resourceID string, languageID string, path string) {
	resourceTranslationDownload, err := client.createResourceTranslationsDownload(
		resourceID, languageID, "default", false,
	)
	fmt.Printf("Downloading resource `%s` and language `%s`\n", resourceID, languageID)
	if err != nil {
		fmt.Printf("Failed to download resource `%s` and language `%s`\n", resourceID, languageID)
		fmt.Print(err)
		return
	}
	ctx, cancelFunc := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancelFunc()

	result, err := client.pollResourceTranslationsDownload(
		ctx, *resourceTranslationDownload.ID, 1*time.Second,
	)

	if result.Attributes.Content != nil {
		err := ioutil.WriteFile(path, *result.Attributes.Content, 0644)
		if err != nil {
		}
	} else {
		fmt.Printf("Failed to download resource with ID `%s`\n", resourceID)
	}
}
