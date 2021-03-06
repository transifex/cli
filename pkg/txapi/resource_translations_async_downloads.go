package txapi

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

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
	for {
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
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	languageDirectory := filepath.Dir(filePath)
	err = os.MkdirAll(languageDirectory, os.ModePerm)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filePath, bodyBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}
