package jsonapi

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Links struct {
	Self    string `json:"self,omitempty"`
	Related string `json:"related,omitempty"`
}

// Used to parse JSON

type PayloadSingular struct {
	Data     PayloadResource   `json:"data"`
	Included []PayloadResource `json:"included,omitempty"`
}

type PayloadPluralRead struct {
	Data     []PayloadResource `json:"data"`
	Links    PaginationLinks   `json:"links,omitempty"`
	Included []PayloadResource `json:"included,omitempty"`
}

type PayloadPluralWrite struct {
	Data []PayloadResource `json:"data"`
}

type PaginationLinks struct {
	Previous string `json:"previous,omitempty"`
	Next     string `json:"next,omitempty"`
}

type PayloadResource struct {
	Type          string                 `json:"type"`
	Id            string                 `json:"id,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
}

type PayloadRelationshipSingularRead struct {
	Data  ResourceIdentifier `json:"data,omitempty"`
	Links Links              `json:"links,omitempty"`
}

type PayloadRelationshipSingularWrite struct {
	Data ResourceIdentifier `json:"data,omitempty"`
}

type PayloadRelationshipPlural struct {
	Data  []ResourceIdentifier
	Links Links
}

type ResourceIdentifier struct {
	Type string `json:"type,omitempty"`
	Id   string `json:"id,omitempty"`
}

func payloadToResource(
	in PayloadResource,
	included *map[string]Resource,
	API *Connection,
) (Resource, error) {
	out := Resource{
		API:           API,
		Type:          in.Type,
		Id:            in.Id,
		Attributes:    in.Attributes,
		Relationships: make(map[string]*Relationship),
	}

	for key, value := range in.Relationships {
		// Here we try to map the JSON relationship to both a singular and
		// plural struct to see which matches
		body, err := json.Marshal(value)
		if err != nil {
			return out, err
		}
		var relationshipSingular PayloadRelationshipSingularRead
		_ = json.Unmarshal(body, &relationshipSingular)
		var relationshipPlural PayloadRelationshipPlural
		_ = json.Unmarshal(body, &relationshipPlural)

		if relationshipSingular.Data != (ResourceIdentifier{}) {
			Type := relationshipSingular.Data.Type
			Id := relationshipSingular.Data.Id

			includedKey := fmt.Sprintf("%s:%s", Type, Id)
			var item Resource
			var exists bool
			if included == nil {
				exists = false
			} else {
				item, exists = (*included)[includedKey]
			}
			var Data Relationship
			if exists {
				Data = Relationship{
					Type:         SINGULAR,
					Fetched:      true,
					DataSingular: &item,
					Links:        relationshipSingular.Links,
				}
			} else {
				Data = Relationship{
					Type:    SINGULAR,
					Fetched: false,
					DataSingular: &Resource{
						API:  API,
						Type: Type,
						Id:   Id,
					},
					Links: relationshipSingular.Links,
				}
			}
			out.Relationships[key] = &Data

		} else if relationshipPlural.Links != (Links{}) {
			out.Relationships[key] = &Relationship{
				Type:    PLURAL,
				Fetched: false,
				DataPlural: Collection{
					API:  API,
					Data: make([]Resource, 0, len(relationshipPlural.Data)),
				},
				Links: relationshipPlural.Links,
			}
			for _, item := range relationshipPlural.Data {
				_ = append(out.Relationships[key].DataPlural.Data, Resource{
					API:  API,
					Type: item.Type,
					Id:   item.Id,
				})
			}
		} else {
			out.Relationships[key] = &Relationship{
				Type: NULL,
			}
		}
	}

	return out, nil
}

func jsonEqual(leftBytes, rightBytes []byte) (bool, error) {
	var left interface{}
	err := json.Unmarshal(leftBytes, &left)
	if err != nil {
		return false, err
	}

	var right interface{}
	err = json.Unmarshal(rightBytes, &right)
	if err != nil {
		return false, err
	}

	return reflect.DeepEqual(left, right), nil
}

// Convert response payload's included array and to a <type>.<id>: resource map
func makeIncludedMap(
	includedPayload []PayloadResource,
	API *Connection,
) (map[string]Resource, error) {
	result := make(map[string]Resource)
	for _, payloadResource := range includedPayload {
		includedResource, err := payloadToResource(payloadResource, nil, API)
		if err != nil {
			return result, err
		}
		key := fmt.Sprintf("%s:%s", includedResource.Type, includedResource.Id)
		result[key] = includedResource
	}
	return result, nil
}
