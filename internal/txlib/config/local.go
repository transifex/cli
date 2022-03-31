package config

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

type LocalConfig struct {
	Host             string
	LanguageMappings map[string]string
	Resources        []Resource
	Path             string
}

type Resource struct {
	OrganizationSlug  string
	ProjectSlug       string
	ResourceSlug      string
	FileFilter        string
	SourceFile        string
	SourceLanguage    string
	Type              string
	LanguageMappings  map[string]string
	Overrides         map[string]string
	MinimumPercentage int
}

func loadLocalConfig() (*LocalConfig, error) {
	localPath, err := findLocalPath("")
	if err != nil {
		return nil, err
	}
	return loadLocalConfigFromPath(localPath)
}

func loadLocalConfigFromPath(path string) (*LocalConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"local configuration file does not exist, run 'tx init' "+
					"first: %w",
				err,
			)
		} else {
			return nil, err
		}
	}
	localCfg, err := loadLocalConfigFromBytes(data)
	if err != nil {
		return nil, err
	}
	localCfg.Path = path
	return localCfg, nil
}

func loadLocalConfigFromBytes(data []byte) (*LocalConfig, error) {
	result := LocalConfig{
		LanguageMappings: make(map[string]string),
	}

	cfg, err := ini.Load(data)
	if err != nil {
		return nil, err
	}

	mainSection := cfg.Section("main")
	if mainSection == nil {
		return nil, errors.New("local config file has no main section")
	}
	result.Host = mainSection.Key("host").String()
	if result.Host == "" {
		return nil, errors.New("local config's main section has no host")
	}
	languageMappings := mainSection.Key("lang_map").String()
	if languageMappings != "" {
		for _, mapping := range strings.Split(languageMappings, ",") {
			err := fmt.Errorf("invalid language mapping '%s'", mapping)

			split := strings.Split(mapping, ":")
			if len(split) != 2 {
				return nil, err
			}
			key := strings.Trim(split[0], " ")
			value := strings.Trim(split[1], " ")
			if key == "" || value == "" {
				return nil, err
			}
			result.LanguageMappings[key] = value
		}
	}

	for _, section := range cfg.Sections() {
		if section.Name() == "main" || section.Name() == "DEFAULT" {
			continue
		}

		var organizationSlug, projectSlug, resourceSlug string

		// If : is there these are new resources, if not it's a migration case
		if strings.Contains(section.Name(), ":") {
			organizationSlug, projectSlug, resourceSlug, err = nameToSlugs(
				section.Name(),
			)
		} else {
			organizationSlug, projectSlug,
				resourceSlug, err = nameToSlugsForMigrate(
				section.Name(),
			)
		}
		if err != nil {
			return nil, err
		}

		resource := Resource{
			OrganizationSlug:  organizationSlug,
			ProjectSlug:       projectSlug,
			ResourceSlug:      resourceSlug,
			FileFilter:        section.Key("file_filter").String(),
			SourceFile:        section.Key("source_file").String(),
			SourceLanguage:    section.Key("source_lang").String(),
			Type:              section.Key("type").String(),
			LanguageMappings:  make(map[string]string),
			Overrides:         make(map[string]string),
			MinimumPercentage: -1,
		}

		// Get first the perc in string to check if exists because .Key returns
		// 0 if it doesn't exist
		if section.HasKey("minimum_perc") {
			minimum_perc, err := section.Key("minimum_perc").Int()
			if err == nil {
				resource.MinimumPercentage = minimum_perc
			}
		}

		languageMappings := section.Key("lang_map").String()
		if languageMappings != "" {
			for _, mapping := range strings.Split(languageMappings, ",") {
				err := fmt.Errorf("invalid language mapping %s", mapping)
				split := strings.Split(mapping, ":")
				if len(split) != 2 {
					return nil, err
				}
				key := strings.Trim(split[0], " ")
				value := strings.Trim(split[1], " ")
				if key == "" || value == "" {
					return nil, err
				}
				resource.LanguageMappings[key] = value
			}
		}

		for _, key := range section.Keys() {
			if strings.Index(key.Name(), "trans.") != 0 {
				continue
			}
			code := key.Name()[len("trans."):]
			resource.Overrides[code] = key.String()
		}

		result.Resources = append(result.Resources, resource)
	}

	result.sortResources()

	return &result, nil
}

func (localCfg LocalConfig) Save() error {
	return localCfg.saveToPath(localCfg.Path)
}

func (localCfg LocalConfig) saveToPath(path string) error {
	file, err := os.OpenFile(path,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0755)
	if err != nil {
		return err
	}
	defer file.Close()
	return localCfg.saveToWriter(file)
}

