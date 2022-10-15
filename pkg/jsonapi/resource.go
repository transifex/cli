package jsonapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
)

type Resource struct {
	API           *Connection
	Type          string
	Id            string
	Attributes    map[string]interface{}
	Relationships map[string]*Relationship
	Redirect      string
	Links         Links
}

const (
	NULL     = iota
	SINGULAR = iota
	PLURAL   = iota
)

type Relationship struct {
	Type         int
	Fetched      bool
	DataSingular *Resource
	DataPlural   Collection
	Links        Links
}

/*
Fetch data for a relationship and return a reference to it.

If the data was previously fetched (the 'Fetched' field is true), 'Save'
returns immediately.
*/
func (r *Resource) Fetch(key string) (*Relationship, error) {
	relationship, exists := r.Relationships[key]
	if !exists {
		return nil, fmt.Errorf("relationship %s does not exist", key)
	}
	if relationship.Type == NULL {
		return nil, fmt.Errorf("cannot fetch null relationship")
	}
	if relationship.Fetched {
		return relationship, nil
	}

	// Now lets actually fetch
	if relationship.Type == SINGULAR {
		var url string
		if relationship.Links.Related != "" {
			url = relationship.Links.Related
		} else {
			url = fmt.Sprintf("/%s/%s",
				relationship.DataSingular.Type,
				relationship.DataSingular.Id)
		}
		Data, err := r.API.getFromPath(url)
		if err != nil {
			return relationship, err
		}
		relationship.Fetched = true
		relationship.DataSingular = &Data
	} else if relationship.Type == PLURAL {
		Url := relationship.Links.Related
		if Url == "" {
			return relationship, errors.New(
				"plural relationship doesn't have a 'related' link",
			)
		}
		Data, err := r.API.listFromPath(Url)
		if err != nil {
			return relationship, err
		}
		relationship.Fetched = true
		relationship.DataPlural = Data
	}

	return relationship, nil
}

