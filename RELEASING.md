## Releasing

We currently use [GoReleaser](https://github.com/goreleaser/goreleaser) to create binaries for Transifex CLI.

To release a new version:

* Create a new PR from `devel` and merge to `master` branch.
* Create a [new release](https://github.com/transifex/cli/releases/new) with `master` as target with the version number - eg `v0.0.1` - you want to release.
* A Github Action will run with GoReleaser that will create the new binaries which you can find in the [Releases](https://github.com/transifex/cli/releases) page.

Note: Configuration for GoReleaser is in `.goreleaser.yml` file.