func (localCfg LocalConfig) saveToWriter(file io.Writer) error {
	cfg := ini.Empty(ini.LoadOptions{})

	main, err := cfg.NewSection("main")
	if err != nil {
		return err
	}
	_, err = main.NewKey("host", localCfg.Host)
	if err != nil {
		return err
	}
	if len(localCfg.LanguageMappings) != 0 {
		var mappings []string
		for key, value := range localCfg.LanguageMappings {
			mappings = append(mappings, fmt.Sprintf("%s: %s", key, value))
		}
		_, err = main.NewKey("lang_map", strings.Join(mappings, ", "))
		if err != nil {
			return err
		}
	}

	for _, resource := range localCfg.Resources {
		section, err := cfg.NewSection(resource.Name())
		if err != nil {
			return err
		}

		if resource.FileFilter != "" {
			_, err := section.NewKey("file_filter", resource.FileFilter)
			if err != nil {
				return err
			}
		}

		if resource.SourceFile != "" {
			_, err := section.NewKey("source_file", resource.SourceFile)
			if err != nil {
				return err
			}
		}

		if resource.SourceLanguage != "" {
			_, err := section.NewKey("source_lang", resource.SourceLanguage)
			if err != nil {
				return err
			}
		}

		if resource.Type != "" {
			_, err := section.NewKey("type", resource.Type)
			if err != nil {
				return err
			}
		}

		if resource.MinimumPercentage != -1 {
			_, err := section.NewKey("minimum_perc",
				strconv.Itoa(resource.MinimumPercentage))
			if err != nil {
				return err
			}
		}

		if len(resource.LanguageMappings) != 0 {
			var mappings []string
			for key, value := range resource.LanguageMappings {
				mappings = append(mappings, fmt.Sprintf("%s: %s", key, value))
			}
			_, err = section.NewKey("lang_map", strings.Join(mappings, ", "))
			if err != nil {
				return err
			}
		}

		if len(resource.Overrides) != 0 {
			for key, value := range resource.Overrides {
				_, err = section.NewKey(fmt.Sprintf("trans.%s", key), value)
				if err != nil {
					return err
				}
			}
		}
	}

	_, err = cfg.WriteTo(file)
	return err
}

func (localCfg *LocalConfig) sortResources() {
	sort.Slice(localCfg.Resources, func(i, j int) bool {
		left := localCfg.Resources[i].Name()
		right := localCfg.Resources[j].Name()
		return strings.Compare(left, right) == -1
	})
}

func localConfigsEqual(left, right *LocalConfig) bool {
	if left.Host != right.Host {
		return false
	}

	if len(left.LanguageMappings) != len(right.LanguageMappings) {
		return false
	}
	for key, leftValue := range left.LanguageMappings {
		rightValue, exists := right.LanguageMappings[key]
		if !exists {
			return false
		}
		if leftValue != rightValue {
			return false
		}
	}

	if len(left.Resources) != len(right.Resources) {
		return false
	}
	for i, leftResource := range left.Resources {
		rightResource := right.Resources[i]

		if leftResource.Name() != rightResource.Name() {
			return false
		}
		if leftResource.FileFilter != rightResource.FileFilter {
			return false
		}
		if leftResource.SourceFile != rightResource.SourceFile {
			return false
		}
		if leftResource.SourceLanguage != rightResource.SourceLanguage {
			return false
		}
		if leftResource.Type != rightResource.Type {
			return false
		}

		if leftResource.MinimumPercentage != rightResource.MinimumPercentage {
			return false
		}

		if len(leftResource.LanguageMappings) !=
			len(rightResource.LanguageMappings) {
			return false
		}
		for key, leftValue := range leftResource.LanguageMappings {
			rightValue, exists := rightResource.LanguageMappings[key]
			if !exists {
				return false
			}
			if leftValue != rightValue {
				return false
			}
		}

		if len(leftResource.Overrides) != len(rightResource.Overrides) {
			return false
		}
		for key, leftValue := range leftResource.Overrides {
			rightValue, exists := rightResource.Overrides[key]
			if !exists {
				return false
			}
			if leftValue != rightValue {
				return false
			}
		}
	}

	return true
}

func nameToSlugs(in string) (string, string, string, error) {
	parts := strings.Split(in, ":")
	if len(parts) != 6 {
		return "", "", "", fmt.Errorf(
			"wrong number of parts in resource ID '%s'", in,
		)
	}
	if parts[0] != "o" || parts[2] != "p" || parts[4] != "r" {
		return "", "", "", fmt.Errorf("invalid resource ID '%s'", in)
	}

	return parts[1], parts[3], parts[5], nil
}

func nameToSlugsForMigrate(in string) (string, string, string, error) {
	parts := strings.Split(in, ".")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf(
			"wrong number of parts in resource ID '%s'", in,
		)
	}

	return "", parts[0], parts[1], nil
}

/*
Name Return the name of a resource as it appears in the configuration file. The
format is the same as the ID of the resource in APIv3 */
func (localCfg *Resource) Name() string {
	var result string
	if localCfg.OrganizationSlug != "" {
		result = fmt.Sprintf(
			"o:%s:p:%s:r:%s",
			localCfg.OrganizationSlug,
			localCfg.ProjectSlug,
			localCfg.ResourceSlug,
		)
	} else {
		result = fmt.Sprintf(
			"%s.%s",
			localCfg.ProjectSlug,
			localCfg.ResourceSlug,
		)
	}
	return result
}

func (localCfg *Resource) ResourceName() string {
	parts := strings.Split(localCfg.SourceFile, string(os.PathSeparator))
	return parts[len(parts)-1]
}

func getLocalPath() (string, error) {
	curDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(curDir, ".tx", "config"), nil
}

func findLocalPath(path string) (string, error) {
	curDir := path
	if path == "" {
		dir, err := os.Getwd()
		if err != nil {
			return "", err
		}
		curDir = dir
	}

	fp := filepath.Join(curDir, ".tx", "config")
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		curDir = filepath.Dir(curDir)
		if curDir != "/" && curDir != "." {
			return findLocalPath(curDir)
		} else {
			return "", nil
		}

	}
	return filepath.Join(curDir, ".tx", "config"), nil
}

func (localCfg *Resource) GetAPv3Id() string {
	return fmt.Sprintf(
		"o:%s:p:%s:r:%s",
		localCfg.OrganizationSlug,
		localCfg.ProjectSlug,
		localCfg.ResourceSlug,
	)
}
