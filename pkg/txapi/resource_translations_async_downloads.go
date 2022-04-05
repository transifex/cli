package txapi

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
)

type CreateDownloadArguments struct {
	OrganizationSlug string
	ProjectSlug      string
	ResourceSlug     string
	Resource         *jsonapi.Resource
	Language         *jsonapi.Resource
	FileType         string
	Mode             string
	ContentEncoding  string
}

type ResourceTranslationsAsyncDownloadAttributes struct {
	ContentEncoding string `json:"content_encoding"`
	FileType        string `json:"file_type"`
	Pseudo          bool   `json:"pseudo"`
	Mode            string `json:"mode"`
}

func CreateTranslationsAsyncDownload(api *jsonapi.Connection,
	arguments CreateDownloadArguments) (*jsonapi.Resource, error) {
	download := jsonapi.Resource{
		API:  api,
		Type: "resource_translations_async_downloads",
		Attributes: map[string]interface{}{
			"content_encoding": arguments.ContentEncoding,
			"file_type":        arguments.FileType,
			"mode":             arguments.Mode,
			"pseudo":           false,
		},
	}
	download.SetRelated("resource", arguments.Resource)
	download.SetRelated("language", arguments.Language)

	err := download.Save(nil)
	if err != nil {
		return nil, err
	}

	return &download, nil
}

func PollTranslationDownload(languageMappings map[string]string,
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

			languageRelationship, err := download.Fetch("language")
			if err != nil {
				return err
			}
			txLanguageCode := languageRelationship.DataSingular.Attributes["code"].(string)

			localLanguageCode, languageDirectory := GetLanguageDirectory(
				languageMappings,
				txLanguageCode,
				cfgResource)

			err = os.MkdirAll(languageDirectory, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}

			translationFile := strings.Replace(
				cfgResource.FileFilter, "<lang>", localLanguageCode, -1,
			)

			if cfgResource.Overrides[localLanguageCode] != "" {
				translationFile = cfgResource.Overrides[localLanguageCode]
			}

			if fileType == "xliff" {
				translationFile = fmt.Sprintf("%s.xlf", translationFile)
			} else if fileType == "json" {
				translationFile = fmt.Sprintf("%s.json", translationFile)
			}

			err = ioutil.WriteFile(translationFile, bodyBytes, 0644)
			if err != nil {
				return err
			}
			resp.Body.Close()
			break
		}

		if download.Attributes["status"] == "failed" {
			return fmt.Errorf(
				"download of translation '%s' failed",
				download.Relationships["resource"].DataSingular.Id,
			)
		} else if download.Attributes["status"] == "succeeded" {
			break
		}
		time.Sleep(duration)
	}
	return nil
}

func GetLanguageDirectory(
	languageMappings map[string]string,
	txLanguageCode string,
	cfgResource *config.Resource,
) (string, string) {
	localLanguageCode := getLocalLanguageCode(
		languageMappings, txLanguageCode, cfgResource,
	)

	path := strings.ReplaceAll(
		cfgResource.FileFilter, "<lang>", localLanguageCode,
	)
	languageDirectory := filepath.Dir(path)
	return localLanguageCode, languageDirectory
}

func getLocalLanguageCode(
	languageMappings map[string]string, lang string, resource *config.Resource,
) string {

	if val, ok := resource.LanguageMappings[lang]; ok {
		return val
	}

	if val, ok := languageMappings[lang]; ok {
		return val
	}

	return lang
}
