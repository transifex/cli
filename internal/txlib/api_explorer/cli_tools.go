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

func addRelationshipCommand(
	cmd *cli.Command, verb, resourceName, relationshipName string, jsopenapi *jsopenapi_t,
) {
	resource := jsopenapi.Resources[resourceName]
	relationship := resource.Relationships[relationshipName]

	var cliFunc func(*cli.Context, string, string, *jsopenapi_t) error
	var summary string
	if verb == "get" {
		cliFunc = cliCmdGetRelated
		summary = relationship.Operations.Get.Summary
	} else if verb == "change" {
		cliFunc = cliCmdChange
		summary = relationship.Operations.Change.Summary
	} else if verb == "add" {
		cliFunc = cliCmdAdd
		summary = relationship.Operations.Add.Summary
	} else if verb == "remove" {
		cliFunc = cliCmdRemove
		summary = relationship.Operations.Remove.Summary
	} else if verb == "reset" {
		cliFunc = cliCmdReset
		summary = relationship.Operations.Reset.Summary
	} else {
		panic("Wrong verb")
	}

	subcommand := getOrCreateSubcommand(cmd, verb)
	parent := getOrCreateSubcommand(subcommand, resource.SingularName)
	if !flagExists(parent.Flags, "id") {
		parent.Flags = append(parent.Flags, &cli.StringFlag{
			Name: "id",
			// If we want to `get something` and the `somethings`
			// resource does not support `get_many`, then the user
			// won't be able to fuzzy-select the something and
			// `--id` should be required
			Required: resource.Operations.GetMany == nil,
		})
	}
	addFilterTags(parent, resourceName, jsopenapi, true)
	operation := &cli.Command{
		Name:  relationshipName,
		Usage: summary,
		Action: func(c *cli.Context) error {
			return cliFunc(c, resourceName, relationshipName, jsopenapi)
		},
	}

	if verb != "get" {
		addFilterTags(operation, relationship.Resource, jsopenapi, true)
		if verb == "change" {
			operation.Flags = append(
				operation.Flags,
				&cli.StringFlag{Name: "related-id", Usage: "ID to use for the relationship"},
			)
		} else {
			operation.Flags = append(
				operation.Flags,
				&cli.StringFlag{
					Name:  "ids",
					Usage: "Comma-separated IDs to use for the relationship",
				},
			)
		}
	}

	parent.Subcommands = append(parent.Subcommands, operation)
}

func addCreateFlags(operation *cli.Command, resourceName string, jsopenapi *jsopenapi_t) {
	resource := jsopenapi.Resources[resourceName]
	for _, attribute := range resource.Operations.CreateOne.Attributes.Required {
		operation.Flags = append(operation.Flags, &cli.StringFlag{Name: attribute})
	}
	for _, attribute := range resource.Operations.CreateOne.Attributes.Optional {
		operation.Flags = append(operation.Flags, &cli.StringFlag{Name: attribute})
	}
	requiredRelationships := resource.Operations.CreateOne.Relationships.Required
	for relationshipName := range requiredRelationships {
		operation.Flags = append(
			operation.Flags,
			&cli.StringFlag{Name: fmt.Sprintf("%s-id", relationshipName)},
		)
	}
	optionalRelationships := resource.Operations.CreateOne.Relationships.Optional
	for relationshipName := range optionalRelationships {
		operation.Flags = append(
			operation.Flags,
			&cli.StringFlag{Name: fmt.Sprintf("%s-id", relationshipName)},
		)
	}
}
