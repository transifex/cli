package api_explorer

import (
	"errors"
	"fmt"
	"sort"

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

func addFilterFlags(
	command *cli.Command, resourceName string, jsopenapi *jsopenapi_t, optional bool,
) error {
	resource := jsopenapi.Resources[resourceName]
	if resource.Operations.GetMany == nil {
		return nil
	}
	for _, parameter := range resource.Operations.GetMany.Parameters {
		flagName, err := getFlagName(parameter.Name)
		if err != nil {
			return err
		}
		if !flagExists(command.Flags, flagName) {
			command.Flags = append(
				command.Flags,
				&cli.StringFlag{
					Name:     flagName,
					Usage:    parameter.Description,
					Required: parameter.Resource == "" && !optional && parameter.Required,
				},
			)
		}
	}
	return nil
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
) error {
	resource := jsopenapi.Resources[resourceName]
	relationship := resource.ResponseRelationships[relationshipName]

	var cliFunc func(*cli.Context, string, string, *jsopenapi_t) error
	var summary string
	if verb == "get" {
		cliFunc = cliCmdGetRelated
		if relationship.Operations.Get != nil {
			summary = relationship.Operations.Get.Summary
		} else {
			summary = ""
		}
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
		return errors.New("wrong verb")
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
	err := addFilterFlags(parent, resourceName, jsopenapi, true)
	if err != nil {
		return err
	}
	operation := &cli.Command{
		Name:  relationshipName,
		Usage: summary,
		Action: func(c *cli.Context) error {
			return cliFunc(c, resourceName, relationshipName, jsopenapi)
		},
	}

	if verb != "get" {
		err := addFilterFlags(operation, relationship.Resource, jsopenapi, true)
		if err != nil {
			return err
		}
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
	return nil
}

func getCreateFlags(
	resourceName string, jsopenapi *jsopenapi_t, hasContent bool,
) ([]cli.Flag, error) {
	resource := jsopenapi.Resources[resourceName]

	var fields []string
	fields = append(fields, resource.Operations.CreateOne.RequiredFields...)
	fields = append(fields, resource.Operations.CreateOne.OptionalFields...)

	var result []cli.Flag
	for _, field := range fields {
		if hasContent && (field == "content" || field == "content_encoding") {
			continue
		}
		_, isAttribute := resource.RequestAttributes[field]
		_, isRelationship := resource.RequestRelationships[field]
		if isAttribute {
			result = append(
				result,
				&cli.StringFlag{
					Name:  field,
					Usage: resource.RequestAttributes[field].Description,
				},
			)
		} else if isRelationship {
			result = append(
				result,
				&cli.StringFlag{
					Name:  fmt.Sprintf("%s-id", field),
					Usage: resource.RequestRelationships[field].Description,
				},
			)
		} else {
			return nil, fmt.Errorf("unknown field %s of %s", field, resourceName)
		}
	}
	sort.Sort(cli.FlagsByName(result))
	return result, nil
}
