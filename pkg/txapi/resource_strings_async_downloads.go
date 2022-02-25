package txapi

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
)

type CreateResourceStringDownloadArguments struct {
	OrganizationSlug string
	ProjectSlug      string
	ResourceSlug     string
	Resource         *jsonapi.Resource
	FileType         string
	ContentEncoding  string
}

type ResourceStringsAsyncDownloadAttributes struct {
	ContentEncoding string `json:"content_encoding"`
	FileType        string `json:"file_type"`
	Pseudo          bool   `json:"pseudo"`
}

func CreateResourceStringsAsyncDownload(
	api *jsonapi.Connection, arguments CreateResourceStringDownloadArguments,
) (*jsonapi.Resource, error) {
	download := jsonapi.Resource{
		API:  api,
		Type: "resource_strings_async_downloads",
		Attributes: map[string]interface{}{
			"content_encoding": arguments.ContentEncoding,
			"file_type":        arguments.FileType,
			"pseudo":           false,
		},
	}
	download.SetRelated("resource", arguments.Resource)
	err := download.Save(nil)
	if err != nil {
		return nil, err
	}
	return &download, nil
}

func PollResourceStringsDownload(
	download *jsonapi.Resource,
	duration time.Duration,
	cfgResource *config.Resource,
	fileType string) error {
	for {
		err := download.Reload()
		if err != nil {
			return err
		}

		if download.Redirect != "" {
			resp, err := http.Get(download.Redirect)
			if err != nil {
				return err
			}
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			dir, _ := filepath.Split(cfgResource.SourceFile)

			if dir != "" {
				if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
					err := fmt.Errorf("directory '%s' does not exist", dir)
					return err
				}
			}

			sourceFile := cfgResource.SourceFile

			if fileType == "xliff" {
				sourceFile = fmt.Sprintf("%s.xlf", sourceFile)
			}
			err = ioutil.WriteFile(sourceFile, bodyBytes, 0644)
			if err != nil {
				return err
			}
			resp.Body.Close()
			break
		}

		if download.Attributes["status"] == "failed" {
			err = fmt.Errorf(
				"download of translation '%s' failed",
				download.Relationships["resource"].DataSingular.Id,
			)
			return err

		} else if download.Attributes["status"] == "succeeded" {
			break
		}
		time.Sleep(duration)
	}
	return nil
}
