package txapi

import (
	"errors"

	"github.com/transifex/cli/pkg/jsonapi"
)

type ResourceAttributes struct {
	AcceptTranslation bool     `json:"accept_translations"`
	Categories        []string `json:"categories"`
	DatetimeCreated   string   `json:"datetime_created"`
	DatetimeModified  string   `json:"datetime_modified"`
	I18nOptions       struct {
		AllowDuplicateStrings bool `json:"allow_duplicate_strings"`
	} `json:"i18n_options"`
	I18nVersion int    `json:"i18n_version"`
	Mp4Url      string `json:"mp4_url"`
	Name        string `json:"name"`
	OggUrl      string `json:"ogg_url"`
	Priority    string `json:"priority"`
	Slug        string `json:"slug"`
	StringCount int    `json:"string_count"`
	WebmUrl     string `json:"webm_url"`
	WordCount   int    `json:"word_count"`
	YoutubeUrl  string `json:"youtube_url"`
}

func GetResources(
	api *jsonapi.Connection, project *jsonapi.Resource,
) ([]*jsonapi.Resource, error) {
	query := jsonapi.Query{Filters: map[string]string{
		"project": project.Id,
	}}.Encode()

	resources, err := api.List("resources", query)

	if err != nil {
		return nil, err
	}

	var result []*jsonapi.Resource

	for {
		for i := range resources.Data {
			var resourceAttributes ResourceAttributes
			var resource = resources.Data[i]
			err := resource.MapAttributes(&resourceAttributes)
			if err != nil {
				return nil, err
			}
			resource.SetRelated("project", project)
			result = append(result, &resource)
		}
		if resources.Next == "" {
			break
		} else {
			resources, err = resources.GetNext()
			if err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func GetResource(
	api *jsonapi.Connection, project *jsonapi.Resource, resourceSlug string,
) (*jsonapi.Resource, error) {
	query := jsonapi.Query{Filters: map[string]string{
		"project": project.Id,
	}}.Encode()
	resources, err := api.List("resources", query)
	if err != nil {
		return nil, err
	}

	for { // pagination
		for i := range resources.Data {
			resource := resources.Data[i]
			var resourceAttributes ResourceAttributes
			err := resource.MapAttributes(&resourceAttributes)
			if err != nil {
				return nil, err
			}
			if resourceAttributes.Slug == resourceSlug {
				resource.SetRelated("project", project)
				return &resource, nil
			}
		}
		if resources.Next != "" {
			resources, err = resources.GetNext()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, nil
		}
	}
}

func CreateResource(
	api *jsonapi.Connection, project_id string,
	resourceName, resourceSlug, Type string,
) (*jsonapi.Resource, error) {
	resource := &jsonapi.Resource{
		API:  api,
		Type: "resources",
	}
	err := resource.UnmapAttributes(ResourceAttributes{
		Name: resourceName,
		Slug: resourceSlug,
	})
	if err != nil {
		return nil, err
	}
	resource.SetRelated("project", &jsonapi.Resource{Type: "projects", Id: project_id})
	resource.SetRelated("i18n_format",
		&jsonapi.Resource{Type: "i18n_formats", Id: Type})

	err = resource.Save([]string{"name", "slug", "project", "i18n_format"})
	resource.Relationships["project"].Fetched = false
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func DeleteResource(
	api *jsonapi.Connection, resource *jsonapi.Resource,
) error {
	err := resource.Delete()

	if err != nil {
		return err
	}

	return nil
}

func GetResourceFromId(api *jsonapi.Connection, id string) (*jsonapi.Resource, error) {
	resource, err := api.Get("resources", id)
	if err != nil {
		var e *jsonapi.Error
		if errors.As(err, &e) {
			if e.StatusCode == 404 {
				return nil, nil
			}
		}
		return nil, err
	}
	return &resource, nil
}
