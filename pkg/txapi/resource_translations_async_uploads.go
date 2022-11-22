package txapi

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/transifex/cli/pkg/jsonapi"
)

func UploadTranslation(
	api *jsonapi.Connection,
	resource,
	language *jsonapi.Resource,
	file io.Reader,
	xliff bool,
) (*jsonapi.Resource, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var fileType string
	if xliff {
		fileType = "xliff"
	} else {
		fileType = "default"
	}

	upload := jsonapi.Resource{
		API:  api,
		Type: "resource_translations_async_uploads",
		// Setting attributes directly here because POST and GET attributes are
		// different
		Attributes: map[string]interface{}{
			"content":   data,
			"file_type": fileType,
		},
	}
	upload.SetRelated("resource", resource)
	upload.SetRelated("language", language)
	err = upload.SaveAsMultipart(nil)
	if err != nil {
		return nil, err
	}

	return &upload, nil
}

type ResourceTranslationsAsyncUploadAttributes struct {
	DateCreated  string `json:"date_created"`
	DateModified string `json:"date_modified"`
	Status       string `json:"status"`
	Details      struct {
		TranslationsCreated int `json:"translations_created"`
		TranslationsUpdated int `json:"translations_updated"`
	} `json:"details"`
	Errors []struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

func (err *ResourceTranslationsAsyncUploadAttributes) Error() string {
	// Lets make this into an error type
	parts := make([]string, 0, len(err.Errors))
	for _, item := range err.Errors {
		parts = append(parts,
			fmt.Sprintf("%s: %s", item.Code, item.Detail))
	}
	return strings.Join(parts, ", ")
}

func PollTranslationUpload(
	upload *jsonapi.Resource, duration time.Duration,
) error {
	for {
		time.Sleep(duration)
		err := upload.Reload()
		if err != nil {
			return err
		}
		var uploadAttributes ResourceTranslationsAsyncUploadAttributes
		err = upload.MapAttributes(&uploadAttributes)
		if err != nil {
			return err
		}
		if uploadAttributes.Status == "failed" {
			// Wrap the "error"
			return fmt.Errorf(
				"upload of resource '%s', language '%s' failed - %w",
				upload.Relationships["resource"].DataSingular.Id,
				upload.Relationships["language"].DataSingular.Id,
				&uploadAttributes)
		} else if uploadAttributes.Status == "succeeded" {
			break
		}
	}
	return nil
}
