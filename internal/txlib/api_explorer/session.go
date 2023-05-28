package api_explorer

import (
	"encoding/json"
	"errors"
	"os"
	"os/user"
	"path/filepath"
)

func save(key, value string) error {
	sessionPath, err := getSessionPath()
	if err != nil {
		return err
	}
	var body []byte
	if _, err := os.Stat(sessionPath); err == nil {
		body, err = os.ReadFile(sessionPath)
		if err != nil {
			return err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		body = []byte("{}")
	} else {
		return err
	}
	var data map[string]string
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}
	data[key] = value
	body, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(sessionPath, body, 0644)
	if err != nil {
		return err
	}
	return nil
}

func load(key string) (string, error) {
	sessionPath, err := getSessionPath()
	if err != nil {
		return "", err
	}
	_, err = os.Stat(sessionPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	body, err := os.ReadFile(sessionPath)
	if err != nil {
		return "", err
	}
	var data map[string]string
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}
	value, exists := data[key]
	if !exists {
		return "", nil
	}
	return value, nil
}

func clear(key string) error {
	sessionPath, err := getSessionPath()
	if err != nil {
		return err
	}
	_, err = os.Stat(sessionPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	body, err := os.ReadFile(sessionPath)
	if err != nil {
		return err
	}
	var data map[string]string
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}
	delete(data, key)
	body, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(sessionPath, body, 0644)
	if err != nil {
		return err
	}
	return nil
}

func getSessionPath() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			usr, err := user.Current()
			if err != nil {
				return "", err
			}
			homeDir = usr.HomeDir
		}
		base = filepath.Join(homeDir, ".local", "state")
	}
	err := os.MkdirAll(filepath.Join(base, "tx"), 0755)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "tx", "api_explorer_session.json"), nil
}
