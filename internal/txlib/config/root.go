package config

import (
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

type RootConfig struct {
	Hosts []Host
	Path  string
}

type Host struct {
	Name         string
	ApiHostname  string
	Hostname     string
	Username     string
	Password     string
	RestHostname string
	Token        string
}

func loadRootConfig() (*RootConfig, error) {
	rootPath, err := GetRootPath()
	if err != nil {
		return nil, nil
	}
	return loadRootConfigFromPath(rootPath)
}

func loadRootConfigFromPath(path string) (*RootConfig, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &RootConfig{Path: path}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rootCfg, err := loadRootConfigFromBytes(data)
	if err != nil {
		return nil, err
	}
	rootCfg.Path = path
	return rootCfg, nil
}

func loadRootConfigFromBytes(data []byte) (*RootConfig, error) {
	cfg, err := ini.Load(data)
	if err != nil {
		return nil, err
	}

	var result RootConfig

	for _, section := range cfg.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}
		host := Host{
			Name:         section.Name(),
			ApiHostname:  section.Key("api_hostname").String(),
			Hostname:     section.Key("hostname").String(),
			Username:     section.Key("username").String(),
			Password:     section.Key("password").String(),
			RestHostname: section.Key("rest_hostname").String(),
			Token:        section.Key("token").String(),
		}
		result.Hosts = append(result.Hosts, host)
	}

	result.sortHosts()

	return &result, nil
}

func (rootCfg *RootConfig) sortHosts() {
	sort.Slice(rootCfg.Hosts, func(i, j int) bool {
		left := rootCfg.Hosts[i].Name
		right := rootCfg.Hosts[j].Name
		return strings.Compare(left, right) == -1
	})
}

func (rootCfg *RootConfig) save() error {
	return rootCfg.saveToPath()
}

func (rootCfg *RootConfig) saveToPath() error {
	file, err := os.OpenFile(rootCfg.Path,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0755)
	if err != nil {
		return err
	}
	defer file.Close()
	return rootCfg.saveToWriter(file)
}

func (rootCfg *RootConfig) saveToWriter(file io.Writer) error {
	cfg := ini.Empty(ini.LoadOptions{})

	for _, host := range rootCfg.Hosts {
		section, err := cfg.NewSection(host.Name)
		if err != nil {
			return err
		}

		if host.ApiHostname != "" {
			_, err := section.NewKey("api_hostname", host.ApiHostname)
			if err != nil {
				return err
			}
		}

		if host.Hostname != "" {
			_, err := section.NewKey("hostname", host.Hostname)
			if err != nil {
				return err
			}
		}

		if host.Username != "" {
			_, err := section.NewKey("username", host.Username)
			if err != nil {
				return err
			}
		}

		if host.Password != "" {
			_, err := section.NewKey("password", host.Password)
			if err != nil {
				return err
			}
		}

		if host.RestHostname != "" {
			_, err := section.NewKey("rest_hostname", host.RestHostname)
			if err != nil {
				return err
			}
		}

		if host.Token != "" {
			_, err := section.NewKey("token", host.Token)
			if err != nil {
				return err
			}
		}
	}

	_, err := cfg.WriteTo(file)
	return err
}

func rootConfigsEqual(left, right *RootConfig) bool {
	if (left == nil && right != nil) || (left != nil && right == nil) {
		return false
	}
	if len(left.Hosts) != len(right.Hosts) {
		return false
	}

	for i := range left.Hosts {
		leftHost := left.Hosts[i]
		rightHost := right.Hosts[i]

		if leftHost.Name != rightHost.Name {
			return false
		}
		if leftHost.ApiHostname != rightHost.ApiHostname {
			return false
		}
		if leftHost.Hostname != rightHost.Hostname {
			return false
		}
		if leftHost.Username != rightHost.Username {
			return false
		}
		if leftHost.Password != rightHost.Password {
			return false
		}
		if leftHost.RestHostname != rightHost.RestHostname {
			return false
		}
		if leftHost.Token != rightHost.Token {
			return false
		}
	}
	return true
}

func GetRootPath() (string, error) {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		homeDir = usr.HomeDir
	}
	return filepath.Join(homeDir, ".transifexrc"), nil
}
