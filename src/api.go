package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Organization struct {
	Attributes struct {
		LogoURL string `json:"logo_url"`
		Name    string `json:"name"`
		Private bool   `json:"private"`
		Slug    string `json:"slug"`
	} `json:"attributes"`
	ID    string `json:"id"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
	Type string `json:"type"`
}

type Links struct {
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Self     string `json:"self"`
}

type OrganizationList struct {
	Data  []Organization `json:"data"`
	Links Links          `json:"links"`
}

func getOrganizations(token string, baseURL string) []Organization {
	req, _ := http.NewRequest("GET", baseURL+"/organizations", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	var output OrganizationList
	decoder := json.NewDecoder(resp.Body)
	decoder.DisallowUnknownFields()
	decoder.Decode(&output)
	org, _ := JSONMarshal(output.Data)
	fmt.Printf(string(org))
	links, _ := JSONMarshal(output.Links)
	fmt.Printf(string(links))
	return output.Data
}

type Project struct {
	Attributes struct {
		Archived                bool     `json:"archived"`
		DatetimeCreated         string   `json:"datetime_created"`
		DatetimeModified        string   `json:"datetime_modified"`
		Description             string   `json:"description"`
		HomepageURL             string   `json:"homepage_url"`
		InstructionsURL         string   `json:"instructions_url"`
		License                 string   `json:"license"`
		LongDescription         string   `json:"long_description"`
		Name                    string   `json:"name"`
		Private                 bool     `json:"private"`
		RepositoryURL           string   `json:"repository_url"`
		Slug                    string   `json:"slug"`
		Tags                    []string `json:"tags"`
		TranslationMemoryFillup bool     `json:"translation_memory_fillup"`
		Type                    string   `json:"type"`
	} `json:"attributes"`
	ID    string `json:"id"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
	Relationships struct {
		Languages struct {
			Links struct {
				Related string `json:"related"`
			} `json:"links"`
		} `json:"languages"`
		Organization struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
			Links struct {
				Related string `json:"related"`
			} `json:"links"`
		} `json:"organization"`
		SourceLanguage struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
			Links struct {
				Related string `json:"related"`
			} `json:"links"`
		} `json:"source_language"`
	} `json:"relationships"`
	Type string `json:"type"`
}

type ProjectList struct {
	Data  []Project `json:"data"`
	Links Links     `json:"links"`
}

func getProjects(organizationID string, token string, baseURL string) []Project {
	req, _ := http.NewRequest("GET", baseURL+"/projects?filter[organization]="+organizationID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	var output ProjectList
	decoder := json.NewDecoder(resp.Body)
	decoder.DisallowUnknownFields()
	decoder.Decode(&output)
	org, _ := JSONMarshal(output.Data)
	fmt.Printf(string(org))
	links, _ := JSONMarshal(output.Links)
	fmt.Printf(string(links))
	return output.Data
}

type Resource struct {
	Attributes struct {
		AcceptTranslations bool     `json:"accept_translations"`
		Categories         []string `json:"categories"`
		DatetimeCreated    string   `json:"datetime_created"`
		DatetimeModified   string   `json:"datetime_modified"`
		I18NVersion        int      `json:"i18n_version"`
		Mp4URL             string   `json:"mp4_url"`
		Name               string   `json:"name"`
		OggURL             string   `json:"ogg_url"`
		Priority           string   `json:"priority"`
		Slug               string   `json:"slug"`
		StringCount        int      `json:"string_count"`
		WebmURL            string   `json:"webm_url"`
		WordCount          int      `json:"word_count"`
		YoutubeURL         string   `json:"youtube_url"`
	} `json:"attributes"`
	ID    string `json:"id"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
	Relationships struct {
		I18NFormat struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
		} `json:"i18n_format"`
		Project struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
			Links struct {
				Related string `json:"related"`
			} `json:"links"`
		} `json:"project"`
	} `json:"relationships"`
	Type string `json:"type"`
}

type ResourceList struct {
	Data  []Resource `json:"data"`
	Links Links      `json:"links"`
}

func getResources(projectID string, token string, baseURL string) []Resource {
	req, _ := http.NewRequest("GET", baseURL+"/resources?filter[project]="+projectID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	var output ResourceList
	decoder := json.NewDecoder(resp.Body)
	decoder.DisallowUnknownFields()
	decoder.Decode(&output)
	org, _ := JSONMarshal(output.Data)
	fmt.Printf(string(org))
	links, _ := JSONMarshal(output.Links)
	fmt.Printf(string(links))
	return output.Data
}
