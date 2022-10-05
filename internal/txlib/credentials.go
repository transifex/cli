package txlib

import (
	"errors"
	"fmt"

	"github.com/transifex/cli/internal/txlib/config"
)

/*
GetHostAndToken
Function for getting the *final* API server hostname and token from a
combination of environment variables, flags, config files and/or user input.

- 'cfg' is a 'config.Config' object that has already been loaded either based
  on the default configuration paths or ones that have been supplied by the
  user.

- 'hostname' is an override for the hostname to used that the user has maybe
  provided either as a flag or an environment variable.

- 'token' is an override for the API token to be used that the user has maybe
  provided either as a flag or an environment variable.

The logic for retrieving the final hostname and token is:

1. If the hostname flag/env variable is provided, use it as a section *key* in
   the root configuration file. For example, if the user provides 'aaa' and the
   root configuration file looks like this:

       [aaa]
       rest_hostname = bbb

   Then, the returned hostname will be 'bbb'.

   If a matching host isn't found, then the returned hostname will be the
   provided value.

2. If the user didn't provide a hostname, try to find the "active host" based
   on both the local and root configuration. For example, if the local
   configuration looks like this:

       [main]
       host = ccc

   And the root configuration looks like this:

       [aaa]
       rest_hostname = bbb

       [ccc]
       rest_hostname = ddd

       [eee]
       rest_hostname = fff

   Then the "active host" will be the second one and the returned hostname will
   be 'ddd'.

   If an active host cannot be found, then 'https://rest.api.transifex.com'
   will be returned.

3. If a token was provided by the user, simply return it.

4. If a token wasn't provided, retrieve the token from either the "matching
   host" (see step 1) or the "active host" (see step 2). If a "matching" or
   "active" host wasn't found during the resolution of the hostname, the
   program will ask the user to provide a token. After the token is provided,
   it will be saved in the root configuration using the appropriate section key
   and hostname that were already retrieved.
*/
func GetHostAndToken(
	cfg *config.Config, hostname, token string,
) (string, string, error) {
	var restHostname string
	var selectedHost *config.Host
	if hostname != "" {
		// User provided hostname, see if there is a host in the root
		// configuration that matches
		host := cfg.FindHost(hostname)
		if host != nil {
			// Found
			selectedHost = host
			restHostname = host.RestHostname
		} else {
			restHostname = hostname
		}
	} else {
		// User did not provide hostname, Lets see if we can find one based on
		// the active host
		activeHost := cfg.GetActiveHost()
		if activeHost != nil {
			selectedHost = activeHost
			hostname = activeHost.Name
			restHostname = activeHost.RestHostname
		} else {
			// Fall back to defaults
			hostname = "https://www.transifex.com"
			restHostname = "https://rest.api.transifex.com"
		}
	}

	if token == "" {
		// User did not provide token
		if selectedHost != nil {
			// If a host was found in the root configuration during the search
			// for the hostname
			token = selectedHost.Token
		} else {
			fmt.Println("API token not found. Please provide it and it will " +
				"be saved in '~/.transifexrc'.")
			fmt.Println("If you don't have an API token, you can generate " +
				"one in https://www.transifex.com/user/settings/api/")
			fmt.Print("> ")
			_, err := fmt.Scanln(&token)
			if err != nil {
				return "", "", err
			}

			if cfg.Root == nil {
				rootConfigPath, err := config.GetRootPath()
				if err != nil {
					return "", "", err
				}
				cfg.Root = &config.RootConfig{
					Path: rootConfigPath,
				}
			}
			cfg.Root.Hosts = append(cfg.Root.Hosts, config.Host{
				Name:         hostname,
				RestHostname: restHostname,
				Token:        token,
			})
			err = cfg.Save()
			if err != nil {
				return "", "", err
			}
		}
	}
	if restHostname == "" || token == "" {
		return "", "", errors.New(
			"could not find a Transifex API host and/or TOKEN, please inspect your " +
				".transifexrc and .tx/config files",
		)
	}
	return restHostname, token, nil
}
