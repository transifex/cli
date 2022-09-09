package txapi

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/transifex/cli/pkg/jsonapi"
)

func CreateResourceStringsAsyncDownload(
	api *jsonapi.Connection,
	resource *jsonapi.Resource,
	contentEncoding string,
	fileType string,
) (*jsonapi.Resource, error) {
	download := &jsonapi.Resource{
		API:  api,
		Type: "resource_strings_async_downloads",
		Attributes: map[string]interface{}{
			"content_encoding": contentEncoding,
			"file_type":        fileType,
			"pseudo":           false,
		},
	}
	download.SetRelated("resource", resource)
	err := download.Save(nil)
	return download, err
}

func PollResourceStringsDownload(
	download *jsonapi.Resource,
	duration time.Duration,
	filePath string,
) error {
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
			if resp.StatusCode != 200 {
				return errors.New("file download error")
			}
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			dir := filepath.Dir(filePath)
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return err
			}

			err = ioutil.WriteFile(filePath, bodyBytes, 0644)
			if err != nil {
				return err
			}
			resp.Body.Close()
			return nil
		} else if download.Attributes["status"] == "failed" {
			return fmt.Errorf(
				"download of translation '%s' failed",
				download.Relationships["resource"].DataSingular.Id,
			)

		} else if download.Attributes["status"] == "succeeded" {
			return nil
		}
		time.Sleep(duration)
	}
}
