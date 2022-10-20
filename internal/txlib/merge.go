package txlib

import (
	"fmt"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
)

type MergeCommandArguments struct {
	ResourceId         string
	Branch             string
	ConflictResolution string
}

func MergeCommand(
	cfg *config.Config,
	api jsonapi.Connection,
	args MergeCommandArguments,
) error {
	args.Branch = figureOutBranch(args.Branch)

	cfgResources, err := figureOutResources([]string{args.ResourceId}, cfg)

	applyBranchToResources(cfgResources, args.Branch, "")
	if err != nil {
		return err
	}
	cfgResource := cfgResources[0]

	mergeResource(&api, cfgResource, args.ConflictResolution)

	return nil
}

func mergeResource(
	api *jsonapi.Connection, cfgResource *config.Resource, conflictResolution string,
) error {
	organization, err := txapi.GetOrganization(api,
		cfgResource.OrganizationSlug)
	if err != nil {
		return err
	}

	if organization == nil {
		return fmt.Errorf("organization '%s' not found",
			cfgResource.OrganizationSlug)
	}

	// Get Project from Server
	project, err := txapi.GetProject(api, organization,
		cfgResource.ProjectSlug)
	if err != nil {
		return err
	}

	if project == nil {
		return fmt.Errorf("project '%s - %s' not found",
			cfgResource.OrganizationSlug,
			cfgResource.ProjectSlug)

	}

	// Get Resource from Server
	resource, err := txapi.GetResource(api, project, cfgResource.ResourceSlug)
	if err != nil {
		return err
	}

	if resource == nil {
		return fmt.Errorf("resource '%s - %s - %s' not found",
			cfgResource.OrganizationSlug,
			cfgResource.ProjectSlug,
			cfgResource.ResourceSlug)
	}

	err = txapi.CreateAsyncResourceMerge(api, resource, conflictResolution)

	return err
}
