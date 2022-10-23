package api_explorer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

const CREATE_PROJECT_WEBHOOK_STRING = `{
  "Required fields": "",

  "active": true,
  "callback_url": "The callback URL of the webhook",
  "event_type": "translation_completed/translation_updated_completed/review_completed/proofread_completed/fillup_completed",

  "Optional fields (remember to remove the leading '//' from the keys)": "",

  "//secret_key": ""
}`

func selectProjectWebhookId(c *cli.Context, api *jsonapi.Connection) (string, error) {
	organizationId, err := getOrganizationId(c, api)
	if err != nil {
		return "", err
	}
	query := jsonapi.Query{Filters: map[string]string{"organization": organizationId}}
	projectId, err := getProjectId(c, api, organizationId, true)
	if err != nil {
		return "", err
	}
	if projectId != "" {
		query.Filters["project"] = projectId
	}
	body, err := api.ListBody("project_webhooks", query.Encode())
	if err != nil {
		return "", err
	}
	projectWebhookId, err := fuzzy(api, body, "Select webhook", nil, false)
	if err != nil {
		return "", err
	}
	return projectWebhookId, nil
}

func cliCmdGetProjectWebhooks(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	organizationId, err := getOrganizationId(c, api)
	if err != nil {
		return err
	}
	query := jsonapi.Query{
		Filters: map[string]string{"organization": organizationId},
	}
	projectId, err := getProjectId(c, api, organizationId, true)
	if err != nil {
		return err
	}
	if projectId != "" {
		query.Filters["project"] = projectId
	}
	body, err := api.ListBody("project_webhooks", query.Encode())
	if err != nil {
		return err
	}
	err = handlePagination(body)
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdGetProjectWebhook(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectWebhookId, err := selectProjectWebhookId(c, api)
	if err != nil {
		return err
	}
	body, err := api.GetBody("project_webhooks", projectWebhookId)
	if err != nil {
		return err
	}
	err = page(c.String("pager"), body)
	if err != nil {
		return err
	}
	return nil
}

func cliCmdCreateProjectWebhook(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		_, err = api.Request("POST", "/project_webhooks", body, "")
		if err != nil {
			return err
		}
		return nil
	}
	projectId, err := getProjectId(c, api, "", false)
	if err != nil {
		return err
	}
	attributes, err := create(
		CREATE_PROJECT_WEBHOOK_STRING,
		c.String("editor"),
		[]string{"active", "callback_url", "event_type", "secret_key"},
	)
	if err != nil {
		return err
	}
	projectWebhook := jsonapi.Resource{
		API:        api,
		Type:       "project_webhooks",
		Attributes: attributes,
	}
	projectWebhook.SetRelated("project", &jsonapi.Resource{
		Type: "projects",
		Id:   projectId,
	})
	err = projectWebhook.Save(nil)
	if err != nil {
		return err
	}
	fmt.Printf("Created project webhook: %s\n", projectWebhook.Id)
	return nil
}

func cliCmdDeleteProjectWebhook(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectWebhookId, err := selectProjectWebhookId(c, api)
	if err != nil {
		return err
	}
	fmt.Printf(
		"About to delete project webhook: %s, are you sure (y/N)? ",
		projectWebhookId,
	)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(strings.ToLower(answer)) == "y" {
		projectWebhook := jsonapi.Resource{
			API:  api,
			Type: "project_webhooks",
			Id:   projectWebhookId,
		}
		err = projectWebhook.Delete()
		if err != nil {
			return err
		}
		fmt.Printf("Deleted project webhook: %s\n", projectWebhookId)
	}
	return nil
}

func cliCmdEditProjectWebhook(c *cli.Context) error {
	api, err := getApi(c)
	if err != nil {
		return err
	}
	projectWebhookId, err := selectProjectWebhookId(c, api)
	if err != nil {
		return err
	}
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		_, err = api.Request(
			"PATCH",
			fmt.Sprintf("/project_webhooks/%s", projectWebhookId),
			body,
			"",
		)
		if err != nil {
			return err
		}
		return nil
	}
	projectWebhook, err := api.Get("project_webhooks", projectWebhookId)
	if err != nil {
		return err
	}
	err = edit(
		c.String("editor"),
		&projectWebhook,
		[]string{"active", "callback_url", "event_type", "secret_key"},
	)
	if err != nil {
		return err
	}
	return nil
}
