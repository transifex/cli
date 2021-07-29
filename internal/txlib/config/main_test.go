package config

import (
	"testing"

	"github.com/transifex/cli/pkg/assert"
)

func TestGetActiveHost(t *testing.T) {
	cfg := Config{
		Root: &RootConfig{
			Hosts: []Host{
				{Name: "aaa", RestHostname: "AAA"},
				{Name: "bbb", RestHostname: "BBB"},
			},
		},
		Local: &LocalConfig{
			Host: "aaa",
		},
	}

	activeHost := cfg.GetActiveHost()
	if activeHost != &cfg.Root.Hosts[0] {
		t.Errorf("Found wrong host '%s', expected '{aaa AAA}'", activeHost)
	}
}

func TestFind(t *testing.T) {
	cfg := Config{
		Local: &LocalConfig{
			Resources: []Resource{
				{ProjectSlug: "aaa", ResourceSlug: "bbb"},
				{ProjectSlug: "ccc", ResourceSlug: "ddd"},
			},
		},
	}

	resource := cfg.FindResource("aaa.bbb")
	if resource != &cfg.Local.Resources[0] {
		t.Errorf(
			"Got wrong resource %s, expected %s",
			*resource,
			cfg.Local.Resources[0],
		)
	}

	resource = cfg.FindResource("ccc.ddd")
	if resource != &cfg.Local.Resources[1] {
		t.Errorf(
			"Got wrong resource %s, expected %s",
			*resource,
			cfg.Local.Resources[0],
		)
	}

	resource = cfg.FindResource("something else")
	if resource != nil {
		t.Errorf("Got wrong resource %s, expected nil", *resource)
	}
}

func TestFindResourcesByProject(t *testing.T) {
	cfg := Config{
		Local: &LocalConfig{
			Resources: []Resource{
				{ProjectSlug: "aaa", ResourceSlug: "bbb"},
				{ProjectSlug: "aaa", ResourceSlug: "ddd"},
				{ProjectSlug: "bbb", ResourceSlug: "ccc"},
			},
		},
	}

	resources := cfg.FindResourcesByProject("aaa")
	if resources[0] != &cfg.Local.Resources[0] {
		t.Errorf(
			"Got wrong resource %s, expected %s",
			*resources[0],
			cfg.Local.Resources[0],
		)
	}
	if resources[1] != &cfg.Local.Resources[1] {
		t.Errorf(
			"Got wrong resource %s, expected %s",
			*resources[1],
			cfg.Local.Resources[1],
		)
	}

	resources = cfg.FindResourcesByProject("something else")
	if resources != nil {
		t.Error("Got wrong resource %, expected nil")
	}
}

func TestRemoveResource(t *testing.T) {
	cfg := Config{
		Local: &LocalConfig{
			Resources: []Resource{
				{ProjectSlug: "aaa", ResourceSlug: "bbb"},
				{ProjectSlug: "aaa", ResourceSlug: "ddd"},
				{ProjectSlug: "bbb", ResourceSlug: "ccc"},
			},
		},
	}

	cfg.RemoveResource(cfg.Local.Resources[1])
	if cfg.Local.Resources[1].ResourceSlug == "ddd" {
		t.Errorf(
			"This resource should have been removed",
		)
	}
	assert.Equal(t, len(cfg.Local.Resources), 2)

}
