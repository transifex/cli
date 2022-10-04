package txlib

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gosimple/slug"
	"github.com/mattn/go-isatty"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
)

func figureOutBranch(branch string) string {
	if branch == "-1" {
		return ""
	} else if branch == "" {
		return getGitBranch()
	} else {
		return branch
	}
}

func figureOutResources(
	resourceIds []string,
	cfg *config.Config,
) ([]*config.Resource, error) {
	var result []*config.Resource

	if len(resourceIds) != 0 {
		result = make([]*config.Resource, 0, len(resourceIds))
		for _, resourceId := range resourceIds {
			cfgResource := cfg.FindResource(resourceId)
			if cfgResource == nil {
				return nil, fmt.Errorf(
					"could not find resource '%s' in local configuration or your "+
						"resource slug is invalid",
					resourceId,
				)
			}

			result = append(result, cfgResource)
		}
	} else {
		for i := range cfg.Local.Resources {
			result = append(result, &cfg.Local.Resources[i])
		}
	}
	return result, nil
}

func applyBranchToResources(cfgResources []*config.Resource, branch string) {
	for i := range cfgResources {
		cfgResource := cfgResources[i]
		if branch != "" {
			cfgResource.ResourceSlug = fmt.Sprintf(
				"%s--%s",
				slug.Make(branch),
				cfgResource.ResourceSlug,
			)
		}
	}
}

func stringSliceContains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}

func makeLocalToRemoteLanguageMappings(
	cfg config.Config, cfgResource config.Resource,
) map[string]string {
	// In the configuration, the language mappings are "remote code -> local
	// code" (eg 'pt_BT: pt-br'). Looking into the filesystem, we get the local
	// language codes; so if we need to find the remote codes, we need to
	// reverse the maps

	result := make(map[string]string)
	for key, value := range cfg.Local.LanguageMappings {
		result[value] = key
	}
	for key, value := range cfgResource.LanguageMappings {
		// Resource language mappings overwrite "global" language mappings
		result[value] = key
	}
	return result
}

func makeRemoteToLocalLanguageMappings(
	localToRemoteLanguageMappings map[string]string,
) map[string]string {
	result := make(map[string]string)
	for key, value := range localToRemoteLanguageMappings {
		result[value] = key
	}
	return result
}

/*
Run 'do'. If the error returned by 'do' is a jsonapi.ThrottleError, sleep the number of
seconds indicated by the error and try again. Meanwhile, inform the user of
what's going on using 'send'.
*/
func handleThrottling(do func() error, initialMsg string, send func(string)) error {
	for {
		if len(initialMsg) > 0 {
			send(initialMsg)
		}
		err := do()
		if err == nil {
			return nil
		} else {
			var e *jsonapi.ThrottleError
			if errors.As(err, &e) {
				retryAfter := e.RetryAfter
				if isatty.IsTerminal(os.Stdout.Fd()) {
					for retryAfter > 0 {
						send(fmt.Sprintf(
							"Throttled, will retry after %d seconds",
							retryAfter,
						))
						time.Sleep(time.Second)
						retryAfter -= 1
					}
				} else {
					send(fmt.Sprintf(
						"Throttled, will retry after %d seconds",
						retryAfter,
					))
					time.Sleep(time.Duration(retryAfter) * time.Second)
				}
			} else {
				return err
			}
		}
	}
}

func checkFileFilter(fileFilter string) error {
	if fileFilter == "" {
		return errors.New("file filter is empty")
	} else {
		return validateFileFilter(fileFilter)
	}
}
