package txlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
)

const (
	branchResourceUrl = "/resources/o:orgslug:p:projslug:r:resslug"
)

func TestMergeSuccess(t *testing.T) {
	mockData := getMockedDataForResourceMerge()
	api := jsonapi.GetTestConnection(mockData)
	commandArgs := MergeCommandArguments{"projslug.resslug", "the_branch", "USE_HEAD", false, false, false}
	resource := getStandardConfigMerge().FindResource("projslug.resslug")
	err := mergeResource(&api, resource, commandArgs)
	assert.Nil(t, err)
}

func TestMergeInvalidPolicy(t *testing.T) {
	mockData := getMockedDataForResourceMerge()
	api := jsonapi.GetTestConnection(mockData)
	commandArgs := MergeCommandArguments{"projslug.resslug", "the_branch", "INVALID_POLICY", false, false, false}
	resource := getStandardConfigMerge().FindResource("projslug.resslug")
	err := mergeResource(&api, resource, commandArgs)
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

func mergeGetResourceEndpoint() *jsonapi.MockEndpoint {
	return jsonapi.GetMockTextResponse(
		`{
			"data": {
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
			}
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
		branchResourceUrl:                  mergeGetResourceEndpoint(),
		"/resource_async_merges":           mergeEndpoint(),
		"/resource_async_merges/some_uuid": mergePollingEndpoint(),
	}
}
