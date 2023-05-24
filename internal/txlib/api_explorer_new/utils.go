package api_explorer_new

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/transifex/cli/internal/txlib"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/urfave/cli/v2"
)

func getApi(c *cli.Context) (*jsonapi.Connection, error) {
	cfg, err := config.LoadFromPaths(
		c.String("root-config"),
		c.String("config"),
	)
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %s", err)
	}
	hostname, token, err := txlib.GetHostAndToken(
		&cfg,
		c.String("hostname"),
		c.String("token"),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting API token: %s", err)
	}

	client, err := txlib.GetClient(c.String("cacert"))
	if err != nil {
		return nil, fmt.Errorf("error getting HTTP client configuration: %s", err)
	}

	return &jsonapi.Connection{
		Host:    hostname,
		Token:   token,
		Client:  client,
		Headers: map[string]string{"Integration": "txclient"},
	}, nil
}

func save(key, value string) error {
	if _, err := os.Stat(".tx"); os.IsNotExist(err) {
		err := os.Mkdir(".tx", 0755)
		if err != nil {
			return err
		}
	}
	var body []byte
	if _, err := os.Stat(".tx/api_explorer_session.json"); err == nil {
		body, err = os.ReadFile(".tx/api_explorer_session.json")
		if err != nil {
			return err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		body = []byte("{}")

	} else {
		return err
	}
	var data map[string]string
	err := json.Unmarshal(body, &data)
	if err != nil {
		return err
	}
	data[key] = value
	body, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(".tx/api_explorer_session.json", body, 0644)
	if err != nil {
		return err
	}
	return nil
}

func clear(key string) error {
	_, err := os.Stat(".tx/api_explorer_session.json")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	body, err := os.ReadFile(".tx/api_explorer_session.json")
	if err != nil {
		return err
	}
	var data map[string]string
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}
	delete(data, key)
	body, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(".tx/api_explorer_session.json", body, 0644)
	if err != nil {
		return err
	}
	return nil
}

func handlePagination(body []byte) error {
	var payload struct {
		Links struct {
			Next     string
			Previous string
		}
	}
	err := json.Unmarshal(body, &payload)
	if err != nil {
		return err
	}
	if payload.Links.Next != "" {
		err = save("next", payload.Links.Next)
		if err != nil {
			return err
		}
	} else {
		clear("next")
	}
	if payload.Links.Previous != "" {
		err = save("previous", payload.Links.Previous)
		if err != nil {
			return err
		}
	} else {
		clear("previous")
	}
	return nil
}

func page(pager string, body []byte) error {
	var unmarshalled map[string]interface{}
	err := json.Unmarshal(body, &unmarshalled)
	if err != nil {
		return err
	}
	output, err := json.MarshalIndent(unmarshalled, "", "  ")
	if err != nil {
		return err
	}
	cmd := exec.Command(pager)
	cmd.Stdin = bytes.NewBuffer(output)
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
