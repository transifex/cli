package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
)

type Connection struct {
	Host string
	Auth string
}

func (c Connection) JsonApiRequest(method string, path string, queryParams map[string]string) (*http.Response, error) {
	// certPEM := []byte("xxxx") // load from file https://raw.githubusercontent.com/transifex/transifex-client/master/txclib/cacert.pem
	// certPool := x509.NewCertPool()
	// if !certPool.AppendCertsFromPEM(certPEM) {
	// 	log.Fatal("AppendCertsFromPEM failed")
	// }

	// https://golang.org/src/crypto/x509/ check roots
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
		// RootCAs:            certPool,
	}
	tr.Proxy = http.ProxyFromEnvironment
	client := &http.Client{Transport: tr}

	url := fmt.Sprintf("%s/%s", c.Host, path)
	req, _ := http.NewRequest("GET", url, nil)

	q := req.URL.Query()
	for k, v := range queryParams {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Auth))
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, _ := client.Do(req)
	return resp, nil
}

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

func (c Connection) getOrganizations() []Organization {
	resp, _ := c.JsonApiRequest("GET", "organizations", make(map[string]string))
	var output OrganizationList
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	decoder.DisallowUnknownFields()
	decoder.Decode(&output)
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

func (c Connection) getProjects(organizationID string) []Project {
	resp, _ := c.JsonApiRequest("GET", "projects", map[string]string{"filter[organization]": organizationID})
	var output ProjectList
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	decoder.DisallowUnknownFields()
	decoder.Decode(&output)
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

func (c Connection) getResources(projectID string) []Resource {
	resp, _ := c.JsonApiRequest("GET", "resources", map[string]string{"filter[project]": projectID})
	var output ResourceList
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	decoder.DisallowUnknownFields()
	decoder.Decode(&output)
	return output.Data
}