/*
Save the resource on the server. If there is an Id present, send a PATCH
request, otherwise send a POST request. The Attributes and relationships that
will be sent are the ones in the 'fields' argument. If the 'fields' argument is
nil, everything will be saved.
*/
func (r *Resource) Save(fields []string) error {
	if len(fields) == 0 {
		keys := make([]string, 0,
			len(r.Attributes)+len(r.Relationships))
		for key := range r.Attributes {
			keys = append(keys, key)
		}
		for key := range r.Relationships {
			keys = append(keys, key)
		}
		return r.Save(keys)
	}
	var method, url string
	if r.Id != "" {
		method = "PATCH"
		url = fmt.Sprintf("/%s/%s", r.Type, r.Id)
	} else {
		method = "POST"
		url = fmt.Sprintf("/%s", r.Type)
	}

	payload := PayloadSingular{}
	payload.Data.Type = r.Type
	payload.Data.Id = r.Id

	for _, field := range fields {
		attribute, attributeExists := r.Attributes[field]
		relationship, relationshipsExists := r.Relationships[field]
		if attributeExists {
			if payload.Data.Attributes == nil {
				payload.Data.Attributes = make(map[string]interface{})
			}
			payload.Data.Attributes[field] = attribute
		} else if relationshipsExists && relationship.Type == SINGULAR {
			if payload.Data.Relationships == nil {
				payload.Data.Relationships = make(map[string]interface{})
			}
			payload.Data.Relationships[field] = PayloadRelationshipSingularWrite{
				Data: ResourceIdentifier{
					Type: relationship.DataSingular.Type,
					Id:   relationship.DataSingular.Id,
				},
			}
		} else {
			return fmt.Errorf("field %s is invalid", field)
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	body, err = r.API.request(method, url, body, "")
	if err != nil {
		return err
	}

	err = r.overwrite(body)
	if err != nil {
		return err
	}

	return nil
}

func (r *Resource) SaveAsMultipart(fields []string) error {
	if len(fields) == 0 {
		keys := make([]string, 0,
			len(r.Attributes)+len(r.Relationships))
		for key := range r.Attributes {
			keys = append(keys, key)
		}
		for key := range r.Relationships {
			keys = append(keys, key)
		}
		return r.SaveAsMultipart(keys)
	}

	var method, url string
	if r.Id != "" {
		method = "PATCH"
		url = fmt.Sprintf("/%s/%s", r.Type, r.Id)
	} else {
		method = "POST"
		url = fmt.Sprintf("/%s", r.Type)
	}
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	defer writer.Close()

	for _, field := range fields {
		attribute, attributeExists := r.Attributes[field]
		relationship, relationshipsExists := r.Relationships[field]
		if attributeExists {
			switch data := attribute.(type) {
			case string:
				err := writer.WriteField(field, data)
				if err != nil {
					return err
				}
			case []byte:
				w, err := writer.CreateFormFile(field,
					fmt.Sprintf("%s.txt", field))
				if err != nil {
					return nil
				}
				_, err = w.Write(data)
				if err != nil {
					return nil
				}
			default:
				return fmt.Errorf("field %s is not of type string or bytes",
					field)
			}
		} else if relationshipsExists {
			if relationship.Type != SINGULAR {
				return fmt.Errorf("field %s is not a singular relationship",
					field)
			}
			err := writer.WriteField(field, relationship.DataSingular.Id)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("field %s is invalid", field)
		}
	}
	err := writer.Close()
	if err != nil {
		return err
	}

	body, err := r.API.request(
		method, url, payload.Bytes(),
		fmt.Sprintf("multipart/form-data;boundary=%s", writer.Boundary()),
	)

	if err != nil {
		return err
	}

	err = r.overwrite(body)
	if err != nil {
		return err
	}

	return nil
}

/*
Delete a resource from the server. Response is empty on success
*/
func (r *Resource) Delete() error {
	url := r.Links.Self
	if url == "" {
		// Make an extra effort
		url = fmt.Sprintf("/%s/%s", r.Type, r.Id)
	}
	_, err := r.API.request("DELETE", url, nil, "")

	if err != nil {
		return err
	}
	r.Id = ""
	return nil
}

func (r *Resource) Reload() error {
	url := r.Links.Self
	if url == "" {
		// Make an extra effort
		url = fmt.Sprintf("/%s/%s", r.Type, r.Id)
	}
	body, err := r.API.request("GET", url, nil, "")
	if err != nil {
		var e *RedirectError
		if errors.As(err, &e) {
			r.Redirect = e.Location
			return nil
		} else {
			return err
		}
	}

	err = r.overwrite(body)
	if err != nil {
		return err
	}

	return nil
}

func (r *Resource) Add(field string, items []*Resource) error {
	return r.modifyPluralRelationship("POST", field, items)
}

func (r *Resource) Remove(field string, items []*Resource) error {
	return r.modifyPluralRelationship("DELETE", field, items)
}

func (r *Resource) Reset(field string, items []*Resource) error {
	return r.modifyPluralRelationship("PATCH", field, items)
}

func (r *Resource) modifyPluralRelationship(
	method, field string, items []*Resource,
) error {
	relationship, exists := r.Relationships[field]
	if !exists {
		return fmt.Errorf("relationship '%s' does not exist", field)
	}
	if relationship.Type != PLURAL {
		return fmt.Errorf("cannot modify the non-plural relationship '%s'",
			field)
	}
	url := relationship.Links.Self
	if url == "" {
		// Make an extra effort
		url = fmt.Sprintf("/%s/%s/relationships/%s", r.Type, r.Id, field)
	}
	payload := PayloadPluralWrite{Data: make([]PayloadResource, 0)}
	for i := range items {
		item := items[i]
		payload.Data = append(payload.Data, PayloadResource{
			Type: item.Type,
			Id:   item.Id,
		})
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.API.request(method, url, payloadBytes, "")
	if err != nil {
		return err
	}

	// Make sure relationship needs re-fetching
	r.Relationships[field].DataPlural = Collection{}
	r.Relationships[field].Fetched = false

	return nil
}

func (r *Resource) overwrite(body []byte) error {
	var response PayloadSingular
	err := json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	included, err := makeIncludedMap(response.Included, r.API)
	if err != nil {
		return err
	}

	result, err := payloadToResource(response.Data, &included, r.API)
	if err != nil {
		return err
	}

	r.Type = result.Type
	r.Id = result.Id
	r.Attributes = result.Attributes

	// Delete current relationships that are absent in response
	for key := range r.Relationships {
		_, exists := result.Relationships[key]
		if !exists {
			delete(r.Relationships, key)
		}
	}

	// Overwrite current relationships with the ones in response
	for key := range result.Relationships {
		newRelationship := result.Relationships[key]
		oldRelationship, exists := r.Relationships[key]

		shouldOverwrite := !exists ||

			// Relationships have different plurality, or...
			oldRelationship.Type != newRelationship.Type ||

			// Comparison of singular relationships
			(oldRelationship.Type == SINGULAR &&
				newRelationship.Type == SINGULAR &&
				(
				// Related object was changed, or
				oldRelationship.DataSingular.Type !=
					newRelationship.DataSingular.Type ||
					oldRelationship.DataSingular.Id !=
						newRelationship.DataSingular.Id ||

					// Related object was not changed, but response has
					// included information, or...
					newRelationship.Fetched)) ||

			// Comparison of plural relationships
			(oldRelationship.Type == PLURAL &&
				newRelationship.Type == PLURAL &&

				// Response has included information
				newRelationship.Fetched)

		if shouldOverwrite {
			r.Relationships[key] = newRelationship
		}

		// Regardless of whether things were overwritten or not, the new
		// relationship's links should apply
		r.Relationships[key].Links = newRelationship.Links
	}

	return nil
}

/*
MapAttributes Map a resource's attributes to a struct. Usage:

    type ProjectAttributes struct {
        Name string
        ...
    }

    func main() {
        api := jsonapi.Connection{...}
        project, _ := api.Get("projects", "XXX")
        var projectAttributes ProjectAttributes
        project.MapAttributes(&projectAttributes)

        fmt.Println(projectAttributes.Name)
    }

*/
func (r *Resource) MapAttributes(result interface{}) error {
	data, err := json.Marshal(r.Attributes)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return err
	}
	return nil
}

/*
UnmapAttributes Unmap a struct to a resource's attributes (possibly before
calling 'Save').

Usage:

    type ProjectAttributes struct {
        Name string
        ...
    }

    func main() {
        api := jsonapi.Connection{...}
        project, _ := api.Get("projects", "XXX")
        var projectAttributes ProjectAttributes
        project.MapAttributes(&projectAttributes)

        projectAttributes.Name = "New name"
        project.UnmapAttributes(projectAttributes)
        project.Save([]string{"name"})
    }
*/
func (r *Resource) UnmapAttributes(source interface{}) error {
	data, err := json.Marshal(source)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &r.Attributes)
	if err != nil {
		return err
	}
	return nil
}

/*
SetRelated Set a relationship to a resource. Can be used either before saving
or after getting a resource from the API and wanting to "pre-fetch" a parent
resource that is at hand.

For saving:

    parent := ...
    child := ...
    child.SetRelated("parent", parent)
    child.Save("parent")

For "pre-fetching":

    parent := ...
    query := Query{Filters: map[string][string]{"parent": parent.Id}}.Encode()
    page, _ := api.List("children", query)
    child := page.Data[0]
    child.SetRelated("parent", parent)
*/
func (r *Resource) SetRelated(field string, related *Resource) {
	var links Links
	existing, exists := r.Relationships[field]
	if exists {
		links = existing.Links
	}

	if r.Relationships == nil {
		r.Relationships = make(map[string]*Relationship)
	}

	r.Relationships[field] = &Relationship{
		Type:         SINGULAR,
		DataSingular: related,
		Links:        links,
		Fetched:      true,
	}
}
