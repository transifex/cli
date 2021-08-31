/*
Package config
Slightly object-oriented tx configuration package.

Usage:

    import "github.com/transifex/cli/internal/txlib/config"

    cfg, err := config.Load()  // Loads based on current directory
    if err != nil { ... }

    // Lets add a resource
    cfg.AddResource(config.Resource{
        OrganizationSlug: "my_org",
        ProjectSlug: "my_project",
        ResourceSlug: "my_resource",
        FileFilter: "locale/<lang>.po",
        SourceFile: "locale/en.po",
        SourceLanguage: "en",
        Type: "PO",
    })

    cfg.Save()  // Saves changes to disk

    resource := cfg.FindResource("my_org.my_project")

    file, err := os.Open(resource.SourceFile)
    if err != nil { ... }
    defer file.Close()

    resource.LanguageMappings["en_US"] = "en-us"
    cfg.Save()

*/
package config

import (
	"strings"
)

type Config struct {
	Root  *RootConfig
	Local *LocalConfig
}

/*
Load Transifex configuration from the usual paths:

- ~/.transifexrc for the root configuration

- ./.tx/config for the local configuration

If any of these files are missing, the relevant attribute will be set to nil.

TODO: Load local configuration from any parent folder's `.tx` folder.
*/
func Load() (Config, error) {
	rootConfig, err := loadRootConfig()
	if err != nil {
		return Config{}, err
	}

	localConfig, err := loadLocalConfig()
	if err != nil {
		return Config{}, err
	}

	return Config{Root: rootConfig, Local: localConfig}, nil
}

func LoadFromPaths(rootPath, localPath string) (Config, error) {
	var err error
	var rootConfig *RootConfig
	if rootPath == "" {
		rootConfig, err = loadRootConfig()
	} else {
		rootConfig, err = loadRootConfigFromPath(rootPath)
	}
	if err != nil {
		return Config{}, err
	}

	var localConfig *LocalConfig
	if localPath == "" {
		localConfig, err = loadLocalConfig()
	} else {
		localConfig, err = loadLocalConfigFromPath(localPath)
	}
	if err != nil {
		return Config{}, err
	}

	return Config{Root: rootConfig, Local: localConfig}, nil
}

/*
GetActiveHost
Return the URL that will be used based on the configuration.

The local configuration has a 'host' field in its 'main' section. That host
points to a section in the root configuration. We return the rest_hostname of
that section. The fallback value is `https://rest.api.transifex.com` */
func (cfg *Config) GetActiveHost() *Host {
	if cfg.Root.Hosts == nil || len(cfg.Root.Hosts) == 0 ||
		cfg.Local == nil {
		return nil
	}
	activeHostName := cfg.Local.Host
	for i := range cfg.Root.Hosts {
		host := &cfg.Root.Hosts[i]
		if host.Name == activeHostName {
			return host
		}
	}
	return nil
}

/*
Save
Save changes to disk */
func (cfg *Config) Save() error {
	if cfg.Root != nil {
		var oldRootConfig *RootConfig
		var err error
		if cfg.Root.Path != "" {
			oldRootConfig, err = loadRootConfigFromPath(cfg.Root.Path)
		} else {
			oldRootConfig, err = loadRootConfig()
		}
		if err != nil {
			return err
		}

		cfg.Root.sortHosts()

		if !rootConfigsEqual(oldRootConfig, cfg.Root) {
			err = cfg.Root.save()
			if err != nil {
				return err
			}
		}
	}

	if cfg.Local != nil {
		var oldLocalConfig *LocalConfig
		var err error
		if cfg.Local.Path != "" {
			oldLocalConfig, err = loadLocalConfigFromPath(cfg.Local.Path)
		} else {
			oldLocalConfig, err = loadLocalConfig()
		}
		if err != nil {
			return err
		}

		cfg.Local.sortResources()

		if !localConfigsEqual(oldLocalConfig, cfg.Local) {
			err = cfg.Local.Save()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

/*
FindHost
Return a Host reference that matches the argument.
*/
func (cfg *Config) FindHost(hostname string) *Host {
	if cfg.Root.Hosts == nil {
		return nil
	}
	for i := range cfg.Root.Hosts {
		// range returns copies: https://stackoverflow.com/q/20185511
		host := &cfg.Root.Hosts[i]
		if host.Name == hostname {
			return host
		}
	}
	for i := range cfg.Root.Hosts {
		// range returns copies: https://stackoverflow.com/q/20185511
		host := &cfg.Root.Hosts[i]
		if host.RestHostname == hostname {
			return host
		}
	}
	return nil
}

/*
FindResource
Return a Resource reference that matches the argument. The format of the
argument is "<project_slug>.<resource_slug>" */
func (cfg *Config) FindResource(id string) *Resource {
	parts := strings.Split(id, ".")
	if len(parts) != 2 {
		return nil
	}
	projectSlug := parts[0]
	resourceSlug := parts[1]

	for i := range cfg.Local.Resources {
		// range returns copies: https://stackoverflow.com/q/20185511
		resource := &cfg.Local.Resources[i]
		if resource.ProjectSlug == projectSlug &&
			resource.ResourceSlug == resourceSlug {
			return resource
		}
	}
	return nil
}

/*
FindResourcesByProject
Returns a list of all the resources matching the given projectSlug
*/
func (cfg *Config) FindResourcesByProject(projectSlug string) []*Resource {
	var resources []*Resource
	for i := range cfg.Local.Resources {
		// range returns copies: https://stackoverflow.com/q/20185511
		resource := &cfg.Local.Resources[i]
		if resource.ProjectSlug == projectSlug {
			resources = append(resources, resource)
		}
	}

	return resources
}

/*
RemoveResource
Removes a resource from the Local Resources by creating a new list and
replacing the existing list
*/
func (cfg *Config) RemoveResource(r Resource) {
	cfgResources := []Resource{}
	for _, resource := range cfg.Local.Resources {
		if resource.ProjectSlug == r.ProjectSlug &&
			resource.ResourceSlug == r.ResourceSlug {
			continue
		}
		cfgResources = append(cfgResources, resource)

	}

	cfg.Local.Resources = cfgResources
}

/*
AddResource
Adds a resource to the Local.Resources list
*/
func (cfg *Config) AddResource(resource Resource) {
	cfg.Local.Resources = append(cfg.Local.Resources, resource)
}
