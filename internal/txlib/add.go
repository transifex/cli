package txlib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/transifex/cli/pkg/txapi"

	"github.com/manifoldco/promptui"
	"github.com/pterm/pterm"
	"github.com/transifex/cli/internal/txlib/config"
	"github.com/transifex/cli/pkg/jsonapi"
)

var PromptMap = map[string]map[string]string{
	"sourceFile": {
		"text": `
The Transifex Client syncs files between your local directory and Transifex.
The mapping configuration between the two is stored in a file called .tx/config
in your current directory. For more information, visit
https://docs.transifex.com/client/config/.`,
		"label": "What is the path of the source file?",
	},
	"fileFilter": {
		"text": `
Next, we’ll need a path expression pointing to the location of the
translation files (whether they exist yet or not) associated with
the source file ‘%s’.
You should include <lang> as a wildcard for the language code.
Example: 'path/<lang>/%s'`,
		"label": "What is your path expression?",
	},
}

type AddCommandArguments struct {
	OrganizationSlug string
	ProjectSlug      string
	ResourceSlug     string
	FileFilter       string
	RType            string
	SourceFile       string
}

func validateFileFilter(input string) error {
	res := strings.Count(input, "<lang>")
	if res != 1 {
		return errors.New("you need one <lang> in your File Filter")
	}
	if len(filepath.Ext(input)) < 2 {
		return errors.New("you need to add an extension to your file")
	}
	return nil
}

func validateSourceFile(input string) error {
	if len(input) < 1 {
		return errors.New("you need to add a Source File")
	}

	if len(filepath.Ext(input)) < 2 {
		return errors.New("you need to add an extension to your Source File")
	}

	curDir, err := os.Getwd()
	_, err = os.Stat(filepath.Join(curDir, input))

	if err != nil {
		return errors.New("you need to add a Source File that exists")
	}
	return nil
}

func validateResourceSlug(input string) error {
	if len(input) < 1 {
		return errors.New("you need to add a Resource Slug")
	}
	return nil
}

func i18nFormatExists(list []string, ext string) bool {
	for _, value := range list {
		if value == ext {
			return true
		}

	}
	return false
}

func getSelectTemplate(str string) *promptui.SelectTemplates {
	var template = &promptui.SelectTemplates{
		Active:   "> {{.Name }} ({{.Value | faint}})",
		Inactive: "  {{.Name }} ({{.Value | faint}})",
		Selected: fmt.Sprintf(`%s {{ "%s:" | faint }} {{ .Name }}`,
			promptui.IconGood, str),
	}
	return template
}

func getInputTemplate(str string) *promptui.PromptTemplates {
	var template = &promptui.PromptTemplates{
		Prompt:  fmt.Sprintf("%s {{ . }} ", promptui.IconInitial),
		Valid:   fmt.Sprintf("%s {{ . }} ", promptui.IconGood),
		Invalid: fmt.Sprintf("%s {{ . }} ", promptui.IconBad),
		Success: fmt.Sprintf(`%s {{ "%s:" | faint }} `,
			promptui.IconGood, str),
	}
	return template
}

