# Transifex's cli application (name pending...)

This cli app was created to automate localization processes.

## Development

To begin development on this repository you must first clone it localy:

`git clone git@github.com:transifex/cli.git`


### Building from source

To build the app run `make build` and a cli binary compatible with your platform will be generated in the `bin/` directory.

Run `./bin/cli --help` to list all available options.


### Example configuration

Example configuration files can be found in the `exampleconf/` directory.

1. In the `.tx/` directory you will find example project configuation. To use it run `cp -r exampleconf/.tx ./` to copy it and update the copy.

1. The `locale` directory contains an example source file  and some translation files to be used with the example `.tx/config` file. To use it run `cp -r exampleconf/locale ./`.

1. The `.transiferc` is the root configuration file. To use it run `cp exampleconf/.transifexrc $HOME` and update it.
