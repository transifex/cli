package config

import (
	"bytes"
	"testing"
)

func TestLoadExampleRootConfig(t *testing.T) {
	path := "../../../examples/exampleconf/.transifexrc"
	rootCfg, err := loadRootConfigFromPath(path)
	if err != nil {
		t.Error(err)
	}

	expected := RootConfig{
		Path: path,
		Hosts: []Host{{
			Name:         "https://app.transifex.com",
			ApiHostname:  "https://api.transifex.com",
			Hostname:     "https://app.transifex.com",
			Username:     "__username_or_api__",
			Password:     "__password_or_api_token__",
			RestHostname: "https://rest.api.transifex.com",
			Token:        "__api_token__",
		}},
	}

	if !rootConfigsEqual(rootCfg, &expected) {
		t.Errorf(
			"Root config is wrong; got %s, expected %s",
			rootCfg,
			expected,
		)
	}
}

func TestSaveAndLoadRootConfig(t *testing.T) {
	expected := RootConfig{
		Hosts: []Host{
			{
				Name:         "My Name",
				ApiHostname:  "My API Hostname",
				Hostname:     "My Hostname",
				Username:     "My Username",
				Password:     "My Password",
				RestHostname: "My RestHostname",
				Token:        "My Token",
			},
		},
	}

	var buffer bytes.Buffer
	err := expected.saveToWriter(&buffer)
	if err != nil {
		t.Error(err)
	}

	newRootCfg, err := loadRootConfigFromBytes(buffer.Bytes())
	if err != nil {
		t.Error(err)
	}

	if !rootConfigsEqual(&expected, newRootCfg) {
		t.Errorf(
			"Root config is wrong; got %s, expected %s",
			newRootCfg,
			expected,
		)
	}
}
