package api_explorer

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/urfave/cli/v2"
)

const UPLOAD_RESOURCE_STRINGS_ASYNC_STRING = `{
  "Upload options (remember to remove the leading '//')": "",

	"//replace_edited_strings": false,
	"//callback_url": "The url that will be called when the processing is completed"
}`

func cliCmdUploadResourceStringsAsyncUpload(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	resourceId, err := getResourceId(api, "")
	if err != nil {
		return err
	}
	attributes, err := create(
		UPLOAD_RESOURCE_STRINGS_ASYNC_STRING,
		c.String("editor"),
		[]string{"replace_edited_strings", "callback_url"},
	)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(c.String("input"))
	if err != nil {
		return err
	}
	if utf8.Valid(body) {
		attributes["content"] = string(body)
		attributes["content_encoding"] = "text"
	} else {
		attributes["content"] = base64.StdEncoding.EncodeToString(body)
		attributes["content_encoding"] = "base64"
	}
	upload := jsonapi.Resource{
		API:        api,
		Type:       "resource_strings_async_uploads",
		Attributes: attributes,
	}
	upload.SetRelated("resource", &jsonapi.Resource{Type: "resources", Id: resourceId})
	err = upload.Save(nil)
	if err != nil {
		return err
	}
	var uploadAttributes txapi.ResourceStringAsyncUploadAttributes
	for {
		err = upload.MapAttributes(&uploadAttributes)
		if err != nil {
			return err
		}
		if uploadAttributes.Status == "failed" {
			var errorsMessages []string
			for _, err := range upload.Attributes["errors"].([]map[string]string) {
				errorsMessages = append(errorsMessages, err["detail"])
			}
			return fmt.Errorf("upload failed: %s", strings.Join(errorsMessages, ", "))
		} else if uploadAttributes.Status == "succeeded" {
			break
		}
		time.Sleep(time.Duration(c.Int("interval")) * time.Second)
		err = upload.Reload()
		if err != nil {
			return err
		}
	}
	fmt.Printf(
		"Upload succeeded; created: %d, deleted: %d, skipped: %d, updated: %d "+
			"strings\n",
		uploadAttributes.Details.StringsCreated,
		uploadAttributes.Details.StringsDeleted,
		uploadAttributes.Details.StringsSkipped,
		uploadAttributes.Details.StringsUpdated,
	)
	return nil
}
