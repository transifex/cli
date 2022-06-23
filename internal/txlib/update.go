package txlib

import (
	"fmt"
	"os"

	"github.com/blang/semver"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

type UpdateCommandArguments struct {
	Version       string
	NoInteractive bool
	Check         bool
	Debug         bool
}

func UpdateCommand(arguments UpdateCommandArguments) error {
	if arguments.Debug {
		selfupdate.EnableLog()
	}
	// Gets the version from txlib
	version := arguments.Version

	current, err := semver.Parse(version)
	if err != nil {
		return err
	}

	latest, _, err := selfupdate.DetectLatest("transifex/cli")
	if err != nil {
		return err
	}
	if arguments.Check {
		if current.GE(latest.Version) {
			fmt.Println("Congratulations, you are up to date with v", version)
		} else {
			fmt.Printf(
				"There is a new latest release for you"+
					" v%s -> v%s", current, latest.Version.String(),
			)
			fmt.Println()
			fmt.Println(
				"Use `tx update` or `tx update --no-interactive` " +
					"command to update to the latest version.")
			fmt.Println("If you want to download and install it manually, " +
				"you can get the asset from")
			fmt.Println(latest.AssetURL)
		}
	} else {
		if current.GE(latest.Version) {
			fmt.Println("Congratulations, you are up to date with v", version)
		} else {
			fmt.Printf(
				"There is a new latest release for you v"+
					" v%s -> v%s", current, latest.Version.String(),
			)
			// Show prompt if there is no no-interactive flag
			if !arguments.NoInteractive {
				prompt := promptui.Prompt{
					Label:     "Do you want to update",
					IsConfirm: true,
				}

				_, err := prompt.Run()

				if err != nil {
					fmt.Println("Update Cancelled")
					return nil
				}
			}

			exe, err := os.Executable()
			if err != nil {
				fmt.Println("Could not locate executable path")
				return err
			}

			msg := fmt.Sprintf("# Updating to v%s", latest.Version)
			fmt.Println(msg)
			if err != nil {
				return err
			}
			// Update executable
			if err := selfupdate.UpdateTo(latest.AssetURL, exe); err != nil {
				return err
			}
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Printf(green(
				"Successfully updated to version v%s", latest.Version))

		}

	}
	return nil
}
