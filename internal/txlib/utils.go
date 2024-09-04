package txlib

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/mattn/go-isatty"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"golang.org/x/term"
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
		existingResourceIds := make(map[string]*config.Resource)
		for i := range cfg.Local.Resources {
			resource := &cfg.Local.Resources[i]
			resourceId := fmt.Sprintf("%s.%s", resource.ProjectSlug, resource.ResourceSlug)
			existingResourceIds[resourceId] = resource
		}

		for _, resourceId := range resourceIds {
			pattern, err := regexp.Compile(
				"^" + strings.ReplaceAll(regexp.QuoteMeta(resourceId), "\\*", ".*") + "$",
			)
			if err != nil {
				return nil, err
			}
			atLeastOne := false
			for existingResourceId := range existingResourceIds {
				if pattern.MatchString(existingResourceId) {
					result = append(result, existingResourceIds[existingResourceId])
					atLeastOne = true
				}
			}
			if !atLeastOne {
				return nil, fmt.Errorf(
					"could not find resource '%s' in local configuration or your "+
						"resource slug is invalid",
					resourceId,
				)
			}
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
			cfgResource.ResourceSlug = getBranchResourceSlug(cfgResource, branch)
		}
	}
}

func getBaseResourceSlug(cfgResource *config.Resource, branch string, base string) string {
	if branch != "" {
		branchPrefix := fmt.Sprintf(
			"%s--",
			slug.Make(branch),
		)
		mainResourceSlug := cfgResource.ResourceSlug[len(branchPrefix):]
		baseBranch := base
		if base == "-1" {
			baseBranch = ""
		}
		if baseBranch == "" {
			return mainResourceSlug
		} else {
			return fmt.Sprintf(
				"%s--%s",
				slug.Make(baseBranch),
				mainResourceSlug,
			)
		}
	} else {
		return cfgResource.ResourceSlug
	}
}

func getBranchResourceSlug(cfgResource *config.Resource, branch string) string {
	if branch != "" {
		return fmt.Sprintf(
			"%s--%s",
			slug.Make(branch),
			cfgResource.ResourceSlug,
		)
	} else {
		return cfgResource.ResourceSlug
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

func makeRemoteToLocalLanguageMappings(
	cfg config.Config, cfgResource config.Resource,
) map[string]string {
	// In the configuration, the language mappings are "remote code -> local
	// code" (eg 'pt_BT: pt-br'). Looking into the filesystem, we get the local
	// language codes; so if we need to find the remote codes, we need to
	// reverse the maps

	result := make(map[string]string)
	for transifexLanguageCode, localLanguageCode := range cfg.Local.LanguageMappings {
		result[transifexLanguageCode] = localLanguageCode
	}
	for transifexLanguageCode, localLanguageCode := range cfgResource.LanguageMappings {
		// Resource language mappings overwrite "global" language mappings
		result[transifexLanguageCode] = localLanguageCode
	}
	return result
}

func reverseMap(src map[string]string) map[string]string {
	dst := make(map[string]string)
	for key, value := range src {
		dst[value] = key
	}
	return dst
}

/*
Run 'do'. If the error returned by 'do' is a jsonapi.RetryError, sleep the number of
seconds indicated by the error and try again. Meanwhile, inform the user of
what's going on using 'send'.
*/
func handleRetry(do func() error, initialMsg string, send func(string)) error {
	for {
		if len(initialMsg) > 0 {
			send(initialMsg)
		}
		err := do()
		if err == nil {
			return nil
		} else {
			var e *jsonapi.RetryError
			if errors.As(err, &e) {
				retryAfter := e.RetryAfter
				if isatty.IsTerminal(os.Stdout.Fd()) {
					for retryAfter > 0 {
						send(fmt.Sprint(
							err,
						))
						time.Sleep(time.Second)
						retryAfter -= 1
					}
				} else {
					send(fmt.Sprint(
						err,
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

func isValidResolutionPolicy(policy string) (IsValid bool) {
	res := [2]string{"USE_HEAD", "USE_BASE"}
	for _, requestPolicy := range res {
		if requestPolicy == policy {
			return true
		}
	}
	return false

}

type getSizeFuncType func(fd int) (int, int, error)

var getSizeFunc getSizeFuncType = term.GetSize

func truncateMessage(message string) string {
	width, _, err := getSizeFunc(int(os.Stdout.Fd()))
	if err != nil {
		width = 80
	}

	maxLength := width - 2
	if maxLength < 0 {
		maxLength = 0
	}

	if len(message) > maxLength && maxLength > 0 {
		return message[:maxLength] + ".."
	}
	return message
}
