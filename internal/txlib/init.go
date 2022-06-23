package txlib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/transifex/cli/internal/txlib/config"
)

func InitCommand() error {
	// Create based on OS
	configFolder := ".tx"
	configFolder = filepath.Join("./", configFolder)
	configName := filepath.Join(configFolder, "config")

	fmt.Println()
	// In case config is in place ask the users if they want to rewrite it
	// This will result to all contents be overridden
	// If the answer is "no" we need to cancel everything
	if _, err := os.Stat(configName); !os.IsNotExist(err) {
		fmt.Println("It seems that this project is already initialized in " +
			"this folder.")
		prompt := promptui.Prompt{
			Label:     "Do you want to delete it and reinit the project",
			IsConfirm: true,
		}

		_, err := prompt.Run()

		if err != nil {
			fmt.Println("Init was cancelled!")
			return nil
		}
	}

	// Create the .tx folder in a given path
	// In case something goes wrong abort and return error
	if _, err := os.Stat(configFolder); os.IsNotExist(err) {
		err := os.Mkdir(configFolder, 0755)
		if err != nil {
			return fmt.Errorf("we couldn't create a .tx folder: %w", err)
		}
	}

	// Try to create the config file
	_, err := os.Create(configName)
	if err != nil {
		return fmt.Errorf(
			"we couldn't create a CONFIG file inside .tx directory: %w", err)
	}

	// Add the required permissions to the file
	err = os.Chmod(configName, 0755)
	if err != nil {
		return fmt.Errorf("we couldn't change permissions for .tx file: %w",
			err)
	}

	cfg := config.LocalConfig{
		Path: configName,
		Host: "https://www.transifex.com",
	}

	err = cfg.Save()

	if err != nil {
		return fmt.Errorf("we could not add data to config: %w", err)
	}

	// Everything is great! Continue!
	green := color.New(color.FgGreen).SprintFunc()
	msg := green(fmt.Sprintf("Successful creation of '%s' file", configName))

	fmt.Println(msg)
	return nil
}
