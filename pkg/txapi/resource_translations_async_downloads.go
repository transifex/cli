package txapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/transifex/cli/pkg/jsonapi"
)

func CreateTranslationsAsyncDownload(
	api *jsonapi.Connection,
	resource *jsonapi.Resource,
	languageCode string,
	contentEncoding string,
	fileType string,
	mode string,
) (*jsonapi.Resource, error) {
	download := &jsonapi.Resource{
		API:  api,
		Type: "resource_translations_async_downloads",
		Attributes: map[string]interface{}{
			"content_encoding": contentEncoding,
			"file_type":        fileType,
			"mode":             mode,
			"pseudo":           false,
		},
	}
	download.SetRelated("resource", resource)
	download.SetRelated(
		"language",
		&jsonapi.Resource{Type: "languages", Id: fmt.Sprintf("l:%s", languageCode)},
	)
	err := download.Save(nil)
	return download, err
}

func PollTranslationDownload(download *jsonapi.Resource, filePath string) error {
	backoff := getBackoff(nil)
	for {
		time.Sleep(time.Duration(backoff()) * time.Second)
		err := download.Reload()
		if err != nil {
			return err
		}
		if download.Redirect != "" {
			break
		}
	}
	resp, err := http.Get(download.Redirect)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("file download error")
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	languageDirectory := filepath.Dir(filePath)
	err = os.MkdirAll(languageDirectory, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.WriteFile(filePath, bodyBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}
