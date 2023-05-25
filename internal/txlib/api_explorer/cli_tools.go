package api_explorer

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
)

func findSubcommand(subcommands []*cli.Command, name string) *cli.Command {
	for _, subcommand := range subcommands {
		if subcommand.Name == name {
			return subcommand
		}
	}
	return nil
}

func addFilterTags(
	command *cli.Command, resourceName string, jsopenapi *jsopenapi_t, optional bool,
) {
	resource := jsopenapi.Resources[resourceName]
	if resource.Operations.GetMany == nil {
		return
	}
	for filterName, filter := range resource.Operations.GetMany.Filters {
		if filter.Resource != "" {
			flagName := fmt.Sprintf("%s-id", filterName)
			if !flagExists(command.Flags, flagName) {
				command.Flags = append(
					command.Flags,
					&cli.StringFlag{Name: flagName, Usage: filter.Description},
				)
			}
		} else {
			flagName := strings.ReplaceAll(filterName, "__", "-")
			if !flagExists(command.Flags, flagName) {
				command.Flags = append(
					command.Flags,
					&cli.StringFlag{
						Name:     strings.ReplaceAll(filterName, "__", "-"),
						Usage:    filter.Description,
						Required: !optional && filter.Required,
					},
				)
			}
		}
	}
}

func flagExists(flags []cli.Flag, name string) bool {
	for _, flag := range flags {
		if stringSliceContains(flag.Names(), name) {
			return true
		}
	}
	return false
}

func getOrCreateSubcommand(parent *cli.Command, name string) *cli.Command {
	subcommand := findSubcommand(parent.Subcommands, name)
	if subcommand == nil {
		subcommand = &cli.Command{Name: name}
		parent.Subcommands = append(parent.Subcommands, subcommand)
	}
	return subcommand
}
