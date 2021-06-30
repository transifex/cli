package txapi

import (
	"fmt"

	"github.com/transifex/cli/pkg/jsonapi"
)

type ProjectAttributes struct {
	Archived        bool     `json:"archived"`
	Created         string   `json:"datetime_created"`
	Modified        string   `json:"datetime_modified"`
	Description     string   `json:"description"`
	HomepageURL     string   `json:"homepage_url"`
	InstructionsURL string   `json:"instructions_url"`
	License         string   `json:"license"`
	LongDescription string   `json:"long_description"`
	Name            string   `json:"name"`
	Private         bool     `json:"private"`
	RepositoryURL   string   `json:"repository_url"`
	Slug            string   `json:"slug"`
	Tags            []string `json:"tags"`
	TMFillup        bool     `json:"translation_memory_fillup"`
	Type            string   `json:"type"`
}

func GetProjects(
	api *jsonapi.Connection, organization *jsonapi.Resource,
) ([]*jsonapi.Resource, error) {
	query := jsonapi.Query{Filters: map[string]string{
		"organization": organization.Id,
	}}.Encode()
	projects, err := api.List("projects", query)
	if err != nil {
		return nil, err
	}

	var result []*jsonapi.Resource

	for {
		for i := range projects.Data {
			var projectAttributes ProjectAttributes
			var project = projects.Data[i]
			err := project.MapAttributes(&projectAttributes)
			if err != nil {
				return nil, err
			}
			project.SetRelated("organization", organization)
			result = append(result, &project)
		}
		if projects.Next == "" {
			break
		} else {
			projects, err = projects.GetNext()
			if err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func GetProject(
	api *jsonapi.Connection,
	organization *jsonapi.Resource,
	projectSlug string,
) (*jsonapi.Resource, error) {
	query := jsonapi.Query{Filters: map[string]string{
		"organization": organization.Id,
		"slug":         projectSlug,
	}}.Encode()
	projects, err := api.List("projects", query)
	if err != nil {
		return nil, err
	}

	if len(projects.Data) == 0 {
		return nil, nil
	} else if len(projects.Data) > 1 {
		return nil, fmt.Errorf(
			"somehow found more than 1 projects with slug %s", projectSlug,
		)
	}

	project := &projects.Data[0]
	project.SetRelated("organization", organization)

	return project, nil
}

func GetProjectLanguages(
	project *jsonapi.Resource,
) (map[string]*jsonapi.Resource, error) {
	languagesRelationship, err := project.Fetch("languages")
	if err != nil {
		return nil, err
	}

	result := make(map[string]*jsonapi.Resource)
	page := languagesRelationship.DataPlural
	for {
		for i := range page.Data {
			language := page.Data[i]
			var languageAttributes LanguageAttributes
			err := language.MapAttributes(&languageAttributes)
			if err != nil {
				return nil, err
			}
			result[languageAttributes.Code] = &language
		}
		if page.Next == "" {
			break
		} else {
			page, err = page.GetNext()
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}
