package txlib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
)

const (
	projectsUrlMergeCommand = "/projects?" +
		"filter%5Borganization%5D=o%3Aorgslug&filter%5Bslug%5D=projslug"
	resourcesUrlMergeCommand = "/resources?filter%5B" +
		"project%5D=o%3Aorgslug%3Ap%3Aprojslug"
)

func TestMergeSuccess(t *testing.T) {
	mockData := getMockedDataForResourceMerge()
	api := jsonapi.GetTestConnection(mockData)
	commandArgs := MergeCommandArguments{"projslug.resslug", "the_branch", "USE_HEAD", false, false, false}
	resourceId := getStandardConfigMerge().FindResource("projslug.resslug")
	err := mergeResource(&api, resourceId, commandArgs)
	assert.Nil(t, err)

}

func TestMergeInvalidPolicy(t *testing.T) {
	mockData := getMockedDataForResourceMerge()
	api := jsonapi.GetTestConnection(mockData)
	commandArgs := MergeCommandArguments{"projslug.resslug", "the_branch", "INVALID_POLICY", false, false, false}
	resourceId := getStandardConfigMerge().FindResource("projslug.resslug")
	err := mergeResource(&api, resourceId, commandArgs)
	assert.NotNil(t, err)

}

func getStandardConfigMerge() *config.Config {
	return &config.Config{
		Local: &config.LocalConfig{
			Resources: []config.Resource{
				{
					OrganizationSlug: "orgslug",
					ProjectSlug:      "projslug",
					ResourceSlug:     "resslug",
					Type:             "I18N_TYPE",
					SourceFile:       "aaa.json",
					FileFilter:       "aaa-<lang>.json",
				},
			},
		},
	}
}
func mergeGetOrganizationEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{
			{
				Response: jsonapi.MockResponse{
					Text: `{"data": [{"type": "organizations",
									"id": "o:orgslug",
									"attributes": {"slug": "orgslug"}}]}`,
				},
			},
		},
	}
}

func mergeGetProjectsEndpoint() *jsonapi.MockEndpoint {
	selfUrl := "/projects/o:orgslug:p:projslug/relationships/languages"
	relatedUrl := "/projects/o:orgslug:p:projslug/languages"
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{
			{
				Response: jsonapi.MockResponse{
					Text: fmt.Sprintf(`{"data": [{
						"type": "projects",
						"id": "o:orgslug:p:projslug",
						"attributes": {"slug": "projslug"},
						"relationships": {
							"languages": {
								"links": {
									"self": "%s",
									"related": "%s"
								}
							},
							"source_language": {
								"data": {"type": "languages", "id": "l:en"},
								"links": {"related": "/languages/l:en"}
							}
						}
					}]}`, selfUrl, relatedUrl),
				},
			},
			{
				Response: jsonapi.MockResponse{
					Text: fmt.Sprintf(`{"data": [{
						"type": "projects",
						"id": "o:orgslug:p:projslug",
						"attributes": {"slug": "projslug"},
						"relationships": {
							"languages": {
								"links": {
									"self": "%s",
									"related": "%s"
								}
							},
							"source_language": {
								"data": {"type": "languages", "id": "l:en"},
								"links": {"related": "/languages/l:en"}
							}
						}
					}]}`, selfUrl, relatedUrl),
				},
			},
		},
	}
}

func mergeGetResourceEndpoint() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(
		`{
  "data": [{
    "type": "resources",
    "id": "o:orgslug:p:projslug:r:resslug",
    "attributes": {
      "slug": "resslug"
    },
    "relationships": {
      "project": {
        "links": {
          "related": "/projects/o:orgslug:p:projslug"
        },
        "data": {
          "type": "projects",
          "id": "o:orgslug:p:projslug"
        }
      }
 
    }
  }]
}`,
	)
}
func mergeEndpoint() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(`{
  "data": {
    "type": "resource_async_merges",
    "id": "some_uuid",
    "attributes": {
      "status": "CREATED",
      "conflict_resolution": "USE_HEAD"
    },
    "relationships": {
      "base": {
        "data": {
          "type": "resources",
          "id": "o:orgslug:p:projslug:r:resslug"
        }
      },
      "head": {
        "data": {
          "type": "resources",
          "id": "o:orgslug:p:projslug:r:resslug"
        }
      }
    },
    "links": {
      "self": "/resource_async_merges/some_uuid"
    }
  }
}`)
}

func mergePollingEndpoint() *jsonapi.MockEndpoint {
	return &jsonapi.MockEndpoint{
		Requests: []jsonapi.MockRequest{
			{
				Response: jsonapi.MockResponse{
					Text: `{
						"data": {
							"type": "resource_async_merges",
							"id": "some_uuid",
							"attributes": {
								"status": "CREATED",
								"conflict_resolution": "USE_HEAD"
							},
							"relationships": {
								"base": {
									"data": {
										"type": "resources",
										"id": "o:organization_slug:p:project_slug:r:resource_slug"
									}
								},
								"head": {
									"data": {
										"type": "resources",
										"id": "o:organization_slug:p:project_slug:r:resource_slug"
									}
								}
							},
							"links": {
								"self": "/resource_strings_async_downloads/some_uuid"
							}
						}
						}`,
				},
			},
			{
				Response: jsonapi.MockResponse{
					Text: `{
						"data": {
							"type": "resource_async_merges",
							"id": "some_uuid",
							"attributes": {
								"status": "COMPLETED",
								"conflict_resolution": "USE_HEAD"								
							},
							"relationships": {
								"base": {
									"data": {
										"type": "resources",
										"id": "o:organization_slug:p:project_slug:r:resource_slug"
									}
								},
								"head": {
									"data": {
										"type": "resources",
										"id": "o:organization_slug:p:project_slug:r:resource_slug"
									}
								}
							},
							"links": {
								"self": "/resource_strings_async_downloads/some_uuid"
							}
						}
						}`,
				},
			},
		},
	}
}
func getMockedDataForResourceMerge() jsonapi.MockData {
	return jsonapi.MockData{
		"/organizations":                   mergeGetOrganizationEndpoint(),
		projectsUrlMergeCommand:            mergeGetProjectsEndpoint(),
		resourcesUrlMergeCommand:           mergeGetResourceEndpoint(),
		"/resource_async_merges":           mergeEndpoint(),
		"/resource_async_merges/some_uuid": mergePollingEndpoint(),
	}
}
