package txlib

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
	"github.com/transifex/cli/pkg/txapi"
	"github.com/transifex/cli/pkg/worker_pool"
)

type MergeCommandArguments struct {
	ResourceId         string
	Branch             string
	ConflictResolution string
	Skip               bool
	Silent             bool
}

func MergeCommand(
	cfg *config.Config,
	api jsonapi.Connection,
	args MergeCommandArguments,
) error {
	args.Branch = figureOutBranch(args.Branch)

	cfgResources, err := figureOutResources([]string{args.ResourceId}, cfg)
	if err != nil {
		return err
	}

	applyBranchToResources(cfgResources, args.Branch, "")

	cfgResource := cfgResources[0]

	err = mergeResource(&api, cfgResource, args)

	return err
}

func mergeResource(
	api *jsonapi.Connection, cfgResource *config.Resource, args MergeCommandArguments,
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

	var merge *jsonapi.Resource
	merge, err = txapi.CreateAsyncResourceMerge(api, resource, args.ConflictResolution)
	if err != nil {
		return err
	}

	pool := worker_pool.New(1, 1, args.Silent)
	pool.Add(&MergeResourcePollTask{merge, args})
	pool.Start()
	<-pool.Wait()
	if pool.IsAborted {
		return errors.New("Aborted")
	}

	return nil
}

type MergeResourcePollTask struct {
	merge *jsonapi.Resource
	args  MergeCommandArguments
}

func (task *MergeResourcePollTask) Run(send func(string), abort func()) {
	merge := task.merge
	args := task.args

	parts := strings.Split(merge.Relationships["base"].DataSingular.Id, ":")
	sendMessage := func(body string, force bool) {
		if args.Silent && !force {
			return
		}
		send(fmt.Sprintf(
			"%s.%s - %s", parts[3], parts[5], body,
		))
	}

	err := handleThrottling(
		func() error {
			return txapi.PollResourceMerge(
				merge,
				time.Second,
			)
		},
		"Polling merge task status",
		func(msg string) { sendMessage(msg, false) },
	)
	if err != nil {
		sendMessage(err.Error(), true)
		if !args.Skip {
			abort()
		}
		return
	}
	sendMessage("Done", false)
}
