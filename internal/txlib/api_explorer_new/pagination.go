package api_explorer_new

import (
	"encoding/json"

	"github.com/transifex/cli/pkg/jsonapi"
)

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

func joinPages(api *jsonapi.Connection, bodyBytes []byte) ([]byte, error) {
	var resultJson struct {
		Data []interface{} `json:"data"`
	}
	var bodyJson struct {
		Data  []interface{} `json:"data"`
		Links struct {
			Next string `json:"next"`
		} `json:"links"`
	}
	err := json.Unmarshal(bodyBytes, &bodyJson)
	if err != nil {
		return nil, err
	}
	resultJson.Data = append(resultJson.Data, bodyJson.Data...)
	for bodyJson.Links.Next != "" {
		bodyBytes, err = api.ListBodyFromPath(bodyJson.Links.Next)
		if err != nil {
			return nil, err
		}
		bodyJson.Links.Next = ""
		err = json.Unmarshal(bodyBytes, &bodyJson)
		if err != nil {
			return nil, err
		}
		resultJson.Data = append(resultJson.Data, bodyJson.Data...)
	}
	resultBody, err := json.Marshal(resultJson)
	if err != nil {
		return nil, err
	}
	return resultBody, nil
}

func getIsEmpty(bodyBytes []byte) (bool, error) {
	var bodyJson struct {
		Data []interface{} `json:"data"`
	}
	err := json.Unmarshal(bodyBytes, &bodyJson)
	if err != nil {
		return false, err
	}
	if len(bodyJson.Data) == 0 {
		return true, nil
	}
	return false, nil
}

func getIfOnlyOne(bodyBytes []byte) (string, error) {
	var bodyJson struct {
		Data []struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	err := json.Unmarshal(bodyBytes, &bodyJson)
	if err != nil {
		return "", err
	}
	if len(bodyJson.Data) == 1 {
		return bodyJson.Data[0].Id, nil
	}
	return "", nil
}