func AddCommandInteractive(cfg *config.Config, api jsonapi.Connection) error {
	type selectedItem struct {
		Name  string
		Value string
	}
	var answers AddCommandArguments
	var selectItems []selectedItem

	// Add the ability to search in lists
	searchList := func(input string, index int) bool {
		item := selectItems[index]
		name := strings.Replace(strings.ToLower(item.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	fmt.Println(PromptMap["sourceFile"]["text"])
	fmt.Println()

	// Prompt for a Source File
	inputPrompt := promptui.Prompt{
		Label:     PromptMap["sourceFile"]["label"],
		Templates: getInputTemplate("Selected Source file"),
		Validate:  validateSourceFile,
	}

	// Run prompt
	res, err := inputPrompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			return err
		} else {
			return fmt.Errorf("something went wrong: %v", err)
		}
	}

	answers.SourceFile = res
	_, fileName := filepath.Split(res)
	fmt.Printf(PromptMap["fileFilter"]["text"], res, fileName)
	fmt.Println()

	// Prompt for File Filter
	inputPrompt = promptui.Prompt{
		Label:     PromptMap["fileFilter"]["label"],
		Templates: getInputTemplate("Selected File Filter"),
		Validate:  validateFileFilter,
	}

	// Run prompt
	res, err = inputPrompt.Run()

	if err != nil {
		if err == promptui.ErrInterrupt {
			return err
		} else {
			return fmt.Errorf("something went wrong: %v", err)
		}
	}

	answers.FileFilter = res

	// Get List of Organizations
	organizations, err := txapi.GetOrganizations(&api)

	if err != nil {
		return fmt.Errorf("API Error: %w", err)
	}

	// Create an array of organizations
	for _, value := range organizations {
		selectItems = append(selectItems, selectedItem{
			Name:  fmt.Sprintf("%s", value.Attributes["name"]),
			Value: fmt.Sprintf("%s", value.Attributes["slug"]),
		})
	}

	// Return no items error
	if len(selectItems) == 0 {
		return fmt.Errorf("we got no Organization results. Maybe create one " +
			"and come back")
	}

	// Create the user prompt
	prompt := promptui.Select{
		Label:     "Which organization will this resource be part of?",
		Items:     selectItems,
		Templates: getSelectTemplate("Selected organization"),
		Searcher:  searchList,
	}

	// Run prompt
	fmt.Println()
	idx, _, err := prompt.Run()

	if err != nil {
		if err == promptui.ErrInterrupt {
			return err
		} else {
			return fmt.Errorf("something went wrong: %v", err)
		}
	}

	answers.OrganizationSlug = selectItems[idx].Value
	var selectedOrganization = organizations[idx]

	// Prompt for projects
	selectItems = nil
	projects, err := txapi.GetProjects(&api, selectedOrganization)
	if err != nil {
		return fmt.Errorf("API Error: %w", err)
	}

	for _, value := range projects {
		selectItems = append(selectItems, selectedItem{
			Name:  fmt.Sprintf("%s", value.Attributes["name"]),
			Value: fmt.Sprintf("%s", value.Attributes["slug"]),
		})
	}
	// Return no items error
	if len(selectItems) == 0 {
		return fmt.Errorf("we found no Projects. Maybe create one and come " +
			"back")
	}

	prompt = promptui.Select{
		Label:     "Which project will this resource be part of?",
		Items:     selectItems,
		Templates: getSelectTemplate("Selected project"),
		Searcher:  searchList,
	}

	fmt.Println()
	idx, _, err = prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			return err
		} else {
			return fmt.Errorf("something went wrong: %v", err)
		}
	}

	answers.ProjectSlug = selectItems[idx].Value
	var selectedProject = projects[idx]

	// Prompt for Resources
	selectItems = nil
	resources, err := txapi.GetResources(&api, selectedProject)
	if err != nil {
		return fmt.Errorf("API Error: %w", err)
	}

	for _, value := range resources {
		selectItems = append(selectItems, selectedItem{
			Name:  fmt.Sprintf("%s", value.Attributes["name"]),
			Value: fmt.Sprintf("%s", value.Attributes["slug"]),
		})
	}

	// Append new resource to the end of the list
	selectItems = append(selectItems, selectedItem{
		Name:  "Create a new resource",
		Value: "",
	})

	prompt = promptui.Select{
		Label:     "Which is the resource for this file?",
		Items:     selectItems,
		Templates: getSelectTemplate("Selected resource"),
		Searcher:  searchList,
	}

	// Run prompt
	fmt.Println()
	idx, _, err = prompt.Run()
	if err != nil {
		return err
	}
	var selectedResource *jsonapi.Resource
	// If the value of the selected item is "" then it's the new resource
	// option
	if selectItems[idx].Value == "" {
		inputPrompt = promptui.Prompt{
			Label:     "What is the slug of your resource?",
			Templates: getInputTemplate("New Resource Slug"),
			Validate:  validateResourceSlug,
		}

		res, err = inputPrompt.Run()

		if err != nil {
			if err == promptui.ErrInterrupt {
				return err
			} else {
				return fmt.Errorf("something went wrong: %v", err)
			}
		}

		// Add the slug to the answers
		answers.ResourceSlug = res
	} else {
		// In case it's a preexisting resource add it to answers and
		// selected resource
		answers.ResourceSlug = selectItems[idx].Value
		selectedResource = resources[idx]
	}

	// If we have a selected resource get the file format from the
	// relationships if not, prompt for i18n formats
	if selectItems[idx].Value != "" &&
		selectedResource != nil {
		answers.RType = selectedResource.
			Relationships["i18n_format"].DataSingular.Id
	} else {
		// Get Formats
		selectItems = nil
		formats, err := txapi.GetI18nFormats(&api, selectedOrganization)
		if err != nil {
			return err
		}
		fileExtension := filepath.Ext(answers.SourceFile)
		for _, value := range formats {
			var i18nFormatsAttributes txapi.I18nFormatsAttributes
			_ = value.MapAttributes(&i18nFormatsAttributes)
			// Add selection only if file extension is included in the extensions
			if i18nFormatExists(i18nFormatsAttributes.FileExtensions, fileExtension) {
				selectItems = append(selectItems, selectedItem{
					Name: i18nFormatsAttributes.Name,
					Value: i18nFormatsAttributes.Description + " " +
						strings.Join(i18nFormatsAttributes.FileExtensions, ", "),
				})
			}
		}

		// Return no items error
		if len(selectItems) == 0 {
			return fmt.Errorf("we found no I18n Formats associated with " +
				"this file. Maybe choose another file")
		}

		prompt = promptui.Select{
			Label:     "What is the file format of the source file?",
			Items:     selectItems,
			Templates: getSelectTemplate("Selected format"),
			Searcher:  searchList,
		}

		fmt.Println()
		idx, _, err = prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				return err
			} else {
				return fmt.Errorf("something went wrong: %v", err)
			}
		}

		answers.RType = selectItems[idx].Name
	}
	err = AddCommand(cfg, &answers)
	if err != nil {
		return err
	}
	return nil
}

func AddCommand(
	cfg *config.Config,
	args *AddCommandArguments,
) error {

	if args.SourceFile == "" {
		return fmt.Errorf("a source file is required to proceed")
	}

	err := validateFileFilter(args.FileFilter)

	if err != nil {
		return err
	}

	cfg.AddResource(config.Resource{
		OrganizationSlug: args.OrganizationSlug,
		ProjectSlug:      args.ProjectSlug,
		ResourceSlug:     args.ResourceSlug,
		FileFilter:       args.FileFilter,
		SourceFile:       args.SourceFile,
		Type:             args.RType,
	})

	err = cfg.Save()
	if err != nil {
		return err
	}

	fmt.Println()
	pterm.Success.Println(`Your configuration has been saved in '.tx/config'
You can now push and pull content with 'tx push' and 'tx pull'`)

	return nil
}
