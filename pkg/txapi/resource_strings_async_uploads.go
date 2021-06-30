package txapi

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/transifex/cli/pkg/jsonapi"
)

type ResourceStringAsyncUploadAttributes struct {
	DateCreated  string `json:"date_created"`
	DateModified string `json:"date_modified"`
	Status       string `json:"status"`
	Details      struct {
		StringsCreated int `json:"strings_created"`
		StringsDeleted int `json:"strings_deleted"`
		StringsSkipped int `json:"strings_skipped"`
		StringsUpdated int `json:"strings_updated"`
	} `json:"details"`
	Errors []struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

func (err *ResourceStringAsyncUploadAttributes) Error() string {
	// Lets make this into an error type
	parts := make([]string, 0, len(err.Errors))
	for _, item := range err.Errors {
		parts = append(parts,
			fmt.Sprintf("%s: %s", item.Code, item.Detail))
	}
	return strings.Join(parts, ", ")
}

func UploadSource(
	api *jsonapi.Connection, resource *jsonapi.Resource, file io.Reader,
) (*jsonapi.Resource, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	upload := jsonapi.Resource{
		API:  api,
		Type: "resource_strings_async_uploads",
		// Setting attributes directly here because POST and GET attributes are
		// different
		Attributes: map[string]interface{}{
			"content": data,
		},
	}
	upload.SetRelated("resource", resource)
	err = upload.SaveAsMultipart(nil)
	if err != nil {
		return nil, err
	}

	return &upload, nil
}

func PollSourceUpload(upload *jsonapi.Resource, duration time.Duration) error {
	for {
		err := upload.Reload()
		if err != nil {
			return err
		}

		var uploadAttributes ResourceStringAsyncUploadAttributes
		err = upload.MapAttributes(&uploadAttributes)
		if err != nil {
			return err
		}

		if uploadAttributes.Status == "failed" {
			// Wrap the "error"
			return fmt.Errorf("upload of resource '%s' failed - %w",
				upload.Relationships["resource"].DataSingular.Id,
				&uploadAttributes)
		} else if uploadAttributes.Status == "succeeded" {
			break
		}
		time.Sleep(duration)
	}
	return nil
}
