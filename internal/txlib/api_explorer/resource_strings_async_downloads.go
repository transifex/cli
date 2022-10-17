package api_explorer

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

const DOWNLOAD_RESOURCE_STRINGS_ASYNC_STRING = `{
  "Download options (remember to remove the leading '//')": "",

	"//content_encoding": "text/base64",
	"//file_type": "default/xliff",
	"//pseudo": false,
	"//pseudo_length_increase": 0,
	"//callback_url": "The url that will be called when the processing is completed"
}`

func cliCmdDownloadResourceStringsAsyncDownload(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	resourceId, err := getResourceId(api, "")
	if err != nil {
		return err
	}
	attributes, err := create(
		DOWNLOAD_RESOURCE_STRINGS_ASYNC_STRING,
		c.String("editor"),
		[]string{
			"content_encoding", "file_type", "pseudo", "pseudo_length_increase",
			"callback_url",
		},
	)
	if err != nil {
		return err
	}
	download := &jsonapi.Resource{
		API:        api,
		Type:       "resource_strings_async_downloads",
		Attributes: attributes,
	}
	download.SetRelated("resource", &jsonapi.Resource{Type: "resources", Id: resourceId})
	err = download.Save(nil)
	if err != nil {
		return err
	}
	for {
		if download.Redirect != "" {
			response, err := http.Get(download.Redirect)
			if err != nil {
				return err
			}
			if response.StatusCode != 200 {
				return errors.New("file download error")
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				return err
			}
			defer response.Body.Close()
			if c.String("output") != "" {
				os.WriteFile(c.String("output"), body, 0644)
			} else {
				os.Stdout.Write(body)
			}
			return nil
		} else if download.Attributes["status"] == "failed" {
			var errorsMessages []string
			for _, err := range download.Attributes["errors"].([]map[string]string) {
				errorsMessages = append(errorsMessages, err["detail"])
			}
			return fmt.Errorf("download failed: %s", strings.Join(errorsMessages, ", "))
		}
		time.Sleep(time.Duration(c.Int("interval")) * time.Second)
		err = download.Reload()
		if err != nil {
			return err
		}
	}
}
